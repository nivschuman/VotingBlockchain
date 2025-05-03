package repositories

import (
	db_models "github.com/nivschuman/VotingBlockchain/internal/database/models"
	mapping "github.com/nivschuman/VotingBlockchain/internal/mapping"
	models "github.com/nivschuman/VotingBlockchain/internal/models"
	structures "github.com/nivschuman/VotingBlockchain/internal/structures"
	"gorm.io/gorm"
)

type TransactionRepository struct {
	db *gorm.DB
}

var GlobalTransactionRepository *TransactionRepository

func InitializeGlobalTransactionRepository(db *gorm.DB) error {
	if GlobalTransactionRepository != nil {
		return nil
	}

	GlobalTransactionRepository = &TransactionRepository{
		db: db,
	}

	return nil
}

func (repo *TransactionRepository) GetTransactions(ids *structures.BytesSet) ([]*models.Transaction, error) {
	var transactionsDB []db_models.TransactionDB
	result := repo.db.Where("id IN (?)", ids.ToBytesSlice()).Find(&transactionsDB)

	if result.Error != nil {
		return nil, result.Error
	}

	transactions := make([]*models.Transaction, len(transactionsDB))

	for idx, txDB := range transactionsDB {
		transactions[idx] = mapping.TransactionDBToTransaction(&txDB)
	}

	return transactions, nil
}

func (repo *TransactionRepository) GetMissingTransactionIds(ids *structures.BytesSet) (*structures.BytesSet, error) {
	transactionIds := ids.ToBytesSlice()

	if len(transactionIds) == 0 {
		return nil, nil
	}

	var existingTransactions []db_models.TransactionDB
	if err := repo.db.Where("id IN (?)", transactionIds).Find(&existingTransactions).Error; err != nil {
		return nil, err
	}

	existingIds := structures.NewBytesSet()
	for _, transaction := range existingTransactions {
		existingIds.Add(transaction.Id)
	}

	missingIds := structures.NewBytesSet()

	for _, id := range transactionIds {
		if !existingIds.Contains(id) {
			missingIds.Add(id)
		}
	}

	return missingIds, nil
}

func (repo *TransactionRepository) InsertIfNotExists(transaction *models.Transaction) error {
	return repo.db.Transaction(func(tx *gorm.DB) error {
		existingTransaction := &db_models.TransactionDB{}
		result := tx.Where("id = ?", transaction.Id).Find(existingTransaction)

		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 0 {
			transactionDB := mapping.TransactionToTransactionDB(transaction)
			if err := tx.Create(transactionDB).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (repo *TransactionRepository) GetMempool(limit int) ([]*models.Transaction, error) {
	var transactionsDB []*db_models.TransactionDB

	query := repo.db.Table("transactions").
		Joins("LEFT JOIN transactions_blocks ON transactions.id = transactions_blocks.transaction_id").
		Joins("LEFT JOIN blocks ON transactions_blocks.block_header_id = blocks.block_header_id").
		Where("blocks.in_active_chain = ? OR blocks.in_active_chain IS NULL", false).
		Limit(limit)

	err := query.Find(&transactionsDB).Error

	if err != nil {
		return nil, err
	}

	var transactions []*models.Transaction

	for _, txDB := range transactionsDB {
		transactions = append(transactions, mapping.TransactionDBToTransaction(txDB))
	}

	return transactions, nil
}

func (repo *TransactionRepository) TransactionIsValid(transaction *models.Transaction) (bool, error) {
	valid, err := transaction.GovernmentSignatureIsValid()

	if err != nil {
		return false, err
	}

	if !valid {
		return false, nil
	}

	valid, err = transaction.SignatureIsValid()

	if err != nil {
		return false, err
	}

	if !valid {
		return false, nil
	}

	var count int64
	err = repo.db.Table("transactions t").
		Joins("JOIN transactions_blocks tb ON tb.transaction_id = t.id").
		Joins("JOIN blocks b ON b.block_header_id = tb.block_header_id").
		Where("t.voter_public_key = ? AND b.in_active_chain = ?", transaction.VoterPublicKey, true).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	if count > 0 {
		return false, nil
	}

	return true, nil
}

func (repo *TransactionRepository) insertIfNotExistsTransactional(transaction *models.Transaction, tx *gorm.DB) error {
	existingTransaction := &db_models.TransactionDB{}
	result := tx.Where("id = ?", transaction.Id).Find(existingTransaction)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		transactionDB := mapping.TransactionToTransactionDB(transaction)
		return tx.Create(transactionDB).Error
	}

	return nil
}
