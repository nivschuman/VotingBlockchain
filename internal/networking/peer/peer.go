package networking_peer

import (
	"errors"
	"io"
	"log"
	"net"
	"sync"
	"time"

	connection "github.com/nivschuman/VotingBlockchain/internal/networking/connection"
	models "github.com/nivschuman/VotingBlockchain/internal/networking/models"
	checksum "github.com/nivschuman/VotingBlockchain/internal/networking/utils/checksum"
	ip_utils "github.com/nivschuman/VotingBlockchain/internal/networking/utils/ip"
	nonce "github.com/nivschuman/VotingBlockchain/internal/networking/utils/nonce"
	structures "github.com/nivschuman/VotingBlockchain/internal/structures"
)

const PING_INTERVAL = 2 * time.Minute
const SEND_DATA_INTERVAL = 100 * time.Second

type CommandHandler func(peer *Peer, message *models.Message)

type PeerEventHandler func(eventData PeerEventData)
type PeerEventData struct {
	Event string
	Peer  *Peer
}

type PeerConfig struct {
	SendDataInterval time.Duration
	PingInterval     time.Duration
	GetAddrInterval  time.Duration
}

type Peer struct {
	Conn net.Conn

	HandshakeDetails *HandshakeDetails
	PingPongDetails  *PingPongDetails
	PeerDetails      *PeerDetails

	Remove       bool
	Disconnected bool

	InventoryToSendMutex sync.Mutex
	InventoryToSend      *models.Inv

	SentGetAddrMutex sync.Mutex
	SentGetAddr      bool

	Address  *models.Address
	LastSeen *time.Time

	myVersion  models.VersionProvider
	peerConfig PeerConfig

	commandHandlersMutex sync.Mutex
	commandHandlers      *structures.BytesMap[[]CommandHandler]

	reader *connection.Reader
	sender *connection.Sender

	readChannel chan models.Message
	sendChannel chan models.Message

	stopChannel    chan bool
	disconnectOnce sync.Once
	wg             sync.WaitGroup
}

func NewPeer(conn net.Conn, initializer bool, peerConfig PeerConfig, myVersion models.VersionProvider) *Peer {
	reader := connection.NewReader()
	sender := connection.NewSender()

	readChannel := make(chan models.Message, 10)
	sendChannel := make(chan models.Message, 10)

	stopChannel := make(chan bool)

	handshakeDetails := &HandshakeDetails{
		HandshakeState: initialHandshakeState(initializer),
		Initializer:    initializer,
	}

	pingPongDetails := &PingPongDetails{
		Nonce:    0,
		PingTime: time.Now(),
		PongTime: time.Now(),
	}

	return &Peer{
		Conn:             conn,
		HandshakeDetails: handshakeDetails,
		PeerDetails:      nil,
		Address:          &models.Address{},
		LastSeen:         nil,
		PingPongDetails:  pingPongDetails,
		Disconnected:     false,
		Remove:           false,
		InventoryToSend:  models.NewInv(),
		SentGetAddr:      false,
		myVersion:        myVersion,
		peerConfig:       peerConfig,
		commandHandlers:  structures.NewBytesMap[[]CommandHandler](),
		reader:           reader,
		sender:           sender,
		readChannel:      readChannel,
		sendChannel:      sendChannel,
		stopChannel:      stopChannel,
	}
}

func (peer *Peer) String() string {
	return peer.Conn.RemoteAddr().String()
}

func (peer *Peer) Start() {
	peer.wg.Add(2)
	go peer.readMessages()
	go peer.sendMessages()
}

func (peer *Peer) StartProcessing() {
	peer.wg.Add(2)
	go peer.processMessages()
	go peer.sendData()
}

func (peer *Peer) AddCommandHandler(command [12]byte, commandHandler CommandHandler) {
	peer.commandHandlersMutex.Lock()
	defer peer.commandHandlersMutex.Unlock()

	handlers, exists := peer.commandHandlers.Get(command[:])

	if exists {
		peer.commandHandlers.Put(command[:], append(handlers, commandHandler))
		return
	}

	peer.commandHandlers.Put(command[:], []CommandHandler{commandHandler})
}

func (peer *Peer) SetPeerAddress() error {
	ip, port, err := ip_utils.ConnToIpAndPort(peer.Conn)
	if err != nil {
		return err
	}

	peer.Address.Ip = ip
	peer.Address.Port = port
	return nil
}

