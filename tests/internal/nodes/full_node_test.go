package nodes_test

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	config "github.com/nivschuman/VotingBlockchain/internal/config"
	repositories "github.com/nivschuman/VotingBlockchain/internal/database/repositories"
	data_models "github.com/nivschuman/VotingBlockchain/internal/models"
	connection "github.com/nivschuman/VotingBlockchain/internal/networking/connection"
	models "github.com/nivschuman/VotingBlockchain/internal/networking/models"
	nodes "github.com/nivschuman/VotingBlockchain/internal/nodes"
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

func TestSendMemPoolToFullNode(t *testing.T) {
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

	fullNode := nodes.NewFullNode(false)
	fullNode.Start()
	t.Cleanup(func() {
		fullNode.Stop()
	})

	ip := config.GlobalConfig.NetworkConfig.Ip
	port := config.GlobalConfig.NetworkConfig.Port
	address := net.JoinHostPort(ip, fmt.Sprint(port))

	conn, err := net.Dial("tcp", address)
	if err != nil {
		t.Fatalf("Failed to dial network: %v", err)
	}

	t.Cleanup(func() {
		conn.Close()
	})

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
}

func TestSendGetDataToFullNode(t *testing.T) {
	inits.ResetTestDatabase()
	_, blocks, _, err := inits.CreateTestData(5, 1)
	if err != nil {
		t.Fatalf("failed to create test data: %v", err)
	}

	getData := models.NewGetData()
	getData.AddItem(models.MSG_BLOCK, blocks[0].Header.Id)
	getData.AddItem(models.MSG_TX, blocks[0].Transactions[0].Id)

	getDataMessage, err := models.NewGetDataMessage(getData)
	if err != nil {
		t.Fatalf("Failed to create get data message: %v", err)
	}

	fullNode := nodes.NewFullNode(false)
	fullNode.Start()
	t.Cleanup(func() {
		fullNode.Stop()
	})

	ip := config.GlobalConfig.NetworkConfig.Ip
	port := config.GlobalConfig.NetworkConfig.Port
	address := net.JoinHostPort(ip, fmt.Sprint(port))

	conn, err := net.Dial("tcp", address)
	if err != nil {
		t.Fatalf("Failed to dial network: %v", err)
	}

	t.Cleanup(func() {
		conn.Close()
	})

	doHandshake(conn)

	sender := connection.NewSender()
	sender.SendMessage(conn, getDataMessage)

	reader := connection.NewReader()

	msg1, err := reader.ReadMessage(conn)
	if err != nil {
		t.Fatalf("Failed to read message: %v", err)
	}

	msg2, err := reader.ReadMessage(conn)
	if err != nil {
		t.Fatalf("Failed to read message: %v", err)
	}

	if bytes.Equal(msg1.MessageHeader.Command[:], models.CommandBlock[:]) {
		block, err := data_models.BlockFromBytes(msg1.Payload)
		if err != nil {
			t.Fatalf("Failed to parse block: %v", err)
		}

		if !bytes.Equal(block.AsBytes(), blocks[0].AsBytes()) {
			t.Fatalf("Received wrong block")
		}
	} else if bytes.Equal(msg1.MessageHeader.Command[:], models.CommandTx[:]) {
		tx, err := data_models.TransactionFromBytes(msg1.Payload)

		if err != nil {
			t.Fatalf("Failed to parse tx: %v", err)
		}

		if !bytes.Equal(tx.AsBytes(), blocks[0].Transactions[0].AsBytes()) {
			t.Fatalf("Received wrong tx")
		}
	} else {
		t.Fatalf("received bad command: %x", msg1.MessageHeader.Command[:])
	}

	if bytes.Equal(msg2.MessageHeader.Command[:], models.CommandBlock[:]) {
		block, err := data_models.BlockFromBytes(msg2.Payload)
		if err != nil {
			t.Fatalf("Failed to parse block: %v", err)
		}

		if !bytes.Equal(block.AsBytes(), blocks[0].AsBytes()) {
			t.Fatalf("Received wrong block")
		}
	} else if bytes.Equal(msg2.MessageHeader.Command[:], models.CommandTx[:]) {
		tx, err := data_models.TransactionFromBytes(msg2.Payload)
		if err != nil {
			t.Fatalf("Failed to parse tx: %v", err)
		}

		if !bytes.Equal(tx.AsBytes(), blocks[0].Transactions[0].AsBytes()) {
			t.Fatalf("Received wrong tx")
		}
	} else {
		t.Fatalf("received bad command: %x", msg2.MessageHeader.Command[:])
	}
}

func TestSendTransactionToFullNode(t *testing.T) {
	inits.ResetTestDatabase()
	govKeyPair, _, _, err := inits.CreateTestData(5, 1)
	if err != nil {
		t.Fatalf("failed to create test data: %v", err)
	}

	fullNode := nodes.NewFullNode(false)
	fullNode.Start()
	t.Cleanup(func() {
		fullNode.Stop()
	})

	ip := config.GlobalConfig.NetworkConfig.Ip
	port := config.GlobalConfig.NetworkConfig.Port
	address := net.JoinHostPort(ip, fmt.Sprint(port))

	conn, err := net.Dial("tcp", address)
	if err != nil {
		t.Fatalf("Failed to dial network: %v", err)
	}

	t.Cleanup(func() {
		conn.Close()
	})

	doHandshake(conn)

	tx, _, err := inits.CreateTestTransaction(govKeyPair)
	if err != nil {
		t.Fatalf("Failed to create test transaction: %v", err)
	}

	sender := connection.NewSender()
	sender.SendMessage(conn, models.NewMessage(models.CommandTx, tx.AsBytes()))

	//wait for transaction to get inserted
	time.Sleep(time.Second * 1)

	_, err = repositories.GlobalTransactionRepository.GetTransaction(tx.Id)
	if err != nil {
		t.Fatalf("Failed get transaction from database: %v", err)
	}
}

func TestSendBlockToFullNode(t *testing.T) {
	inits.ResetTestDatabase()
	_, blocks, _, err := inits.CreateTestData(5, 1)
	if err != nil {
		t.Fatalf("failed to create test data: %v", err)
	}

	fullNode := nodes.NewFullNode(false)
	fullNode.Start()
	t.Cleanup(func() {
		fullNode.Stop()
	})

	ip := config.GlobalConfig.NetworkConfig.Ip
	port := config.GlobalConfig.NetworkConfig.Port
	address := net.JoinHostPort(ip, fmt.Sprint(port))

	conn, err := net.Dial("tcp", address)
	if err != nil {
		t.Fatalf("Failed to dial network: %v", err)
	}

	t.Cleanup(func() {
		conn.Close()
	})

	doHandshake(conn)

	block, err := inits.CreateTestBlock(blocks[len(blocks)-1].Header.Id, make([]*data_models.Transaction, 0))
	if err != nil {
		t.Fatalf("Failed to create test block: %v", err)
	}

	sender := connection.NewSender()
	sender.SendMessage(conn, models.NewMessage(models.CommandBlock, block.AsBytes()))

	//wait for block to get inserted
	time.Sleep(time.Second * 1)

	_, err = repositories.GlobalBlockRepository.GetBlock(block.Header.Id)
	if err != nil {
		t.Fatalf("Failed get transaction from database: %v", err)
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
