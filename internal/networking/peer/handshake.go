package networking_peer

import (
	"bytes"
	"fmt"
	"log"
	"time"

	models "github.com/nivschuman/VotingBlockchain/internal/networking/models"
	nonce "github.com/nivschuman/VotingBlockchain/internal/networking/utils/nonce"
)

type HandshakeState int

const (
	SendVersion HandshakeState = iota
	ReceiveVersion
	SendVerack
	ReceiveVerAck
	Completed
	Failed
)

type HandshakeDetails struct {
	HandshakeState HandshakeState
	Initializer    bool //true if we need to initialize handshake with peer
	Nonce          uint64
	Error          error
}

func (peer *Peer) WaitForHandshake(timeout time.Duration) error {
	go peer.DoHandShake()

	timeoutChan := time.After(timeout)
	done := make(chan bool, 1)

	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-timeoutChan:
				return
			case <-ticker.C:
				if peer.CompletedHandshake() || peer.FailedHandshake() {
					done <- true
					return
				}
			}
		}
	}()

	select {
	case <-done:
		if !peer.CompletedHandshake() {
			return fmt.Errorf("handshake didn't complete, state: %s, error: %v", peer.HandshakeDetails.HandshakeState.AsString(), peer.HandshakeDetails.Error)
		}

		return nil
	case <-timeoutChan:
		return fmt.Errorf("timeout reached while waiting for handshake completion, state: %s", peer.HandshakeDetails.HandshakeState.AsString())
	}
}

func (peer *Peer) DoHandShake() {
	for {
		switch peer.HandshakeDetails.HandshakeState {
		case SendVersion:
			peer.sendVersion()
		case ReceiveVersion:
			peer.receiveVersion()
		case ReceiveVerAck:
			peer.receiveVerAck()
		case SendVerack:
			peer.sendVerAck()
		case Completed:
			return
		case Failed:
			return
		}
	}
}

func (peer *Peer) CompletedHandshake() bool {
	return peer.HandshakeDetails.HandshakeState == Completed
}

func (peer *Peer) FailedHandshake() bool {
	return peer.HandshakeDetails.HandshakeState == Failed
}

func (peer *Peer) sendVersion() {
	myVersion, err := models.MyVersion()

	if err != nil {
		peer.HandshakeDetails.Error = err
		peer.HandshakeDetails.HandshakeState = Failed
		return
	}

	if peer.HandshakeDetails.Initializer {
		n, err := nonce.Generator.GenerateNonce()

		if err != nil {
			peer.HandshakeDetails.Error = err
			peer.HandshakeDetails.HandshakeState = Failed
			return
		}

		peer.HandshakeDetails.HandshakeState = ReceiveVersion
		myVersion.Nonce = n
		peer.HandshakeDetails.Nonce = n
	} else {
		peer.HandshakeDetails.HandshakeState = ReceiveVerAck
		myVersion.Nonce = peer.HandshakeDetails.Nonce
	}

	message := models.NewVersionMessage(myVersion)

	select {
	case <-peer.StopChannel:
		peer.HandshakeDetails.HandshakeState = Failed
		return
	case peer.SendChannel <- *message:
	}
}

func (peer *Peer) sendVerAck() {
	verAckMessage := models.NewVerAckMessage()

	select {
	case <-peer.StopChannel:
		peer.HandshakeDetails.HandshakeState = Failed
		return
	case peer.SendChannel <- *verAckMessage:
	}

	if peer.HandshakeDetails.Initializer {
		peer.HandshakeDetails.HandshakeState = ReceiveVerAck
		return
	}

	peer.HandshakeDetails.HandshakeState = Completed
}

func (peer *Peer) receiveVersion() {
	message, ok := <-peer.readChannel

	if !ok {
		peer.HandshakeDetails.Error = fmt.Errorf("peer %s read channel closed while waiting for version message", peer.Conn.RemoteAddr().String())
		peer.HandshakeDetails.HandshakeState = Failed
		return
	}

	if !bytes.Equal(message.MessageHeader.Command[:], models.CommandVersion[:]) {
		peer.HandshakeDetails.Error = fmt.Errorf("peer %s expected version message, received: %s", peer.Conn.RemoteAddr().String(), message.MessageHeader.Command)
		peer.HandshakeDetails.HandshakeState = Failed
		return
	}

	version := models.VersionFromBytes(message.Payload)
	peer.PeerDetails = NewPeerDetailsFromVersion(version)
	if peer.HandshakeDetails.Initializer {
		if version.Nonce != peer.HandshakeDetails.Nonce {
			peer.HandshakeDetails.Error = fmt.Errorf("received bad nonce")
			peer.HandshakeDetails.HandshakeState = Failed
			return
		}

		peer.HandshakeDetails.HandshakeState = SendVerack
		return
	}

	peer.HandshakeDetails.HandshakeState = SendVersion
}

func (peer *Peer) receiveVerAck() {
	message, ok := <-peer.readChannel

	if !ok {
		log.Printf("Peer %s read channel closed while waiting for verAck message", peer.Conn.RemoteAddr().String())
		peer.HandshakeDetails.HandshakeState = Failed
		return
	}

	if !bytes.Equal(message.MessageHeader.Command[:], models.CommandVerAck[:]) {
		peer.HandshakeDetails.Error = fmt.Errorf("peer %s expected verAck message, received: %s", peer.Conn.RemoteAddr().String(), message.MessageHeader.Command)

		peer.HandshakeDetails.HandshakeState = Failed
		return
	}

	if peer.HandshakeDetails.Initializer {
		peer.HandshakeDetails.HandshakeState = Completed
		return
	}

	peer.HandshakeDetails.HandshakeState = SendVerack
}

func initialHandshakeState(initializer bool) HandshakeState {
	if initializer {
		return SendVersion
	}

	return ReceiveVersion
}

func (hs HandshakeState) AsString() string {
	switch hs {
	case SendVersion:
		return "SendVersion"
	case ReceiveVersion:
		return "ReceiveVersion"
	case SendVerack:
		return "SendVerack"
	case ReceiveVerAck:
		return "ReceiveVerack"
	case Completed:
		return "Completed"
	case Failed:
		return "Failed"
	default:
		return "Unknown"
	}
}
