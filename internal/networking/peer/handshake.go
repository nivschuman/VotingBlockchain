package networking_peer

import (
	"bytes"
	"fmt"
	"time"

	models "github.com/nivschuman/VotingBlockchain/internal/networking/models"
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

func (peer *Peer) WaitForHandshake(timeout time.Duration) error {
	go peer.DoHandShake()

	timeoutChan := time.After(timeout)
	done := make(chan bool)

	go func() {
		for {
			if peer.CompletedHandshake() || peer.FailedHandshake() {
				done <- true
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()

	select {
	case <-done:
		if !peer.CompletedHandshake() {
			return fmt.Errorf("handshake didn't complete, state: %s", peer.HandshakeState.AsString())
		}

		return nil
	case <-timeoutChan:
		return fmt.Errorf("timeout reached while waiting for handshake completion, state: %s", peer.HandshakeState.AsString())
	}
}

func (peer *Peer) DoHandShake() {
	for {
		switch peer.HandshakeState {
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
	return peer.HandshakeState == Completed
}

func (peer *Peer) FailedHandshake() bool {
	return peer.HandshakeState == Failed
}

func (peer *Peer) sendVersion() {
	//TBD send version to send channel

	if peer.Initializer {
		peer.HandshakeState = ReceiveVersion
		return
	}

	peer.HandshakeState = ReceiveVerAck
}

func (peer *Peer) sendVerAck() {
	verAckMessage := models.NewVerAckMessage()
	peer.SendChannel <- *verAckMessage

	if peer.Initializer {
		peer.HandshakeState = ReceiveVerAck
		return
	}

	peer.HandshakeState = Completed
}

func (peer *Peer) receiveVersion() {
	message := <-peer.ReadChannel

	if !bytes.Equal(message.MessageHeader.Command[:], models.CommandVersion[:]) {
		peer.HandshakeState = Failed
		return
	}

	peer.Version = models.VersionFromBytes(message.Payload)

	if peer.Initializer {
		peer.HandshakeState = SendVerack
		return
	}

	peer.HandshakeState = ReceiveVerAck
}

func (peer *Peer) receiveVerAck() {
	message := <-peer.ReadChannel

	if !bytes.Equal(message.MessageHeader.Command[:], models.CommandVerAck[:]) {
		peer.HandshakeState = Failed
		return
	}

	if peer.Initializer {
		peer.HandshakeState = Completed
		return
	}

	peer.HandshakeState = SendVerack
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
