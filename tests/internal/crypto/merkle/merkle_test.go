package merkle_test

import (
	"testing"

	"github.com/nivschuman/VotingBlockchain/internal/crypto/hash"
	"github.com/nivschuman/VotingBlockchain/internal/crypto/merkle"
	"github.com/stretchr/testify/assert"
)

// Custom type to implement the Hashable interface for testing
type TestType struct {
	data string
}

func (t TestType) GetHash() []byte {
	return hash.HashString(t.data)
}

// Test hashPair function
func TestHashPair(t *testing.T) {
	left := []byte("left")
	right := []byte("right")

	// Hash the pair of "left" and "right"
	result := merkle.HashPair(left, right)

	// Expected result is the hash of the concatenated "leftright"
	expected := hash.HashString("leftright")

	// Assert the result matches the expected value
	assert.Equal(t, expected, result)
}

// Test calculateMerkleRoot function
func TestCalculateMerkleRoot(t *testing.T) {
	hashes := [][]byte{
		hash.HashString("a"),
		hash.HashString("b"),
		hash.HashString("c"),
		hash.HashString("d"),
	}

	// Calculate the Merkle root from the hashes
	merkleRoot := merkle.CalculateMerkleRoot(hashes)

	// Expected Merkle root after calculation (we can pre-calculate this value or use the same logic)
	expectedMerkleRoot := hash.HashString("ab")
	expectedMerkleRoot = hash.HashBytes(append(expectedMerkleRoot, hash.HashString("cd")...))

	// Assert the result matches the expected Merkle root
	assert.Equal(t, expectedMerkleRoot, merkleRoot)
}

// Test transferToHash function
func TestTransferToHash(t *testing.T) {
	// Create a slice of Hashable objects
	items := []hash.Hashable{
		TestType{"a"},
		TestType{"b"},
		TestType{"c"},
		TestType{"d"},
	}

	// Calculate the Merkle root from the slice of Hashable objects
	merkleRoot := merkle.TransferToHash(items)

	// Expected Merkle root after calculation (similar to above)
	expectedMerkleRoot := hash.HashString("ab")
	expectedMerkleRoot = hash.HashBytes(append(expectedMerkleRoot, hash.HashString("cd")...))

	// Assert the result matches the expected Merkle root
	assert.Equal(t, expectedMerkleRoot, merkleRoot)
}
