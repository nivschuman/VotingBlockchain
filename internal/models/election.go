package models

import (
	"encoding/binary"

	"github.com/nivschuman/VotingBlockchain/internal/crypto/hash"
)

type Election struct {
	Id                  []byte //hash of (StartTimestamp, EndTimestamp), 32 bytes
	StartTimestamp      int64  //unix timestamp of start of the election period, 8 bytes
	EndTimestamp        int64  //unix timestamp of end of the election period, 8 bytes
	GovernmentSignature []byte //signature of Id, 64 bytes, in ASN1 format, 70-72 bytes, signed by government
}

func (election *Election) GetHash() []byte {
	buf_size := 8 + 8
	buf := make([]byte, buf_size)

	binary.BigEndian.PutUint64(buf[0:8], uint64(election.StartTimestamp))
	binary.BigEndian.PutUint64(buf[8:16], uint64(election.StartTimestamp))

	return hash.HashBytes(buf)
}

func (election *Election) SetId() {
	election.Id = election.GetHash()
}
