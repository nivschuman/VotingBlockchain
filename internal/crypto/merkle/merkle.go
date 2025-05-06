package merkle

import (
	"bytes"

	"github.com/nivschuman/VotingBlockchain/internal/crypto/hash"
)

func HashPair(left []byte, right []byte) []byte {
	combined := append(left, right...)
	hash := hash.HashBytes(combined)
	return hash[:]
}

func CalculateMerkleRoot(bytesArray []hash.Hashable) []byte {
	hashes := TransferToHash(bytesArray)
	for len(hashes) > 1 {
		var newLevel [][]byte
		if len(hashes)%2 != 0 {
			hashes = append(hashes, hashes[len(hashes)-1])
		}

		for i := 0; i < len(hashes); i += 2 {
			newLevel = append(newLevel, HashPair(hashes[i], hashes[i+1]))
		}

		hashes = newLevel
	}

	return hashes[0]
}

func TransferToHash(bytesArray []hash.Hashable) [][]byte {
	var hashes [][]byte
	for _, item := range bytesArray {
		hashes = append(hashes, item.GetHash())
	}
	return hashes
}

func GenerateMerkleProof(items []hash.Hashable, target hash.Hashable) ([][]byte, bool) {
	var proof [][]byte
	hashes := TransferToHash(items)
	index := -1

	targetHash := target.GetHash()
	for i, h := range hashes {
		if bytes.Equal(h, targetHash) {
			index = i
			break
		}
	}

	if index == -1 {
		return nil, false
	}

	for len(hashes) > 1 {
		var newLevel [][]byte
		if len(hashes)%2 != 0 {
			hashes = append(hashes, hashes[len(hashes)-1])
		}

		for i := 0; i < len(hashes); i += 2 {
			combinedHash := HashPair(hashes[i], hashes[i+1])
			newLevel = append(newLevel, combinedHash)

			if i == index {
				proof = append(proof, hashes[i+1])
			} else if i+1 == index {
				proof = append(proof, hashes[i])
			}
		}

		index /= 2
		hashes = newLevel
	}

	return proof, true

}

func VerifyMerkleProof(leaf hash.Hashable, proof [][]byte, root []byte) bool {
	hash := leaf.GetHash()

	for _, sibling := range proof {
		hash = HashPair(hash, sibling)
	}

	return bytes.Equal(hash, root)
}
