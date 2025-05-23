package networking_peer

import (
	"errors"
	"io"
	"log"
	"net"
	"sync"
	"time"

	config "github.com/nivschuman/VotingBlockchain/internal/config"
	connection "github.com/nivschuman/VotingBlockchain/internal/networking/connection"
	models "github.com/nivschuman/VotingBlockchain/internal/networking/models"
	checksum "github.com/nivschuman/VotingBlockchain/internal/networking/utils/checksum"
	nonce "github.com/nivschuman/VotingBlockchain/internal/networking/utils/nonce"
)

const PING_INTERVAL = 2 * time.Minute
const SEND_DATA_INTERVAL = 100 * time.Second

type Peer struct {
	Conn net.Conn

	HandshakeDetails *HandshakeDetails
	PeerDetails      *PeerDetails
	PingPongDetails  *PingPongDetails

	SendChannel    chan<- models.Message
	StopChannel    <-chan bool
	MessageHandler func(*Peer, *models.Message)

	Remove       bool
	Disconnected bool

	InventoryToSendMutex sync.Mutex
	InventoryToSend      *models.Inv

	reader *connection.Reader
	sender *connection.Sender

	readChannel chan models.Message
	sendChannel chan models.Message

	stopChannel    chan bool
	disconnectOnce sync.Once
}

func NewPeer(conn net.Conn, initializer bool) *Peer {
	reader := connection.NewReader()
	sender := connection.NewSender()

	readChannel := make(chan models.Message, 10)
	sendChannel := make(chan models.Message, 10)

	stopChannel := make(chan bool)

	handshakeDetails := &HandshakeDetails{
		HandshakeState: initialHandshakeState(initializer),
		Initializer:    initializer,
		Error:          nil,
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
		PingPongDetails:  pingPongDetails,
		Disconnected:     false,
		Remove:           false,
		SendChannel:      sendChannel,
		StopChannel:      stopChannel,
		InventoryToSend:  models.NewInv(),
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

func (peer *Peer) StartPeer() {
	go peer.ReadMessages()
	go peer.SendMessages()
}

func (peer *Peer) StartProcessing(messageHandler func(peer *Peer, message *models.Message)) {
	peer.MessageHandler = messageHandler
	go peer.ProcessMessages()
	go peer.SendData()
}

func (peer *Peer) ReadMessages() {
	for {
		select {
		case <-peer.StopChannel:
			close(peer.readChannel)
			return
		default:
			message, err := peer.reader.ReadMessage(peer.Conn)

			if err == io.EOF || err == io.ErrClosedPipe || errors.Is(err, net.ErrClosed) {
				close(peer.readChannel)
				peer.Disconnect()
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

func (peer *Peer) SendMessages() {
	for {
		select {
		case <-peer.StopChannel:
			return
		case message := <-peer.sendChannel:
			err := peer.sender.SendMessage(peer.Conn, &message)

			if err == io.EOF || err == io.ErrClosedPipe || errors.Is(err, net.ErrClosed) {
				peer.Disconnect()
				return
			}

			if err != nil {
				log.Printf("Failed to send message to peer %s: %v", peer.Conn.RemoteAddr().String(), err)
			}
		}
	}
}

func (peer *Peer) SendData() {
	sendDataInterval := time.Duration(config.GlobalConfig.NetworkConfig.SendDataInterval) * time.Second
	tickerData := time.NewTicker(sendDataInterval)
	defer tickerData.Stop()

	pingInterval := time.Duration(config.GlobalConfig.NetworkConfig.SendDataInterval) * time.Second
	tickerPing := time.NewTicker(pingInterval)
	defer tickerPing.Stop()

	for {
		select {
		case <-peer.StopChannel:
			return
		case <-tickerPing.C:
			peer.MaybeSendPing()
		case <-tickerData.C:
			peer.SendInventory()
		}
	}
}

func (peer *Peer) ProcessMessages() {
	for {
		select {
		case <-peer.StopChannel:
			return
		case message := <-peer.readChannel:
			peer.MessageHandler(peer, &message)
		}
	}
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
}

func (peer *Peer) MaybeSendPing() {
	if peer.PingPongDetails.Nonce != 0 {
		return
	}

	n, err := nonce.Generator.GenerateNonce()

	if err != nil {
		return
	}

	peer.PingPongDetails.PingTime = time.Now()
	peer.PingPongDetails.Nonce = n

	select {
	case <-peer.StopChannel:
		return
	case peer.SendChannel <- *models.NewMessage(models.CommandPing, nonce.NonceToBytes(n)):
	}
}

func (peer *Peer) SendInventory() {
	peer.InventoryToSendMutex.Lock()
	defer peer.InventoryToSendMutex.Unlock()

	if peer.InventoryToSend.Count == 0 {
		return
	}

	invMessage, err := models.NewInvMessage(peer.InventoryToSend)

	if err != nil {
		return
	}

	select {
	case <-peer.StopChannel:
		return
	case peer.SendChannel <- *invMessage:
		peer.InventoryToSend.Clear()
	}
}
