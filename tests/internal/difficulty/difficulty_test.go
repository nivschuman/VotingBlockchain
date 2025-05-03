package difficulty_test

import (
	"encoding/hex"
	"log"
	"math/big"
	"testing"

	difficulty "github.com/nivschuman/VotingBlockchain/internal/difficulty"
)

func TestGetTargetFromNBits(t *testing.T) {
	nBits := uint32(0x1d00ffff)
	target := difficulty.GetTargetFromNBits(nBits)

	log.Println(target.String())

	expectedTarget := new(big.Int)
	expectedTarget.SetString("00000000ffff0000000000000000000000000000000000000000000000000000", 16)

	if target.Cmp(expectedTarget) != 0 {
		t.Fatalf("Expected target %s, got %s", expectedTarget.String(), target.String())
	}
}

func TestIsHashBelowTarget_WhenHashIsBelowTarget(t *testing.T) {
	target := new(big.Int)
	target.SetString("00000000ffff0000000000000000000000000000000000000000000000000000", 16)

	hashHex := "000000000019d6689c085ae165831e934ff763ae46a2a6c172b3f1b60a8ce26f"
	hash, err := hex.DecodeString(hashHex)

	if err != nil {
		t.Fatalf("Failed to decode hash string to bytes: %v", err)
	}

	if !difficulty.IsHashBelowTarget(hash, target) {
		t.Fatalf("Hash was determined to not be below target")
	}
}
