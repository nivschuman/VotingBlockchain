package merkle_test

import (
	"bytes"
	"encoding/hex"
	"testing"

	hash "github.com/nivschuman/VotingBlockchain/internal/crypto/hash"
	merkle "github.com/nivschuman/VotingBlockchain/internal/crypto/merkle"
)

type stringContent struct {
	content string
}

func (stringContent *stringContent) GetHash() []byte {
	return hash.HashString(stringContent.content)
}

var hashables = []hash.Hashable{
	&stringContent{content: "hello"},
	&stringContent{content: "world"},
	&stringContent{content: "why"},
	&stringContent{content: "what"},
}

/*
	H(hello) = 2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824
	H(world) = 486ea46224d1bb4fb680f34f7c9ad96a8f24ec88be73ea8e5a6c65260e9cb8a7
	H(why) = 2be23c585f15e5fd3279d0663036dd9f6e634f4225ef326fc83fb874dbb81a0f
	H(what) = 749ab2c0d06c42ae3b841b79e79875f02b3a042e43c92378cd28bd444c04d284

	H(H(hello) || H(world)) = 7305db9b2abccd706c256db3d97e5ff48d677cfe4d3a5904afb7da0e3950e1e2
	H(H(why) || H(what)) = dd87a4c8c1271a1e926cd6699021404b18505e6e4def12e270c67556e562c721

	H(H(H(hello) || H(world)) || H(H(why) || H(what))) = 203dca499fcc350b3885ec3f8bdb2a0980ffb62d67f803076ee3f7c5f554a328
*/

func TestCalculateMerkleRoot(t *testing.T) {
	expectedMerkleRoot, err := hex.DecodeString("203dca499fcc350b3885ec3f8bdb2a0980ffb62d67f803076ee3f7c5f554a328")
	if err != nil {
		t.Fatalf("Failed to decode hex string: %v", err)
	}

	merkleRoot := merkle.CalculateMerkleRoot(hashables)
	if !bytes.Equal(merkleRoot, expectedMerkleRoot) {
		t.Fatalf("Calculated wrong merkle root %x", merkleRoot)
	}
}

func TestGenerateMerkleProof(t *testing.T) {
	proof, ok := merkle.GenerateMerkleProof(hashables, &stringContent{content: "world"})
	if !ok {
		t.Fatalf("Failed to generate merkle proof")
	}

	if len(proof.Items) != 2 {
		t.Fatalf("Proof doesn't have right amount of items")
	}

	item1, err := hex.DecodeString("2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824")
	if err != nil {
		t.Fatalf("Failed to decode hex string: %v", err)
	}

	if !bytes.Equal(proof.Items[0].Hash, item1) {
		t.Fatalf("Proof is incorrect")
	}

	if !proof.Items[0].IsLeft {
		t.Fatalf("Proof is incorrect")
	}

	item2, err := hex.DecodeString("dd87a4c8c1271a1e926cd6699021404b18505e6e4def12e270c67556e562c721")
	if err != nil {
		t.Fatalf("Failed to decode hex string: %v", err)
	}

	if !bytes.Equal(proof.Items[1].Hash, item2) {
		t.Fatalf("Proof is incorrect")
	}

	if proof.Items[1].IsLeft {
		t.Fatalf("Proof is incorrect")
	}
}

func TestVerifyMerkleProof_WhenProofIsValid(t *testing.T) {
	leaf := &stringContent{content: "world"}

	item1, err := hex.DecodeString("2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824")
	if err != nil {
		t.Fatalf("Failed to decode hex string: %v", err)
	}

	item2, err := hex.DecodeString("dd87a4c8c1271a1e926cd6699021404b18505e6e4def12e270c67556e562c721")
	if err != nil {
		t.Fatalf("Failed to decode hex string: %v", err)
	}

	proof := &merkle.MerkleProof{
		Items: []merkle.ProofItem{
			{Hash: item1, IsLeft: true},
			{Hash: item2, IsLeft: false},
		},
	}

	expectedRoot, err := hex.DecodeString("203dca499fcc350b3885ec3f8bdb2a0980ffb62d67f803076ee3f7c5f554a328")
	if err != nil {
		t.Fatalf("Failed to decode hex string: %v", err)
	}

	ok := merkle.VerifyMerkleProof(leaf, proof, expectedRoot)
	if !ok {
		t.Fatalf("Merkle proof verification failed for valid proof")
	}
}

func TestVerifyMerkleProof_WhenProofIsInvalid(t *testing.T) {
	leaf := &stringContent{content: "world"}

	item1, err := hex.DecodeString("2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824")
	if err != nil {
		t.Fatalf("Failed to decode hex string: %v", err)
	}

	item2, err := hex.DecodeString("dd87a4c8c1271a1e926cd6699021404b18505e6e4def12e270c67556e562c720") // last byte changed from 0x21 to 0x20
	if err != nil {
		t.Fatalf("Failed to decode hex string: %v", err)
	}

	proof := &merkle.MerkleProof{
		Items: []merkle.ProofItem{
			{Hash: item1, IsLeft: true},
			{Hash: item2, IsLeft: false},
		},
	}

	expectedRoot, err := hex.DecodeString("203dca499fcc350b3885ec3f8bdb2a0980ffb62d67f803076ee3f7c5f554a328")
	if err != nil {
		t.Fatalf("Failed to decode hex string: %v", err)
	}

	ok := merkle.VerifyMerkleProof(leaf, proof, expectedRoot)
	if ok {
		t.Fatalf("Merkle proof verification passed for invalid proof")
	}
}
