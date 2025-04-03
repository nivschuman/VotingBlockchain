package repositories

import (
	db_models "github.com/nivschuman/VotingBlockchain/internal/database/models"
	mapping "github.com/nivschuman/VotingBlockchain/internal/mapping"
	models "github.com/nivschuman/VotingBlockchain/internal/models"
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

func (repo *TransactionRepository) InsertIfNotExists(transaction *models.Transaction, tx *gorm.DB) error {
	existingTransaction := &db_models.TransactionDB{}
	err := tx.Where("id = ?", transaction.Id).First(existingTransaction).Error

	if err == gorm.ErrRecordNotFound {
		transactionDB := mapping.TransactionToTransactionDB(transaction)
		return tx.Create(transactionDB).Error
	}

	return err
}
