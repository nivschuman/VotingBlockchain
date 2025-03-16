package networking_peer_test

import (
	"net"
	"testing"
	"time"

	models "github.com/nivschuman/VotingBlockchain/internal/networking/models"
	peer "github.com/nivschuman/VotingBlockchain/internal/networking/peer"
	nonce "github.com/nivschuman/VotingBlockchain/internal/networking/utils/nonce"
	_ "github.com/nivschuman/VotingBlockchain/tests/init"
	mocks "github.com/nivschuman/VotingBlockchain/tests/internal/networking/mocks"
)

func getTestVersionMessage() *models.Message {
	version := models.Version{
		ProtocolVersion: 1,
		NodeType:        1,
		Timestamp:       time.Now().Unix(),
		Ip:              1,
		Port:            1,
		Nonce:           1,
		LastBlockHeight: 0,
	}

	return models.NewVersionMessage(&version)
}

func TestWaitForHandshake_GivenValidHandshake(t *testing.T) {
	nonce.Generator = &mocks.NonceGeneratorMock{}

	peer1Conn, peer2Conn := net.Pipe()

	go func() {
		versionMessage := getTestVersionMessage()
		peer2Conn.Write(models.MAGIC_BYTES)
		peer2Conn.Write(versionMessage.AsBytes())

		verAckMessage := models.NewVerAckMessage()
		peer2Conn.Write(models.MAGIC_BYTES)
		peer2Conn.Write(verAckMessage.AsBytes())
	}()

	p := peer.NewPeer(peer1Conn, true)
	p.StartPeer()

	err := p.WaitForHandshake(time.Second * 2)
	if err != nil {
		t.Fatalf("peer didn't complete handshake: %v", err)
	}

	p.Disconnect()

	peer1Conn.Close()
	peer2Conn.Close()
}

func TestWaitForHandshake_GivenInvalidHandshake(t *testing.T) {
	nonce.Generator = &mocks.NonceGeneratorMock{}

	peer1Conn, peer2Conn := net.Pipe()

	go func() {
		verAckMessage := models.NewVerAckMessage()
		peer2Conn.Write(models.MAGIC_BYTES)
		peer2Conn.Write(verAckMessage.AsBytes())

		peer2Conn.Write(models.MAGIC_BYTES)
		peer2Conn.Write(verAckMessage.AsBytes())
	}()

	p := peer.NewPeer(peer1Conn, true)
	p.StartPeer()

	err := p.WaitForHandshake(time.Second * 2)
	if err == nil {
		t.Fatalf("peer completed handshake")
	}

	p.Disconnect()

	peer1Conn.Close()
	peer2Conn.Close()
}

func TestWaitForHandshake_GivenIncompleteHandshake(t *testing.T) {
	nonce.Generator = &mocks.NonceGeneratorMock{}

	peer1Conn, peer2Conn := net.Pipe()

	go func() {
		versionMessage := getTestVersionMessage()
		peer2Conn.Write(models.MAGIC_BYTES)
		peer2Conn.Write(versionMessage.AsBytes())
	}()

	p := peer.NewPeer(peer1Conn, true)
	p.StartPeer()

	err := p.WaitForHandshake(time.Second * 2)
	if err == nil {
		t.Fatalf("peer completed handshake")
	}

	p.Disconnect()

	peer1Conn.Close()
	peer2Conn.Close()
}
