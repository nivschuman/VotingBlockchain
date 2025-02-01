package models

import (
	"bytes"
	"encoding/binary"

	"github.com/nivschuman/VotingBlockchain/internal/crypto/hash"
)

type Election struct {
	Id                  []byte //hash of (Version, StartTimestamp, EndTimestamp), 32 bytes
	Version             int32  //version of election, 4 bytes
	StartTimestamp      int64  //unix timestamp of start of the election period, 8 bytes
	EndTimestamp        int64  //unix timestamp of end of the election period, 8 bytes
	GovernmentSignature []byte //signature of Id, in ASN1 format, 70-72 bytes, signed by government
}

func (election *Election) GetHash() []byte {
	buf := new(bytes.Buffer)

	binary.Write(buf, binary.BigEndian, election.Version)
	binary.Write(buf, binary.BigEndian, election.StartTimestamp)
	binary.Write(buf, binary.BigEndian, election.EndTimestamp)

	return hash.HashBytes(buf.Bytes())
}

func (election *Election) SetId() {
	election.Id = election.GetHash()
}
