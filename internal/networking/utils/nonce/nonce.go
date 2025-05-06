package nonce

import (
	"crypto/rand"
	"encoding/binary"
)

var Generator NonceGenerator = &nonceGeneratorImpl{}

type NonceGenerator interface {
	GenerateNonce() (uint64, error)
}

type nonceGeneratorImpl struct {
}

func (*nonceGeneratorImpl) GenerateNonce() (uint64, error) {
	var nonce uint64
	err := binary.Read(rand.Reader, binary.BigEndian, &nonce)
	if err != nil {
		return 0, err
	}

	if nonce == 0 {
		return 1, nil
	}

	return nonce, nil
}

func NonceToBytes(nonce uint64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, nonce)
	return buf
}

func NonceFromBytes(bytes []byte) uint64 {
	return binary.BigEndian.Uint64(bytes)
}
