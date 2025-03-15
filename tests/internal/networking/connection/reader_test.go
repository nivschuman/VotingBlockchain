package networking_connection_test

import (
	"testing"
	"time"

	connection "github.com/nivschuman/VotingBlockchain/internal/networking/connection"
	models "github.com/nivschuman/VotingBlockchain/internal/networking/models"
	_ "github.com/nivschuman/VotingBlockchain/tests/init"
	mocks "github.com/nivschuman/VotingBlockchain/tests/internal/networking/mocks"
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

	conn := mocks.NewConnMock()
	conn.WriteToLocal(models.MAGIC_BYTES)
	conn.WriteToLocal(testMessage.MessageHeader.AsBytes())
	conn.WriteToLocal(testMessage.Payload)

	reader := connection.NewReader()
	message, err := reader.ReadMessageWithTimeout(conn, time.Second*5)

	if err != nil {
		t.Fatalf("error in read message: %v", err)
	}

	if message == nil {
		t.Fatalf("message is nil")
	}

	if !message.Equals(&testMessage) {
		t.Fatalf("messages aren't the same")
	}
}

func TestReader_ReadMultipleMessages(t *testing.T) {
	testMessage := getTestMessage()

	conn := mocks.NewConnMock()
	conn.WriteToLocal(models.MAGIC_BYTES)
	conn.WriteToLocal(testMessage.MessageHeader.AsBytes())
	conn.WriteToLocal(testMessage.Payload)

	conn.WriteToLocal(models.MAGIC_BYTES)
	conn.WriteToLocal(testMessage.MessageHeader.AsBytes())
	conn.WriteToLocal(testMessage.Payload)

	reader := connection.NewReader()
	message, err := reader.ReadMessageWithTimeout(conn, time.Second*5)

	if err != nil {
		t.Fatalf("error in read message: %v", err)
	}

	if message == nil {
		t.Fatalf("message is nil")
	}

	if !message.Equals(&testMessage) {
		t.Fatalf("messages aren't the same")
	}

	message, err = reader.ReadMessageWithTimeout(conn, time.Second*5)

	if err != nil {
		t.Fatalf("error in read message: %v", err)
	}

	if message == nil {
		t.Fatalf("message is nil")
	}

	if !message.Equals(&testMessage) {
		t.Fatalf("messages aren't the same")
	}
}
