package networking_connection_test

import (
	"bytes"
	"log"
	"net"
	"testing"

	connection "github.com/nivschuman/VotingBlockchain/internal/networking/connection"
	models "github.com/nivschuman/VotingBlockchain/internal/networking/models"
	_ "github.com/nivschuman/VotingBlockchain/tests/init"
)

func TestSender_SendMessage(t *testing.T) {
	testMessage := getTestMessage()

	peer1Conn, peer2Conn := net.Pipe()

	go func() {
		sender := connection.NewSender()
		err := sender.SendMessage(peer1Conn, &testMessage)

		if err != nil {
			log.Printf("failed to send message: %v", err)
		}

		peer2Conn.Close()
	}()

	var receivedMagicBytes [4]byte
	amount, err := peer2Conn.Read(receivedMagicBytes[:])

	if err != nil {
		t.Fatalf("error in reading sent magic bytes: %v", err)
	}

	if amount != 4 && !bytes.Equal(receivedMagicBytes[:], models.MAGIC_BYTES) {
		t.Fatalf("received bad magic bytes %x", receivedMagicBytes)
	}

	length := len(testMessage.AsBytes())
	receivedMessage := make([]byte, length)
	totalRead := 0

	for totalRead < length {
		n, err := peer2Conn.Read(receivedMessage[totalRead:])

		if err != nil {
			t.Fatalf("error in reading message: %v", err)
		}

		totalRead += n
	}

	if !bytes.Equal(receivedMessage, testMessage.AsBytes()) {
		t.Fatalf("received message not equal to test message %x", receivedMessage)
	}

	peer1Conn.Close()
}
