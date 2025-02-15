package networking_connection

import (
	"log"
	"net"
	"time"

	models "github.com/nivschuman/VotingBlockchain/internal/networking/models"
)

type Peer struct {
	Conn net.Conn

	Reader *Reader
	Sender *Sender

	ReadChannel      chan models.Message
	SendChannel      chan models.Message
	BroadcastChannel <-chan models.Message

	StopChannel chan bool

	CompletedHandshake bool
	LastMessageTime    time.Time
}

func NewPeer(conn net.Conn, readBufferSize, sendBufferSize int, broadcastChannel <-chan models.Message) *Peer {
	reader := NewReader()
	sender := NewSender()

	readChannel := make(chan models.Message, readBufferSize)
	sendChannel := make(chan models.Message, sendBufferSize)

	stopChannel := make(chan bool)

	return &Peer{
		Conn:               conn,
		Reader:             reader,
		Sender:             sender,
		ReadChannel:        readChannel,
		SendChannel:        sendChannel,
		BroadcastChannel:   broadcastChannel,
		StopChannel:        stopChannel,
		CompletedHandshake: false,
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
			if err == nil {
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
			if err := peer.Sender.SendMessage(peer.Conn, message); err != nil {
				peer.SendChannel <- message
				time.Sleep(time.Millisecond * 500)
			}
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
