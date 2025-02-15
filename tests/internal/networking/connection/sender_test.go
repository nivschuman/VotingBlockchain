package networking_connection_test

import (
	"bytes"
	"testing"

	connection "github.com/nivschuman/VotingBlockchain/internal/networking/connection"
	models "github.com/nivschuman/VotingBlockchain/internal/networking/models"
	mocks "github.com/nivschuman/VotingBlockchain/tests/internal/networking/mocks"
)

func TestSender_SendMessage(t *testing.T) {
	testMessage := getTestMessage()

	conn := mocks.NewConnMock()

	sender := connection.NewSender()
	err := sender.SendMessage(conn, testMessage)

	if err != nil {
		t.Fatalf("error in read message: %v", err)
	}

	var receivedMagicBytes [4]byte
	amount, err := conn.Read(receivedMagicBytes[:])

	if err != nil {
		t.Fatalf("error in reading sent magic bytes: %v", err)
	}

	if amount != 4 && !bytes.Equal(receivedMagicBytes[:], models.MAGIC_BYTES) {
		t.Fatalf("received bad magic bytes %x", receivedMagicBytes)
	}

	receivedMessage := make([]byte, len(testMessage.AsBytes()))
	_, err = conn.Read(receivedMessage)

	if err != nil {
		t.Fatalf("error in reading message: %v", err)
	}

	t.Logf("hello % x\n", receivedMagicBytes)

	if !bytes.Equal(receivedMessage, testMessage.AsBytes()) {
		t.Fatalf("received message not equal to test message %x", receivedMessage)
	}
}
