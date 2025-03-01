package networking_peer

import (
	"bytes"

	models "github.com/nivschuman/VotingBlockchain/internal/networking/models"
)

type HandshakeState int

const (
	SendVersion HandshakeState = iota
	ReceiveVersion
	SendVerack
	ReceiveVerack
	Completed
	Failed
)

func (peer *Peer) DoHandShake() {
	for {
		switch peer.HandshakeState {
		case SendVersion:
			peer.sendVersion()
		case ReceiveVersion:
			peer.receiveVersion()
		case ReceiveVerack:
			peer.receiveVerAck()
		case SendVerack:
			peer.sendVerAck()
		case Completed:
			go peer.ProcessMessages()
			return
		case Failed:
			peer.Disconnect()
			return
		}
	}
}

func (peer *Peer) CompletedHandshake() bool {
	return peer.HandshakeState == Completed
}

func (peer *Peer) sendVersion() {
	//TBD send version to send channel

	if peer.Initializer {
		peer.HandshakeState = ReceiveVersion
		return
	}

	peer.HandshakeState = ReceiveVerack
}

func (peer *Peer) sendVerAck() {
	//TBD send verack to send channel

	if peer.Initializer {
		peer.HandshakeState = ReceiveVerack
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

	if peer.Initializer {
		peer.HandshakeState = SendVerack
		return
	}

	peer.Version = *models.VersionFromBytes(message.Payload)
	peer.HandshakeState = ReceiveVerack
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
