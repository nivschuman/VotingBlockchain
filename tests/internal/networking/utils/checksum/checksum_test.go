package checksum_test

import (
	"testing"

	hash "github.com/nivschuman/VotingBlockchain/internal/crypto/hash"
	pkg "github.com/nivschuman/VotingBlockchain/internal/networking/utils/checksum"
)

func TestCalculateChecksum(t *testing.T) {
	payload := []byte("test payload")

	hashedPayload := hash.HashBytes(payload)
	expectedChecksum := uint32(hashedPayload[0])<<24 | uint32(hashedPayload[1])<<16 | uint32(hashedPayload[2])<<8 | uint32(hashedPayload[3])

	checksum := pkg.CalculateChecksum(payload)
	if checksum != expectedChecksum {
		t.Errorf("CalculateChecksum() = %x, want %x", checksum, expectedChecksum)
	}
}

func TestValidateChecksum(t *testing.T) {
	payload := []byte("test payload")
	checksum := pkg.CalculateChecksum(payload)

	valid := pkg.ValidateChecksum(payload, checksum)
	if !valid {
		t.Errorf("ValidateChecksum() returned false, expected true")
	}

	invalid := pkg.ValidateChecksum(payload, checksum+1)
	if invalid {
		t.Errorf("ValidateChecksum() returned true for incorrect checksum, expected false")
	}
}
