package networking_peer

import (
	"errors"
	"io"
	"log"
	"net"
	"time"

	connection "github.com/nivschuman/VotingBlockchain/internal/networking/connection"
	models "github.com/nivschuman/VotingBlockchain/internal/networking/models"
	checksum "github.com/nivschuman/VotingBlockchain/internal/networking/utils/checksum"
)

type Peer struct {
	Conn net.Conn

	HandshakeDetails *HandshakeDetails
	PeerDetails      *PeerDetails
	PingPongDetails  *PingPongDetails

	SendChannel    chan<- models.Message
	MessageHandler func(*Peer, *models.Message)

	Remove       bool
	Disconnected bool

	reader *connection.Reader
	sender *connection.Sender

	readChannel chan models.Message
	sendChannel chan models.Message
	stopChannel chan bool
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
		reader:           reader,
		sender:           sender,
		readChannel:      readChannel,
		sendChannel:      sendChannel,
		stopChannel:      stopChannel,
	}
}

func (peer *Peer) StartPeer() {
	go peer.ReadMessages()
	go peer.SendMessages()
}

func (peer *Peer) StartProcessing(messageHandler func(peer *Peer, message *models.Message)) {
	peer.MessageHandler = messageHandler
	go peer.ProcessMessages()
}

func (peer *Peer) ReadMessages() {
	for {
		select {
		case <-peer.stopChannel:
			close(peer.readChannel)
			return
		default:
			message, err := peer.reader.ReadMessage(peer.Conn)

			if err == io.EOF || err == io.ErrClosedPipe || errors.Is(err, net.ErrClosed) {
				close(peer.readChannel)
				peer.Remove = true
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
		case <-peer.stopChannel:
			return
		case message := <-peer.sendChannel:
			err := peer.sender.SendMessage(peer.Conn, &message)

			if err == io.EOF || err == io.ErrClosedPipe || errors.Is(err, net.ErrClosed) {
				peer.Remove = true
				return
			}

			if err != nil {
				log.Printf("Failed to send message to peer %s: %v", peer.Conn.RemoteAddr().String(), err)
			}
		}
	}
}

func (peer *Peer) ProcessMessages() {
	for {
		select {
		case <-peer.stopChannel:
			return
		case message := <-peer.readChannel:
			peer.MessageHandler(peer, &message)
		}
	}
}

func (peer *Peer) Disconnect() {
	close(peer.stopChannel)

	if peer.Conn != nil {
		err := peer.Conn.Close()
		if err != nil {
			log.Printf("Error closing connection for peer %s: %v", peer.Conn.RemoteAddr().String(), err)
		}
	}

	peer.Disconnected = true
}
