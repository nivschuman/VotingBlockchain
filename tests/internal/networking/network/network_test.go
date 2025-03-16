package network_test

import (
	"fmt"
	"net"
	"testing"
	"time"

	config "github.com/nivschuman/VotingBlockchain/internal/config"
	connection "github.com/nivschuman/VotingBlockchain/internal/networking/connection"
	models "github.com/nivschuman/VotingBlockchain/internal/networking/models"
	network "github.com/nivschuman/VotingBlockchain/internal/networking/network"
	nonce "github.com/nivschuman/VotingBlockchain/internal/networking/utils/nonce"
	_ "github.com/nivschuman/VotingBlockchain/tests/init"
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

func TestSendPingToNetwork(t *testing.T) {
	network := network.NewNetwork()
	network.StartNetwork()

	ip := config.GlobalConfig.Ip
	port := config.GlobalConfig.Port
	address := net.JoinHostPort(ip, fmt.Sprint(port))

	conn, err := net.Dial("tcp", address)
	if err != nil {
		t.Fatalf("Failed to dial network: %v", err)
	}

	doHandshake(conn)

	n, err := nonce.Generator.GenerateNonce()
	if err != nil {
		t.Fatalf("Failed to generate nonce: %v", err)
	}

	reader := connection.NewReader()
	sender := connection.NewSender()

	pingMessage := models.NewMessage(models.CommandPing, nonce.NonceToBytes(n))
	sender.SendMessage(conn, pingMessage)

	pongMessage, err := reader.ReadMessage(conn)

	if err != nil {
		t.Fatalf("Failed to read pong message: %v", err)
	}

	if nonce.NonceFromBytes(pongMessage.Payload) != n {
		t.Fatalf("Received pong with wrong nonce")
	}

	network.StopNetwork()
	conn.Close()
}

func doHandshake(conn net.Conn) {
	reader := connection.NewReader()
	sender := connection.NewSender()

	versionMessage := getTestVersionMessage()
	sender.SendMessage(conn, versionMessage)

	reader.ReadMessage(conn)

	verAckMessage := models.NewVerAckMessage()
	sender.SendMessage(conn, verAckMessage)

	reader.ReadMessage(conn)
}
