package networking_connection_test

import (
	"net"
	"testing"
	"time"

	connection "github.com/nivschuman/VotingBlockchain/internal/networking/connection"
	models "github.com/nivschuman/VotingBlockchain/internal/networking/models"
	_ "github.com/nivschuman/VotingBlockchain/tests/init"
)

func getTestMessage() models.Message {
	command := "TestCommand"
	var commandBytes [12]byte
	copy(commandBytes[:], []byte(command))

	payload := "TestPayload"
	payloadBytes := make([]byte, len(payload))
	copy(payloadBytes[:], []byte(payload))

	messageHeader := models.MessageHeader{
		Command:  commandBytes,
		Length:   uint32(len(payloadBytes)),
		CheckSum: uint32(2823231487),
	}

	return models.Message{
		MessageHeader: messageHeader,
		Payload:       payloadBytes,
	}
}

func TestReader_ReadMessage(t *testing.T) {
	testMessage := getTestMessage()

	peer1Conn, peer2Conn := net.Pipe()

	go func() {
		peer2Conn.Write(models.MAGIC_BYTES)
		peer2Conn.Write(testMessage.MessageHeader.AsBytes())
		peer2Conn.Write(testMessage.Payload)
		peer2Conn.Close()
	}()

	reader := connection.NewReader()
	message, err := reader.ReadMessageWithTimeout(peer1Conn, time.Second*5)

	if err != nil {
		t.Fatalf("error in read message: %v", err)
	}

	if message == nil {
		t.Fatalf("message is nil")
	}

	if !message.Equals(&testMessage) {
		t.Fatalf("messages aren't the same")
	}

	peer1Conn.Close()
}

func TestReader_ReadMultipleMessages(t *testing.T) {
	testMessage := getTestMessage()

	peer1Conn, peer2Conn := net.Pipe()

	go func() {
		peer2Conn.Write(models.MAGIC_BYTES)
		peer2Conn.Write(testMessage.MessageHeader.AsBytes())
		peer2Conn.Write(testMessage.Payload)

		peer2Conn.Write(models.MAGIC_BYTES)
		peer2Conn.Write(testMessage.MessageHeader.AsBytes())
		peer2Conn.Write(testMessage.Payload)

		peer2Conn.Close()
	}()

	reader := connection.NewReader()
	message, err := reader.ReadMessageWithTimeout(peer1Conn, time.Second*5)

	if err != nil {
		t.Fatalf("error in read message: %v", err)
	}

	if message == nil {
		t.Fatalf("message is nil")
	}

	if !message.Equals(&testMessage) {
		t.Fatalf("messages aren't the same")
	}

	message, err = reader.ReadMessageWithTimeout(peer1Conn, time.Second*5)

	if err != nil {
		t.Fatalf("error in read message: %v", err)
	}

	if message == nil {
		t.Fatalf("message is nil")
	}

	if !message.Equals(&testMessage) {
		t.Fatalf("messages aren't the same")
	}

	peer1Conn.Close()
}