func (peer *Peer) Disconnect() {
	peer.disconnectOnce.Do(func() {
		close(peer.stopChannel)

		if peer.Conn != nil {
			err := peer.Conn.Close()
			if err != nil {
				log.Printf("Error closing connection for peer %s: %v", peer.Conn.RemoteAddr().String(), err)
			}
		}

		peer.Disconnected = true
	})
	peer.wg.Wait()
}

func (peer *Peer) SendMessage(message *models.Message) bool {
	select {
	case <-peer.stopChannel:
		return false
	case peer.sendChannel <- *message:
		return true
	}
}

func (peer *Peer) readMessages() {
	defer peer.wg.Done()
	for {
		select {
		case <-peer.stopChannel:
			close(peer.readChannel)
			return
		default:
			message, err := peer.reader.ReadMessage(peer.Conn)

			if err == io.EOF || err == io.ErrClosedPipe || errors.Is(err, net.ErrClosed) {
				close(peer.readChannel)
				peer.Disconnected = true
				return
			}

			if err != nil {
				log.Printf("Error when receiving message from peer %s: %v", peer.Conn.RemoteAddr().String(), err)
				continue
			}

			validChecksum := checksum.ValidateChecksum(message.Payload, message.MessageHeader.CheckSum)

			if !validChecksum {
				log.Printf("Invalid checksum when receiving message from peer %s", peer.Conn.RemoteAddr().String())
				continue
			}

			peer.readChannel <- *message
		}
	}
}

func (peer *Peer) sendMessages() {
	defer peer.wg.Done()
	for {
		select {
		case <-peer.stopChannel:
			return
		case message := <-peer.sendChannel:
			err := peer.sender.SendMessage(peer.Conn, &message)

			if err == io.EOF || err == io.ErrClosedPipe || errors.Is(err, net.ErrClosed) {
				peer.Disconnected = true
				return
			}

			if err != nil {
				log.Printf("Failed to send message to peer %s: %v", peer.Conn.RemoteAddr().String(), err)
			}
		}
	}
}

func (peer *Peer) sendData() {
	defer peer.wg.Done()

	tickerData := time.NewTicker(peer.peerConfig.SendDataInterval)
	defer tickerData.Stop()

	tickerPing := time.NewTicker(peer.peerConfig.PingInterval)
	defer tickerPing.Stop()

	tickerGetAddr := time.NewTicker(peer.peerConfig.GetAddrInterval)
	defer tickerGetAddr.Stop()

	for {
		select {
		case <-peer.stopChannel:
			return
		case <-tickerPing.C:
			peer.maybeSendPing()
		case <-tickerData.C:
			peer.sendInventory()
		case <-tickerGetAddr.C:
			peer.maybeSendGetAddr()
		}
	}
}

func (peer *Peer) processMessages() {
	defer peer.wg.Done()
	for {
		select {
		case <-peer.stopChannel:
			return
		case message, ok := <-peer.readChannel:
			if !ok {
				return
			}

			peer.commandHandlersMutex.Lock()
			handlers := peer.commandHandlers.GetOrDefault(message.MessageHeader.Command[:], make([]CommandHandler, 0))
			for _, handler := range handlers {
				handler(peer, &message)
			}
			peer.commandHandlersMutex.Unlock()
		}
	}
}

func (peer *Peer) maybeSendPing() {
	if peer.PingPongDetails.Nonce != 0 {
		return
	}

	n, err := nonce.Generator.GenerateNonce()

	if err != nil {
		return
	}

	peer.PingPongDetails.PingTime = time.Now()
	peer.PingPongDetails.Nonce = n

	peer.SendMessage(models.NewMessage(models.CommandPing, nonce.NonceToBytes(n)))
}

func (peer *Peer) maybeSendGetAddr() {
	peer.SentGetAddrMutex.Lock()
	defer peer.SentGetAddrMutex.Unlock()

	if peer.SentGetAddr {
		return
	}

	sent := peer.SendMessage(models.NewGetAddrMessage())
	if sent {
		peer.SentGetAddr = true
	}
}

func (peer *Peer) sendInventory() {
	peer.InventoryToSendMutex.Lock()
	defer peer.InventoryToSendMutex.Unlock()

	if peer.InventoryToSend.Count == 0 {
		return
	}

	invMessage, err := models.NewInvMessage(peer.InventoryToSend)

	if err != nil {
		return
	}

	sent := peer.SendMessage(invMessage)
	if sent {
		peer.InventoryToSend.Clear()
	}
}
