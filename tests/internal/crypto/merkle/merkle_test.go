package merkle_test

import (
	"testing"

	"github.com/nivschuman/VotingBlockchain/internal/crypto/hash"
	"github.com/nivschuman/VotingBlockchain/internal/crypto/merkle"
	"github.com/stretchr/testify/assert"
)

type TestType struct {
	data string
}

func (t TestType) GetHash() []byte {
	return hash.HashString(t.data)
}

func TestCalculateMerkleRoot(t *testing.T) {
	items := []hash.Hashable{
		TestType{"a"},
		TestType{"b"},
		TestType{"c"},
		TestType{"d"},
	}
	merkleRoot := merkle.CalculateMerkleRoot(items)
	hashA := hash.HashString("a")
	hashB := hash.HashString("b")
	abHash := merkle.HashPair(hashA, hashB)

	hashC := hash.HashString("c")
	hashD := hash.HashString("d")
	cdHash := merkle.HashPair(hashC, hashD)

	expectedMerkleRoot := merkle.HashPair(abHash, cdHash)
	assert.Equal(t, expectedMerkleRoot, merkleRoot)
}

func TestTransferToHash(t *testing.T) {
	items := []hash.Hashable{
		TestType{"a"},
		TestType{"b"},
		TestType{"c"},
		TestType{"d"},
	}
	hashes := merkle.TransferToHash(items)
	expectedHashes := [][]byte{
		hash.HashString("a"),
		hash.HashString("b"),
		hash.HashString("c"),
		hash.HashString("d"),
	}
	assert.Equal(t, expectedHashes, hashes)
}
