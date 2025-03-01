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

	return models.NewMessage(models.CommandVersion, version.AsBytes())
}

func TestInitializerHandshakeCompletion_GivenValidHandshake(t *testing.T) {
	conn := mocks.NewConnMock()
	p := peer.NewPeer(conn, nil, true)

	p.DoHandShake()

	versionMessage := getTestVersionMessage()
	conn.Write(versionMessage.AsBytes())

}
