package models_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/nivschuman/VotingBlockchain/internal/crypto/hash"
	"github.com/nivschuman/VotingBlockchain/internal/models"
)

func getTestElection() *models.Election {
	startTimestamp := int64(1700000000) // Example: Fixed Unix timestamp (Nov 14, 2023)
	endTimestamp := int64(1700604800)   // Example: Fixed Unix timestamp (7 days later, Nov 21, 2023)

	election := models.Election{
		Version:        1,
		StartTimestamp: startTimestamp,
		EndTimestamp:   endTimestamp,
	}

	election.SetId()

	return &election
}

func getExpectedElectionHash() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, int32(1))
	binary.Write(buf, binary.BigEndian, uint64(1700000000))
	binary.Write(buf, binary.BigEndian, uint64(1700604800))

	return hash.HashBytes(buf.Bytes())
}

func TestElectionGetHash(t *testing.T) {
	election := getTestElection()
	expectedHash := getExpectedElectionHash()
	receivedHash := election.GetHash()

	if !(bytes.Equal(expectedHash, receivedHash)) {
		t.Fatalf("expected hash isn't same as received hash")
	}
}

func TestElectionSetId(t *testing.T) {
	election := getTestElection()
	expectedId := getExpectedElectionHash()

	election.SetId()

	if !(bytes.Equal(expectedId, election.Id)) {
		t.Fatalf("expected id isn't same as received hash")
	}
}
