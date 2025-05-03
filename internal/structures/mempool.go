package structures

import (
	models "github.com/nivschuman/VotingBlockchain/internal/models"
)

type MemPool struct {
	transactions map[string]*models.Transaction
}

func NewMemPool() *MemPool {
	return &MemPool{
		transactions: make(map[string]*models.Transaction),
	}
}

func (memPool *MemPool) Insert(transaction *models.Transaction) {
	memPool.transactions[string(transaction.Id)] = transaction
}

func (memPool *MemPool) Contains(transactionId []byte) bool {
	_, exists := memPool.transactions[string(transactionId)]
	return exists
}

func (memPool *MemPool) Remove(transactionId []byte) {
	delete(memPool.transactions, string(transactionId))
}

func (memPool *MemPool) Transactions() []*models.Transaction {
	transactions := make([]*models.Transaction, 0, len(memPool.transactions))
	for _, tx := range memPool.transactions {
		transactions = append(transactions, tx)
	}
	return transactions
}

func (memPool *MemPool) GetMissingTransactionIds(ids *BytesSet) *BytesSet {
	missing := NewBytesSet()

	for _, id := range ids.ToBytesSlice() {
		if !memPool.Contains(id) {
			missing.Add(id)
		}
	}

	return missing
}
