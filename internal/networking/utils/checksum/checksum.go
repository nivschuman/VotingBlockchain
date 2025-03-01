package checksum

import (
	"encoding/binary"

	"github.com/nivschuman/VotingBlockchain/internal/crypto/hash"
)

func CalculateChecksum(payload []byte) uint32 {
	hashedPayload := hash.HashBytes(payload)
	return binary.BigEndian.Uint32(hashedPayload)
}

func ValidateChecksum(payload []byte, providedChecksum uint32) bool {
	calculatedChecksum := CalculateChecksum(payload)
	return calculatedChecksum == providedChecksum
}
