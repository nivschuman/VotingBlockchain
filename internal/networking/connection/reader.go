package networking_connection

import (
	"encoding/binary"
	"net"

	models "github.com/nivschuman/VotingBlockchain/internal/networking/models"
)

type Reader struct {
	HeaderBuffer  [20]byte
	PayloadBuffer []byte
}

func NewReader() *Reader {
	return &Reader{
		PayloadBuffer: make([]byte, 0),
	}
}

func (reader *Reader) ReadMessage(conn net.Conn) (*models.Message, error) {
	err := reader.readMagicBytes(conn)

	if err != nil {
		return nil, err
	}

	err = reader.readHeader(conn)

	if err != nil {
		return nil, err
	}

	err = reader.readPayload(conn)

	if err != nil {
		return nil, err
	}

	return reader.processMessage()
}

func (reader *Reader) readMagicBytes(conn net.Conn) error {
	magicBytesIndex := 0
	buf := make([]byte, 1)

	for {
		_, err := conn.Read(buf)

		if err != nil {
			return err
		}

		if buf[0] == models.MAGIC_BYTES[magicBytesIndex] {
			magicBytesIndex++

			if magicBytesIndex == len(models.MAGIC_BYTES) {
				return nil
			}
		} else {
			magicBytesIndex = 0
		}
	}
}

func (reader *Reader) readHeader(conn net.Conn) error {
	totalRead := 0

	for totalRead < len(reader.HeaderBuffer) {
		n, err := conn.Read(reader.HeaderBuffer[totalRead:])

		if err != nil {
			return err
		}

		totalRead += n
	}

	return nil
}

func (reader *Reader) readPayload(conn net.Conn) error {
	length := binary.BigEndian.Uint32(reader.HeaderBuffer[12:16])
	totalRead := uint32(0)

	for totalRead < length {
		n, err := conn.Read(reader.PayloadBuffer[totalRead:])

		if err != nil {
			return err
		}

		totalRead += uint32(n)
	}

	return nil
}

func (reader *Reader) processMessage() (*models.Message, error) {
	messageHeader, err := models.MessageHeaderFromBytes(reader.HeaderBuffer[:])

	if err != nil {
		return nil, err
	}

	message := models.Message{
		MessageHeader: messageHeader,
		Payload:       reader.PayloadBuffer,
	}

	return &message, err
}
