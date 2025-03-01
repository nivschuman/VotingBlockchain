package networking_peer

import (
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

	HandshakeState  HandshakeState
	Initializer     bool
	LastMessageTime time.Time

	Version models.Version
}

func NewPeer(conn net.Conn, broadcastChannel <-chan models.Message, initializer bool) *Peer {
	reader := connection.NewReader()
	sender := connection.NewSender()

	readChannel := make(chan models.Message, 10)
	sendChannel := make(chan models.Message, 10)

	stopChannel := make(chan bool)

	return &Peer{
		Conn:             conn,
		Reader:           reader,
		Sender:           sender,
		ReadChannel:      readChannel,
		SendChannel:      sendChannel,
		BroadcastChannel: broadcastChannel,
		StopChannel:      stopChannel,
		HandshakeState:   initialHandshakeState(initializer),
		Initializer:      initializer,
	}
}

func (peer *Peer) StartPeer() {
	go peer.ReadMessages()
	go peer.SendMessages()
	go peer.DoHandShake()
}

func (peer *Peer) ReadMessages() {
	for {
		select {
		case <-peer.StopChannel:
			close(peer.ReadChannel)
			return
		default:
			message, err := peer.Reader.ReadMessage(peer.Conn)
			validChecksum := checksum.ValidateChecksum(message.Payload, message.MessageHeader.CheckSum)
			if err == nil && validChecksum {
				peer.LastMessageTime = time.Now()
				peer.ReadChannel <- *message
			}
		}
	}
}

func (peer *Peer) SendMessages() {
	for {
		select {
		case <-peer.StopChannel:
			return
		default:
			message := <-peer.SendChannel
			if err := peer.Sender.SendMessage(peer.Conn, &message); err != nil {
				peer.SendChannel <- message
				time.Sleep(time.Millisecond * 500)
			}
		}
	}
}

func (peer *Peer) ProcessMessages() {
	for {
		select {
		case <-peer.StopChannel:
			return
		default:
			message := <-peer.ReadChannel
			peer.processMessage(&message)
		}
	}
}

func (peer *Peer) Disconnect() {
	close(peer.StopChannel)

	if peer.Conn != nil {
		err := peer.Conn.Close()
		if err != nil {
			log.Printf("Error closing connection for peer %s: %v", peer.Conn.RemoteAddr(), err)
		}
	}

	close(peer.SendChannel)
}

func (peer *Peer) processMessage(message *models.Message) {

}
