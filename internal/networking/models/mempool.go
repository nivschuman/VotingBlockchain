package networking_models

func NewMemPoolMessage() *Message {
	return NewMessage(CommandMemPool, []byte{})
}
