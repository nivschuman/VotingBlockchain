package networking_models

func NewGetAddrMessage() *Message {
	return NewMessage(CommandGetAddr, []byte{})
}
