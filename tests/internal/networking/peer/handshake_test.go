package networking_peer_test

import (
	"testing"
	"time"

	models "github.com/nivschuman/VotingBlockchain/internal/networking/models"
	peer "github.com/nivschuman/VotingBlockchain/internal/networking/peer"
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
	conn := mocks.NewConnMock()
	p := peer.NewPeer(conn, nil, true)

	versionMessage := getTestVersionMessage()
	conn.WriteToLocal(models.MAGIC_BYTES)
	conn.WriteToLocal(versionMessage.AsBytes())

	verAckMessage := models.NewVerAckMessage()
	conn.WriteToLocal(models.MAGIC_BYTES)
	conn.WriteToLocal(verAckMessage.AsBytes())

	p.StartPeer()

	err := p.WaitForHandshake(time.Second * 2)

	if err != nil {
		t.Fatalf("peer didn't complete handshake: %v", err)
	}

	p.Disconnect()
}

func TestWaitForHandshake_GivenInvalidHandshake(t *testing.T) {
	conn := mocks.NewConnMock()
	p := peer.NewPeer(conn, nil, true)

	verAckMessage := models.NewVerAckMessage()
	conn.WriteToLocal(models.MAGIC_BYTES)
	conn.WriteToLocal(verAckMessage.AsBytes())

	conn.WriteToLocal(models.MAGIC_BYTES)
	conn.WriteToLocal(verAckMessage.AsBytes())

	p.StartPeer()

	err := p.WaitForHandshake(time.Second * 2)

	if err == nil {
		t.Fatalf("peer completed handshake")
	}

	p.Disconnect()
}

func TestWaitForHandshake_GivenIncompleteHandshake(t *testing.T) {
	conn := mocks.NewConnMock()
	p := peer.NewPeer(conn, nil, true)

	versionMessage := getTestVersionMessage()
	conn.WriteToLocal(models.MAGIC_BYTES)
	conn.WriteToLocal(versionMessage.AsBytes())

	p.StartPeer()

	err := p.WaitForHandshake(time.Second * 2)

	if err == nil {
		t.Fatalf("peer completed handshake")
	}

	p.Disconnect()
}
