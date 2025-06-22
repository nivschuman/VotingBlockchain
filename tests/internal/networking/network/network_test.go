package network_test

import (
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	config "github.com/nivschuman/VotingBlockchain/internal/config"
	connection "github.com/nivschuman/VotingBlockchain/internal/networking/connection"
	models "github.com/nivschuman/VotingBlockchain/internal/networking/models"
	network "github.com/nivschuman/VotingBlockchain/internal/networking/network"
	nonce "github.com/nivschuman/VotingBlockchain/internal/networking/utils/nonce"
	inits "github.com/nivschuman/VotingBlockchain/tests/init"
)

func TestMain(m *testing.M) {
	// === BEFORE ALL TESTS ===
	inits.SetupTests()
	inits.SetupTestsDatabase()

	// Run the tests
	code := m.Run()

	// === AFTER ALL TESTS ===
	inits.CloseTestDatabase()

	// Exit with the right code
	os.Exit(code)
}

func TestSendPingToNetwork(t *testing.T) {
	ip := config.GlobalConfig.NetworkConfig.Ip
	port := config.GlobalConfig.NetworkConfig.Port
	network := network.NewNetwork(ip, port)
	network.Start()

	t.Cleanup(func() {
		network.Stop()
	})

	address := net.JoinHostPort(ip, fmt.Sprint(port))
	conn, err := net.Dial("tcp", address)
	if err != nil {
		t.Fatalf("Failed to dial network: %v", err)
	}

	t.Cleanup(func() {
		conn.Close()
	})

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
}

func doHandshake(conn net.Conn) {
	version := models.Version{
		ProtocolVersion: 1,
		NodeType:        1,
		Timestamp:       time.Now().Unix(),
		Ip:              1,
		Port:            1,
		Nonce:           1,
		LastBlockHeight: 0,
	}

	reader := connection.NewReader()
	sender := connection.NewSender()

	versionMessage := models.NewVersionMessage(&version)
	sender.SendMessage(conn, versionMessage)

	reader.ReadMessage(conn)

	verAckMessage := models.NewVerAckMessage()
	sender.SendMessage(conn, verAckMessage)

	reader.ReadMessage(conn)
}
