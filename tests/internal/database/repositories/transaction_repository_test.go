package repositories_test

import (
	"testing"

	"github.com/nivschuman/VotingBlockchain/internal/database/repositories"
	models "github.com/nivschuman/VotingBlockchain/internal/models"
	structures "github.com/nivschuman/VotingBlockchain/internal/structures"
	inits "github.com/nivschuman/VotingBlockchain/tests/init"
)

func TestGetMempool(t *testing.T) {
	inits.ResetTestDatabase()
	govKeyPair, _, _, err := inits.CreateTestData(4, 2)

	if err != nil {
		t.Fatalf("failed to setup test data: %v", err)
	}

	tx1, _, err := inits.CreateTestTransaction(govKeyPair)
	if err != nil {
		t.Fatalf("failed to create test tx1: %v", err)
	}

	tx2, _, err := inits.CreateTestTransaction(govKeyPair)
	if err != nil {
		t.Fatalf("failed to create test tx2: %v", err)
	}

	err = repositories.GlobalTransactionRepository.InsertIfNotExists(tx1)
	if err != nil {
		t.Fatalf("failed to create insert tx1: %v", err)
	}

	err = repositories.GlobalTransactionRepository.InsertIfNotExists(tx2)
	if err != nil {
		t.Fatalf("failed to create insert tx2: %v", err)
	}

	addedByteSet := structures.NewBytesSet()
	addedByteSet.Add(tx1.Id)
	addedByteSet.Add(tx2.Id)

	txs, err := repositories.GlobalTransactionRepository.GetMempool(10)

	if err != nil {
		t.Fatalf("failed to get mempool: %v", err)
	}

	if len(txs) != addedByteSet.Length() {
		t.Fatalf("received incorrect amount of transactions from mempool: %d", len(txs))
	}

	for _, tx := range txs {
		if !addedByteSet.Contains(tx.Id) {
			t.Fatalf("received invalid transaction: %x", tx.Id)
		}
	}
}

func TestTransactionValidInActiveChain_WhenTransactionIsInvalid(t *testing.T) {
	inits.ResetTestDatabase()
	_, blocks, keyPairs, err := inits.CreateTestData(4, 2)
	if err != nil {
		t.Fatalf("failed to setup test data: %v", err)
	}

	block := blocks[len(blocks)-1]
	blockTransaction := block.Transactions[0]

	tx := &models.Transaction{
		Version:             1,
		CandidateId:         blockTransaction.CandidateId,
		VoterPublicKey:      blockTransaction.VoterPublicKey,
		GovernmentSignature: blockTransaction.GovernmentSignature,
	}

	tx.SetId()
	signature, err := keyPairs[string(blockTransaction.Id)].PrivateKey.CreateSignature(tx.Id)

	if err != nil {
		t.Fatalf("failed to sign transaction: %v", err)
	}

	tx.Signature = signature

	isValid, err := repositories.GlobalTransactionRepository.TransactionValidInActiveChain(tx)

	if err != nil {
		t.Fatalf("failed to check if transaction is valid: %v", err)
	}

	if isValid {
		t.Fatalf("transaction was determined as valid")
	}
}

func TestTransactionValidInActiveChain_WhenTransactionIsValid(t *testing.T) {
	inits.ResetTestDatabase()
	govKeyPair, _, _, err := inits.CreateTestData(4, 2)

	if err != nil {
		t.Fatalf("failed to setup test data: %v", err)
	}

	tx, _, err := inits.CreateTestTransaction(govKeyPair)

	if err != nil {
		t.Fatalf("failed to create test transactions: %v", err)
	}

	isValid, err := repositories.GlobalTransactionRepository.TransactionValidInActiveChain(tx)

	if err != nil {
		t.Fatalf("failed to check if transaction is valid: %v", err)
	}

	if !isValid {
		t.Fatalf("transaction was determined as invalid")
	}
}

func TestTransactionsValidInChain_WhenTransactionsAreValid(t *testing.T) {
	inits.ResetTestDatabase()
	govKeyPair, blocks, _, err := inits.CreateTestData(4, 2)

	if err != nil {
		t.Fatalf("failed to setup test data: %v", err)
	}

	tx1, _, err := inits.CreateTestTransaction(govKeyPair)

	if err != nil {
		t.Fatalf("failed to create test transaction: %v", err)
	}

	tx2 := blocks[len(blocks)-1].Transactions[0]

	txs := []*models.Transaction{tx1, tx2}
	isValid, err := repositories.GlobalTransactionRepository.TransactionsValidInChain(blocks[0].Header.Id, txs)

	if err != nil {
		t.Fatalf("failed to check if transactions are valid: %v", err)
	}

	if !isValid {
		t.Fatalf("transactions were determined as invalid")
	}
}

func TestGetMissingTransactionIds(t *testing.T) {
	inits.ResetTestDatabase()
	govKeyPair, blocks, _, err := inits.CreateTestData(4, 2)
	if err != nil {
		t.Fatalf("failed to setup test data: %v", err)
	}

	tx1, _, err := inits.CreateTestTransaction(govKeyPair)
	if err != nil {
		t.Fatalf("failed to create test tx1: %v", err)
	}

	tx2, _, err := inits.CreateTestTransaction(govKeyPair)
	if err != nil {
		t.Fatalf("failed to create test tx2: %v", err)
	}

	lastBlock := blocks[len(blocks)-1]

	ids := structures.NewBytesSet()
	ids.Add(tx1.Id)
	ids.Add(tx2.Id)

	for _, tx := range lastBlock.Transactions {
		ids.Add(tx.Id)
	}

	missing, err := repositories.GlobalTransactionRepository.GetMissingTransactionIds(ids)

	if err != nil {
		t.Fatalf("failed to get missing transactions: %v", err)
	}

	if !missing.Contains(tx1.Id) || !missing.Contains(tx2.Id) {
		t.Fatalf("missing transactions weren't returned")
	}

	for _, tx := range lastBlock.Transactions {
		if missing.Contains(tx.Id) {
			t.Fatalf("transaction that isn't missing was returned")
		}
	}
}
