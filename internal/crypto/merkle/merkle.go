package merkle

import (
	"bytes"

	"github.com/nivschuman/VotingBlockchain/internal/crypto/hash"
)

type ProofItem struct {
	Hash   []byte
	IsLeft bool
}

type MerkleProof struct {
	Items []ProofItem
}

func CalculateMerkleRoot(hashables []hash.Hashable) []byte {
	hashes := getHashes(hashables)
	for len(hashes) > 1 {
		var newLevel [][]byte
		if len(hashes)%2 != 0 {
			hashes = append(hashes, hashes[len(hashes)-1])
		}

		for i := 0; i < len(hashes); i += 2 {
			newLevel = append(newLevel, hashPair(hashes[i], hashes[i+1]))
		}

		hashes = newLevel
	}

	return hashes[0]
}

func GenerateMerkleProof(items []hash.Hashable, target hash.Hashable) (*MerkleProof, bool) {
	var proofItems []ProofItem
	hashes := getHashes(items)
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
		if len(hashes)%2 != 0 {
			hashes = append(hashes, hashes[len(hashes)-1])
		}

		var newLevel [][]byte

		for i := 0; i < len(hashes); i += 2 {
			left, right := hashes[i], hashes[i+1]
			combinedHash := hashPair(left, right)
			newLevel = append(newLevel, combinedHash)

			if i == index {
				proofItems = append(proofItems, ProofItem{Hash: right, IsLeft: false})
			} else if i+1 == index {
				proofItems = append(proofItems, ProofItem{Hash: left, IsLeft: true})
			}
		}

		index /= 2
		hashes = newLevel
	}

	return &MerkleProof{Items: proofItems}, true
}

func VerifyMerkleProof(leaf hash.Hashable, proof *MerkleProof, root []byte) bool {
	hash := leaf.GetHash()

	for _, item := range proof.Items {
		if item.IsLeft {
			hash = hashPair(item.Hash, hash)
		} else {
			hash = hashPair(hash, item.Hash)
		}
	}

	return bytes.Equal(hash, root)
}

func getHashes(hashables []hash.Hashable) [][]byte {
	var hashes [][]byte
	for _, item := range hashables {
		hashes = append(hashes, item.GetHash())
	}
	return hashes
}

func hashPair(left []byte, right []byte) []byte {
	combined := append(left, right...)
	hash := hash.HashBytes(combined)
	return hash[:]
}
