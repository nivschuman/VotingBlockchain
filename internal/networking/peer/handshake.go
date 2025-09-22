package networking_peer

import (
	"bytes"
	"fmt"
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
)

type HandshakeDetails struct {
	HandshakeState HandshakeState
	Initializer    bool //true if we need to initialize handshake with peer
	Nonce          uint64
}

func (peer *Peer) WaitForHandshake(timeout time.Duration) error {
	result := make(chan error, 1)
	go func() {
		result <- peer.DoHandShake()
	}()

	select {
	case err := <-result:
		return err
	case <-time.After(timeout):
		return fmt.Errorf("timeout reached while waiting for handshake completion, state: %s", peer.HandshakeDetails.HandshakeState.AsString())
	}
}

func (peer *Peer) DoHandShake() error {
	for peer.HandshakeDetails.HandshakeState != Completed {
		switch peer.HandshakeDetails.HandshakeState {
		case SendVersion:
			err := peer.sendVersion()
			if err != nil {
				return err
			}
		case ReceiveVersion:
			err := peer.receiveVersion()
			if err != nil {
				return err
			}
		case ReceiveVerAck:
			err := peer.receiveVerAck()
			if err != nil {
				return err
			}
		case SendVerack:
			err := peer.sendVerAck()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (peer *Peer) sendVersion() error {
	myVersion, err := peer.myVersion()
	if err != nil {
		return err
	}

	if peer.HandshakeDetails.Initializer {
		n, err := nonce.Generator.GenerateNonce()
		if err != nil {
			return err
		}

		peer.HandshakeDetails.HandshakeState = ReceiveVersion
		myVersion.Nonce = n
		peer.HandshakeDetails.Nonce = n
	} else {
		peer.HandshakeDetails.HandshakeState = ReceiveVerAck
		myVersion.Nonce = peer.HandshakeDetails.Nonce
	}

	message := models.NewVersionMessage(myVersion)
	sent := peer.SendMessage(message)
	if !sent {
		return fmt.Errorf("failed to send version")
	}

	return nil
}

func (peer *Peer) sendVerAck() error {
	verAckMessage := models.NewVerAckMessage()
	sent := peer.SendMessage(verAckMessage)
	if !sent {
		return fmt.Errorf("failed to send verack")
	}

	if peer.HandshakeDetails.Initializer {
		peer.HandshakeDetails.HandshakeState = ReceiveVerAck
	} else {
		peer.HandshakeDetails.HandshakeState = Completed
	}

	return nil
}

func (peer *Peer) receiveVersion() error {
	message, ok := <-peer.readChannel

	if !ok {
		return fmt.Errorf("peer %s read channel closed while waiting for version message", peer.Conn.RemoteAddr().String())
	}

	if !bytes.Equal(message.MessageHeader.Command[:], models.CommandVersion[:]) {
		return fmt.Errorf("peer %s expected version message, received: %s", peer.Conn.RemoteAddr().String(), message.MessageHeader.Command)
	}

	version := models.VersionFromBytes(message.Payload)

	peer.SetPeerDetails(version)
	peer.Address.NodeType = version.NodeType

	if peer.HandshakeDetails.Initializer {
		if version.Nonce != peer.HandshakeDetails.Nonce {
			return fmt.Errorf("received bad nonce")
		}

		peer.HandshakeDetails.HandshakeState = SendVerack
		return nil
	}

	peer.HandshakeDetails.Nonce = version.Nonce
	peer.HandshakeDetails.HandshakeState = SendVersion
	return nil
}

func (peer *Peer) receiveVerAck() error {
	message, ok := <-peer.readChannel
	if !ok {
		return fmt.Errorf("Peer %s read channel closed while waiting for verAck message", peer.Conn.RemoteAddr().String())
	}

	if !bytes.Equal(message.MessageHeader.Command[:], models.CommandVerAck[:]) {
		return fmt.Errorf("peer %s expected verAck message, received: %s", peer.Conn.RemoteAddr().String(), message.MessageHeader.Command)
	}

	if peer.HandshakeDetails.Initializer {
		peer.HandshakeDetails.HandshakeState = Completed
		return nil
	}

	peer.HandshakeDetails.HandshakeState = SendVerack
	return nil
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
	default:
		return "Unknown"
	}
}
