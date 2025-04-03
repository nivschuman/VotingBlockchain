package repositories

import (
	"time"

	db_models "github.com/nivschuman/VotingBlockchain/internal/database/models"
	mapping "github.com/nivschuman/VotingBlockchain/internal/mapping"
	models "github.com/nivschuman/VotingBlockchain/internal/models"
	"gorm.io/gorm"
)

type BlockRepository struct {
	db                    *gorm.DB
	TransactionRepository *TransactionRepository
}

var GlobalBlockRepository *BlockRepository

func InitializeGlobalBlockRepository(db *gorm.DB) error {
	if GlobalBlockRepository != nil {
		return nil
	}

	GlobalBlockRepository = &BlockRepository{
		db:                    db,
		TransactionRepository: GlobalTransactionRepository,
	}

	return nil
}

func (blockRepository *BlockRepository) InsertBlock(block *models.Block) error {
	return blockRepository.db.Transaction(func(tx *gorm.DB) error {
		var prevBlockDB db_models.BlockDB
		prevBlockExists := false

		blockDB := mapping.BlockToBlockDB(block)

		//genesis block
		if block.Header.PreviousBlockId == nil {
			blockDB.Height = 0
			blockDB.InActiveChain = true
		} else {
			err := tx.Where("id = ?", block.Header.PreviousBlockId).First(&prevBlockDB).Error
			prevBlockExists = err == nil

			//prev block exists, append to it
			if prevBlockExists {
				blockDB.Height = prevBlockDB.Height + 1
				blockDB.InActiveChain = prevBlockDB.InActiveChain
				// orphan block
			} else {
				blockDB.Height = 0
				blockDB.InActiveChain = false
			}
		}

		blockHeaderDB := mapping.BlockHeaderToBlockHeaderDB(&block.Header)
		if err := tx.Create(blockHeaderDB).Error; err != nil {
			return err
		}

		if err := tx.Create(blockDB).Error; err != nil {
			return err
		}

		for i, transaction := range block.Transactions {
			if err := blockRepository.TransactionRepository.InsertIfNotExists(transaction, tx); err != nil {
				return err
			}

			transactionBlock := db_models.TransactionBlockDB{
				BlockHeaderId: block.Header.Id,
				TransactionId: transaction.Id,
				Order:         uint32(i),
			}

			if err := tx.Create(&transactionBlock).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (*BlockRepository) GenesisBlock() *models.Block {
	genesisBlockHeader := &models.BlockHeader{
		Version:         1,
		PreviousBlockId: nil,
		MerkleRoot:      make([]byte, 32),
		Timestamp:       time.Date(2025, time.March, 30, 0, 0, 0, 0, time.UTC).Unix(),
		NBits:           uint32(0x1d00ffff),
		Nonce:           0,
		MinerPublicKey:  make([]byte, 33),
	}

	genesisBlockHeader.SetId()

	return &models.Block{
		Header:       *genesisBlockHeader,
		Transactions: make([]*models.Transaction, 0),
	}
}
