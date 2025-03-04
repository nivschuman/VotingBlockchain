package networking_models

import (
	"bytes"
	"encoding/binary"
	"fmt"

	chck "github.com/nivschuman/VotingBlockchain/internal/networking/utils/checksum"
)

var MAGIC_BYTES = []byte{0xD9, 0xB4, 0xBE, 0xF9}

type MessageHeader struct {
	Command  [12]byte
	Length   uint32
	CheckSum uint32
}

type Message struct {
	MessageHeader MessageHeader
	Payload       []byte
}

func MessageHeaderFromBytes(bytes []byte) (MessageHeader, error) {
	if len(bytes) < 20 {
		return MessageHeader{}, fmt.Errorf("data too short to extract MessageHeader")
	}

	var command [12]byte
	copy(command[:], bytes[:12])

	length := binary.BigEndian.Uint32(bytes[12:16])
	checksum := binary.BigEndian.Uint32(bytes[16:20])

	header := MessageHeader{
		Command:  command,
		Length:   length,
		CheckSum: checksum,
	}

	return header, nil
}

func (messageHeader *MessageHeader) AsBytes() []byte {
	buf := new(bytes.Buffer)

	buf.Write(messageHeader.Command[:])
	binary.Write(buf, binary.BigEndian, uint32(messageHeader.Length))
	binary.Write(buf, binary.BigEndian, uint32(messageHeader.CheckSum))

	return buf.Bytes()
}

func (m1 *Message) Equals(m2 *Message) bool {
	if m1.MessageHeader != m2.MessageHeader {
		return false
	}

	if len(m1.Payload) != len(m2.Payload) {
		return false
	}

	for i := range m1.Payload {
		if m1.Payload[i] != m2.Payload[i] {
			return false
		}
	}

	return true
}

func (message *Message) AsBytes() []byte {
	buf := new(bytes.Buffer)

	buf.Write(message.MessageHeader.AsBytes())
	buf.Write(message.Payload)

	return buf.Bytes()
}

func NewMessage(command [12]byte, payload []byte) *Message {
	messageHeader := MessageHeader{
		Command:  command,
		Length:   uint32(len(payload)),
		CheckSum: chck.CalculateChecksum(payload),
	}

	return &Message{
		MessageHeader: messageHeader,
		Payload:       payload,
	}
}
