package transaction

import (
	"fmt"

	"github.com/nivschuman/VotingBlockchain/internal/crypto/hash"
	"github.com/nivschuman/VotingBlockchain/internal/crypto/ppk"
)

type Transaction struct {
	Id             []byte //hash of (CandidateId, Year, VoterPublicKey), 32 bytes
	CandidateId    int32  //id of candidate to vote for
	Year           int32  //year of vote
	VoterPublicKey []byte //compressed ECDSA public key, 33 bytes
	Signature      []byte //signature of Id, 64 bytes
}

func (transaction *Transaction) GetTransactionHash() []byte {
	data := fmt.Sprintf("%d%d%s", transaction.CandidateId, transaction.Year, transaction.VoterPublicKey)
	return hash.HashStringAsBytes(data)
}

func (transaction *Transaction) SetId() {
	transaction.Id = transaction.GetTransactionHash()
}

func (transaction *Transaction) IsValidSignature() bool {
	hash := transaction.GetTransactionHash()
	publicKey := ppk.GetPublicKeyFromBytes(transaction.VoterPublicKey)
	return publicKey.VerifySignature(transaction.Signature, hash)
}
