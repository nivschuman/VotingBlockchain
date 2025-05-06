package networking_models

func NewPongMessage(nonce []byte) *Message {
	return NewMessage(CommandPong, nonce)
}
