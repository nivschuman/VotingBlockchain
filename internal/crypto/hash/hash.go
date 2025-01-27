package hash

import (
	"crypto/sha256"
)

type Hashable interface {
	GetHash() []byte
}

func HashString(data string) []byte {
	hash := sha256.Sum256([]byte(data))
	return hash[:]
}

func HashBytes(data []byte) []byte {
	bytes := sha256.Sum256(data)
	return bytes[:]
}
