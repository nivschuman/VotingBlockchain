package hash

import (
	"crypto/sha256"
	"encoding/hex"
)

func HashStringAsBytes(data string) []byte {
	hash := sha256.Sum256([]byte(data))
	return hash[:]
}

func HashStringAsString(data string) string {
	bytes := HashStringAsBytes(data)
	return hex.EncodeToString(bytes[:])
}

func HashBytesAsBytes(data []byte) []byte {
	bytes := sha256.Sum256(data)
	return bytes[:]
}
