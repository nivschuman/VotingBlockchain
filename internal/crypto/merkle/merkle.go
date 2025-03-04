package merkle

import (
	"github.com/nivschuman/VotingBlockchain/internal/crypto/hash"
)

func HashPair(left []byte, right []byte) []byte {
	combined := append(left, right...)
	hash := hash.HashBytes(combined)
	return hash[:]
}

func CalculateMerkleRoot(hashes [][]byte) []byte {
	// Repeat until there's only one hash (the Merkle root)
	for len(hashes) > 1 {
		var newLevel [][]byte
		// If there's an odd number of hashes, duplicate the last one
		if len(hashes)%2 != 0 {
			hashes = append(hashes, hashes[len(hashes)-1])
		}

		// Pair up and hash each pair
		for i := 0; i < len(hashes); i += 2 {
			newLevel = append(newLevel, HashPair(hashes[i], hashes[i+1]))
		}

		// Move to the next level with the newly generated hashes
		hashes = newLevel
	}

	// The remaining hash is the Merkle root
	return hashes[0]
}

func TransferToHash(bytesArray []hash.Hashable) []byte {

	var hashes [][]byte

	// Convert each Hashable object to its byte slice hash
	for _, item := range bytesArray {
		hashes = append(hashes, item.GetHash())
	}
	return CalculateMerkleRoot(hashes)
}
