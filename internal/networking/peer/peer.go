package networking_peer

import (
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

	Reader *connection.Reader
	Sender *connection.Sender

	ReadChannel      chan models.Message
	SendChannel      chan models.Message
	BroadcastChannel <-chan models.Message

	StopChannel chan bool

	LastMessageTime time.Time

	HandshakeDetails *HandshakeDetails
	PeerDetails      *PeerDetails
}

func NewPeer(conn net.Conn, broadcastChannel <-chan models.Message, initializer bool) *Peer {
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

	return &Peer{
		Conn:             conn,
		Reader:           reader,
		Sender:           sender,
		ReadChannel:      readChannel,
		SendChannel:      sendChannel,
		BroadcastChannel: broadcastChannel,
		StopChannel:      stopChannel,
		HandshakeDetails: handshakeDetails,
	}
}

func (peer *Peer) StartPeer() {
	go peer.ReadMessages()
	go peer.SendMessages()
}

func (peer *Peer) ReadMessages() {
	for {
		select {
		case <-peer.StopChannel:
			close(peer.ReadChannel)
			return
		default:
			message, err := peer.Reader.ReadMessage(peer.Conn)

			if err == io.EOF || err == io.ErrClosedPipe {
				close(peer.ReadChannel)
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

			peer.LastMessageTime = time.Now()
			peer.ReadChannel <- *message
		}
	}
}

func (peer *Peer) SendMessages() {
	for {
		select {
		case <-peer.StopChannel:
			return
		case message := <-peer.SendChannel:
			err := peer.Sender.SendMessage(peer.Conn, &message)

			if err == io.EOF || err == io.ErrClosedPipe {
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
		case <-peer.StopChannel:
			return
		case message := <-peer.ReadChannel:
			peer.processMessage(&message)
		}
	}
}

func (peer *Peer) Disconnect() {
	close(peer.StopChannel)

	if peer.Conn != nil {
		err := peer.Conn.Close()
		if err != nil {
			log.Printf("Error closing connection for peer %s: %v", peer.Conn.RemoteAddr().String(), err)
		}
	}
}

func (peer *Peer) processMessage(message *models.Message) {

}
