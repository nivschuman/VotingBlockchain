package repositories

import (
	db_models "github.com/nivschuman/VotingBlockchain/internal/database/models"
	mapping "github.com/nivschuman/VotingBlockchain/internal/mapping"
	models "github.com/nivschuman/VotingBlockchain/internal/models"
	structures "github.com/nivschuman/VotingBlockchain/internal/structures"
	"gorm.io/gorm"
)

type TransactionRepository interface {
	TransactionsValidInChain(chainTipId []byte, transactions []*models.Transaction) (bool, error)
	GetTransaction(txId []byte) (*models.Transaction, error)
	GetTransactions(ids *structures.BytesSet) ([]*models.Transaction, error)
	GetMissingTransactionIds(ids *structures.BytesSet) (*structures.BytesSet, error)
	InsertIfNotExists(transaction *models.Transaction) error
	GetMempool(limit int) ([]*models.Transaction, error)
	TransactionValidInActiveChain(transaction *models.Transaction) (bool, error)
	InsertIfNotExistsTransactional(transaction *models.Transaction, tx *gorm.DB) error
}

type TransactionRepositoryImpl struct {
	db *gorm.DB
}

func NewTransactionRepositoryImpl(db *gorm.DB) *TransactionRepositoryImpl {
	return &TransactionRepositoryImpl{db: db}
}

func (repo *TransactionRepositoryImpl) TransactionsValidInChain(chainTipId []byte, transactions []*models.Transaction) (bool, error) {
	currentId := chainTipId

	voterPublicKeys := structures.NewBytesSet()
	for _, tx := range transactions {
		voterPublicKeys.Add(tx.VoterPublicKey)
	}

	for currentId != nil {
		var count int64
		err := repo.db.Table("transactions_blocks").
			Joins("JOIN transactions ON transactions_blocks.transaction_id = transactions.id").
			Where("transactions_blocks.block_header_id = ?", currentId).
			Where("transactions.voter_public_key IN ?", voterPublicKeys.ToBytesSlice()).
			Count(&count).Error

		if err != nil {
			return false, err
		}

		if count > 0 {
			return false, nil
		}

		var prevId []byte
		err = repo.db.Table("block_headers").
			Select("previous_block_header_id").
			Where("id = ?", currentId).Row().Scan(&prevId)

		if err != nil {
			return false, err
		}

		currentId = prevId
	}

	return true, nil
}

func (repo *TransactionRepositoryImpl) GetTransaction(txId []byte) (*models.Transaction, error) {
	var txDB db_models.TransactionDB
	result := repo.db.Where("id = ?", txId).First(&txDB)

	if result.Error != nil {
		return nil, result.Error
	}

	return mapping.TransactionDBToTransaction(&txDB), nil
}

func (repo *TransactionRepositoryImpl) GetTransactions(ids *structures.BytesSet) ([]*models.Transaction, error) {
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

func (repo *TransactionRepositoryImpl) GetMissingTransactionIds(ids *structures.BytesSet) (*structures.BytesSet, error) {
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

func (repo *TransactionRepositoryImpl) InsertIfNotExists(transaction *models.Transaction) error {
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

func (repo *TransactionRepositoryImpl) GetMempool(limit int) ([]*models.Transaction, error) {
	var transactionsDB []*db_models.TransactionDB

	subquery := repo.db.
		Table("transactions AS t").
		Select("1").
		Joins("JOIN transactions_blocks tb ON t.id = tb.transaction_id").
		Joins("JOIN blocks b ON tb.block_header_id = b.block_header_id").
		Where("b.in_active_chain = ?", true).
		Where("t.voter_public_key = transactions.voter_public_key")

	query := repo.db.
		Table("transactions").
		Joins("LEFT JOIN transactions_blocks ON transactions.id = transactions_blocks.transaction_id").
		Joins("LEFT JOIN blocks ON transactions_blocks.block_header_id = blocks.block_header_id").
		Where("blocks.in_active_chain = ? OR blocks.in_active_chain IS NULL", false).
		Where("NOT EXISTS (?)", subquery).
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

func (repo *TransactionRepositoryImpl) TransactionValidInActiveChain(transaction *models.Transaction) (bool, error) {
	var count int64
	err := repo.db.Table("transactions t").
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

func (repo *TransactionRepositoryImpl) InsertIfNotExistsTransactional(transaction *models.Transaction, tx *gorm.DB) error {
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
