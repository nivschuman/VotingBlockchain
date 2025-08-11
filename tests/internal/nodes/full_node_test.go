package nodes_test

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/nivschuman/VotingBlockchain/internal/mining"
	data_models "github.com/nivschuman/VotingBlockchain/internal/models"
	connection "github.com/nivschuman/VotingBlockchain/internal/networking/connection"
	models "github.com/nivschuman/VotingBlockchain/internal/networking/models"
	"github.com/nivschuman/VotingBlockchain/internal/networking/network"
	nodes "github.com/nivschuman/VotingBlockchain/internal/nodes"
	inits "github.com/nivschuman/VotingBlockchain/tests/init"
	networking_mocks "github.com/nivschuman/VotingBlockchain/tests/internal/networking/mocks"
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

	err = inits.TestTransactionRepository.InsertIfNotExists(tx1)
	if err != nil {
		t.Fatalf("failed to create insert tx1: %v", err)
	}

	err = inits.TestTransactionRepository.InsertIfNotExists(tx2)
	if err != nil {
		t.Fatalf("failed to create insert tx2: %v", err)
	}

	fullNode := newFullNode()
	fullNode.Start()
	t.Cleanup(func() {
		fullNode.Stop()
	})

	ip := inits.TestConfig.NetworkConfig.Ip
	port := inits.TestConfig.NetworkConfig.Port
	address := net.JoinHostPort(ip.String(), fmt.Sprint(port))

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

	fullNode := newFullNode()
	fullNode.Start()
	t.Cleanup(func() {
		fullNode.Stop()
	})

	ip := inits.TestConfig.NetworkConfig.Ip
	port := inits.TestConfig.NetworkConfig.Port
	address := net.JoinHostPort(ip.String(), fmt.Sprint(port))

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

	fullNode := newFullNode()
	fullNode.Start()
	t.Cleanup(func() {
		fullNode.Stop()
	})

	ip := inits.TestConfig.NetworkConfig.Ip
	port := inits.TestConfig.NetworkConfig.Port
	address := net.JoinHostPort(ip.String(), fmt.Sprint(port))

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

	_, err = inits.TestTransactionRepository.GetTransaction(tx.Id)
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

	fullNode := newFullNode()
	fullNode.Start()
	t.Cleanup(func() {
		fullNode.Stop()
	})

	ip := inits.TestConfig.NetworkConfig.Ip
	port := inits.TestConfig.NetworkConfig.Port
	address := net.JoinHostPort(ip.String(), fmt.Sprint(port))

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

	_, err = inits.TestBlockRepository.GetBlock(block.Header.Id)
	if err != nil {
		t.Fatalf("Failed get transaction from database: %v", err)
	}
}

func TestSendAddrToFullNode(t *testing.T) {
	inits.ResetTestDatabase()

	fullNode := newFullNode()
	fullNode.Start()
	t.Cleanup(func() {
		fullNode.Stop()
	})

	ip := inits.TestConfig.NetworkConfig.Ip
	port := inits.TestConfig.NetworkConfig.Port
	address := net.JoinHostPort(ip.String(), fmt.Sprint(port))

	conn, err := net.Dial("tcp", address)
	if err != nil {
		t.Fatalf("Failed to dial network: %v", err)
	}

	t.Cleanup(func() {
		conn.Close()
	})

	doHandshake(conn)

	addr := models.NewAddr()
	addr.AddAddress(&models.Address{Ip: net.ParseIP("192.168.1.1"), Port: 8333, NodeType: 1})
	addr.AddAddress(&models.Address{Ip: net.ParseIP("8.8.8.8"), Port: 8333, NodeType: 2})

	addrMessage, err := models.NewAddrMessage(addr)
	if err != nil {
		t.Fatalf("Failed to create addr message: %v", err)
	}

	sender := connection.NewSender()
	sender.SendMessage(conn, addrMessage)

	//wait for addresses to get processed
	time.Sleep(time.Second * 1)

	for _, address := range addr.Addresses {
		exists, err := inits.TestAddressRepository.AddressExists(address)
		if err != nil {
			t.Fatalf("Failed to check if address exists: %v", err)
		}

		if exists {
			t.Fatalf("Address %s found in repository", address)
		}
	}
}

func doHandshake(conn net.Conn) {
	version := models.Version{
		ProtocolVersion: 1,
		NodeType:        1,
		Timestamp:       time.Now().Unix(),
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

func newFullNode() *nodes.FullNode {
	ntwrk := network.NewNetworkImpl(inits.TestConfig.NetworkConfig.Ip, inits.TestConfig.NetworkConfig.Port, inits.TestAddressRepository, &inits.TestConfig.NetworkConfig, networking_mocks.MockVersionProvider)
	miner := mining.NewDisabledMiner()

	return nodes.NewFullNode(ntwrk, miner, inits.TestBlockRepository, inits.TestTransactionRepository, inits.TestConfig.GovernmentConfig.PublicKey)
}
