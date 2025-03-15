package nonce

import (
	"crypto/rand"
	"encoding/binary"
)

var Generator NonceGenerator = &NonceGeneratorImpl{}

type NonceGenerator interface {
	GenerateNonce() (uint64, error)
}

type NonceGeneratorImpl struct {
}

func (*NonceGeneratorImpl) GenerateNonce() (uint64, error) {
	var nonce uint64
	err := binary.Read(rand.Reader, binary.BigEndian, &nonce)
	if err != nil {
		return 0, err
	}
	return nonce, nil
}
