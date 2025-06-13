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

func TestTargetToNBits(t *testing.T) {
	target := new(big.Int)
	target.SetString("00000000FFFF0000000000000000000000000000000000000000000000000000", 16)
	if got := difficulty.TargetToNBits(target); got != 0x1d00ffff {
		t.Fatalf("Test 1 failed: got 0x%08x, want 0x1d00ffff", got)
	}

	target.SetString("01", 16)
	if got := difficulty.TargetToNBits(target); got != 0x01010000 {
		t.Fatalf("Test 2 failed: got 0x%08x, want 0x01010000", got)
	}

	target.SetString("00", 16)
	if got := difficulty.TargetToNBits(target); got != 0x00000000 {
		t.Fatalf("Test 3 failed: got 0x%08x, want 0x00000000", got)
	}

	target.SetString("0001234500000000000000000000000000000000000000000000000000000000", 16)
	if got := difficulty.TargetToNBits(target); got != 0x1f012345 {
		t.Fatalf("Test 4 failed: got 0x%08x, want 0x1d012345", got)
	}

	target.SetString("0080000000000000000000000000000000000000000000000000000000000000", 16)
	if got := difficulty.TargetToNBits(target); got != 0x20008000 {
		t.Fatalf("Test 5 failed: got 0x%08x, want 0x1e800000", got)
	}
}
