package network_test

import (
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	config "github.com/nivschuman/VotingBlockchain/internal/config"
	"github.com/nivschuman/VotingBlockchain/internal/database/repositories"
	connection "github.com/nivschuman/VotingBlockchain/internal/networking/connection"
	models "github.com/nivschuman/VotingBlockchain/internal/networking/models"
	network "github.com/nivschuman/VotingBlockchain/internal/networking/network"
	nonce "github.com/nivschuman/VotingBlockchain/internal/networking/utils/nonce"
	inits "github.com/nivschuman/VotingBlockchain/tests/init"
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
	network := network.NewNetwork()
	network.StartNetwork()

	ip := config.GlobalConfig.NetworkConfig.Ip
	port := config.GlobalConfig.NetworkConfig.Port
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

func TestSendMemPoolToNetwork(t *testing.T) {
	inits.ResetTestDatabase()
	govKeyPair, err := inits.GenerateTestGovernmentKeyPair()

	if err != nil {
		t.Fatalf("Failed to generate government key pair: %v", err)
	}

	tx1, _, err := inits.CreateTestTransaction(govKeyPair)
	if err != nil {
		t.Fatalf("failed to create test tx1: %v", err)
	}

	tx2, _, err := inits.CreateTestTransaction(govKeyPair)
	if err != nil {
		t.Fatalf("failed to create test tx2: %v", err)
	}

	err = repositories.GlobalTransactionRepository.InsertIfNotExists(tx1)
	if err != nil {
		t.Fatalf("failed to create insert tx1: %v", err)
	}

	err = repositories.GlobalTransactionRepository.InsertIfNotExists(tx2)
	if err != nil {
		t.Fatalf("failed to create insert tx2: %v", err)
	}

	network := network.NewNetwork()
	network.StartNetwork()

	ip := config.GlobalConfig.NetworkConfig.Ip
	port := config.GlobalConfig.NetworkConfig.Port
	address := net.JoinHostPort(ip, fmt.Sprint(port))

	conn, err := net.Dial("tcp", address)
	if err != nil {
		t.Fatalf("Failed to dial network: %v", err)
	}

	doHandshake(conn)

	reader := connection.NewReader()
	sender := connection.NewSender()

	memPoolMessage := models.NewMemPoolMessage()
	sender.SendMessage(conn, memPoolMessage)

	invMessage, err := reader.ReadMessage(conn)

	if err != nil {
		t.Fatalf("Failed to read inv message: %v", err)
	}

	inv, err := models.InvFromBytes(invMessage.Payload)

	if err != nil {
		t.Fatalf("Failed to read parse inv message: %v", err)
	}

	if !inv.Contains(models.MSG_TX, tx1.Id) || !inv.Contains(models.MSG_TX, tx2.Id) {
		t.Fatalf("Inv returned doesn't contain transactions")
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
