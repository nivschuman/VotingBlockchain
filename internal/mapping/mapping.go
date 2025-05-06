package mapping

import (
	"slices"

	db_models "github.com/nivschuman/VotingBlockchain/internal/database/models"
	models "github.com/nivschuman/VotingBlockchain/internal/models"
)

func TransactionToTransactionDB(transaction *models.Transaction) *db_models.TransactionDB {
	return &db_models.TransactionDB{
		Id:                  slices.Clone(transaction.Id),
		Version:             transaction.Version,
		CandidateId:         transaction.CandidateId,
		VoterPublicKey:      slices.Clone(transaction.VoterPublicKey),
		GovernmentSignature: slices.Clone(transaction.GovernmentSignature),
		Signature:           slices.Clone(transaction.Signature),
	}
}

func TransactionDBToTransaction(transactionDB *db_models.TransactionDB) *models.Transaction {
	return &models.Transaction{
		Id:                  slices.Clone(transactionDB.Id),
		Version:             transactionDB.Version,
		CandidateId:         transactionDB.CandidateId,
		VoterPublicKey:      slices.Clone(transactionDB.VoterPublicKey),
		GovernmentSignature: slices.Clone(transactionDB.GovernmentSignature),
		Signature:           slices.Clone(transactionDB.Signature),
	}
}

func BlockHeaderToBlockHeaderDB(blockHeader *models.BlockHeader) *db_models.BlockHeaderDB {
	var prevBlockHeaderId *[]byte

	if blockHeader.PreviousBlockId != nil {
		copyPrev := slices.Clone(blockHeader.PreviousBlockId)
		prevBlockHeaderId = &copyPrev
	}

	return &db_models.BlockHeaderDB{
		Id:                    slices.Clone(blockHeader.Id),
		Version:               blockHeader.Version,
		MerkleRoot:            slices.Clone(blockHeader.MerkleRoot),
		Timestamp:             blockHeader.Timestamp,
		NBits:                 blockHeader.NBits,
		Nonce:                 blockHeader.Nonce,
		MinerPublicKey:        slices.Clone(blockHeader.MinerPublicKey),
		PreviousBlockHeaderId: prevBlockHeaderId,
	}
}

func BlockToBlockDB(block *models.Block) *db_models.BlockDB {
	blockHeaderDB := BlockHeaderToBlockHeaderDB(&block.Header)

	blockDB := &db_models.BlockDB{
		BlockHeaderId: slices.Clone(blockHeaderDB.Id),
		BlockHeader:   *blockHeaderDB,
	}

	return blockDB
}
