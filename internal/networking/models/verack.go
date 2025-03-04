package networking_models

func NewVerAckMessage() *Message {
	return NewMessage(CommandVerAck, []byte{})
}
