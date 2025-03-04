package networking_connection

import (
	"fmt"
	"net"

	models "github.com/nivschuman/VotingBlockchain/internal/networking/models"
)

type Sender struct {
}

func NewSender() *Sender {
	return &Sender{}
}

func (sender *Sender) SendMessage(conn net.Conn, Message *models.Message) error {
	err := sender.sendMagicBytes(conn)

	if err != nil {
		return err
	}

	err = sender.sendHeader(conn, &Message.MessageHeader)

	if err != nil {
		return err
	}

	err = sender.sendPayload(conn, Message.Payload)
	return err
}

func (sender *Sender) sendMagicBytes(conn net.Conn) error {
	return sendBytes(conn, models.MAGIC_BYTES)
}

func (sender *Sender) sendHeader(conn net.Conn, messageHeader *models.MessageHeader) error {
	return sendBytes(conn, messageHeader.AsBytes())
}

func (sender *Sender) sendPayload(conn net.Conn, payload []byte) error {
	chunkSize := 1024
	totalBytesSent := 0
	length := len(payload)

	for totalBytesSent < len(payload) {
		end := totalBytesSent + chunkSize

		if end > length {
			end = length
		}

		chunk := payload[totalBytesSent:end]
		err := sendBytes(conn, chunk)
		if err != nil {
			return err
		}

		totalBytesSent += len(chunk)
	}

	return nil
}

func sendBytes(conn net.Conn, bytesToSend []byte) error {
	totalBytesWritten := 0

	for totalBytesWritten < len(bytesToSend) {
		n, err := conn.Write(bytesToSend[totalBytesWritten:])
		if err != nil {
			return fmt.Errorf("failed to write bytes: %v", err)
		}

		totalBytesWritten += n
	}

	return nil
}
