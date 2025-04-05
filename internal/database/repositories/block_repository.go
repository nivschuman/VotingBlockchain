package repositories

import (
	"fmt"
	"math/big"
	"time"

	db_models "github.com/nivschuman/VotingBlockchain/internal/database/models"
	types "github.com/nivschuman/VotingBlockchain/internal/database/types"
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

func (blockRepository *BlockRepository) GetBlockCumulativeWork(blockHeaderId []byte) (*big.Int, error) {
	var blockDB db_models.BlockDB
	err := blockRepository.db.Where("block_header_id = ?", blockHeaderId).First(&blockDB).Error

	if err != nil {
		return nil, err
	}

	return (*big.Int)(&blockDB.CumulativeWork), nil
}

func (blockRepository *BlockRepository) InsertBlock(block *models.Block) error {
	return blockRepository.db.Transaction(func(tx *gorm.DB) error {
		blockDB := mapping.BlockToBlockDB(block)
		blockWork := types.NewBigInt(block.GetBlockWork())

		if block.Header.PreviousBlockId == nil {
			blockDB.Height = 0
			blockDB.InActiveChain = true
			blockDB.CumulativeWork = blockWork
		} else {
			var prevBlockDB db_models.BlockDB
			err := tx.Where("block_header_id = ?", block.Header.PreviousBlockId).First(&prevBlockDB).Error
			prevBlockExists := err == nil

			if prevBlockExists {
				blockDB.Height = prevBlockDB.Height + 1
				blockDB.InActiveChain = prevBlockDB.InActiveChain
				blockDB.CumulativeWork = blockWork.Add(prevBlockDB.CumulativeWork)
			} else {
				return fmt.Errorf("cannot insert orphan block, previous block %x not found", block.Header.PreviousBlockId)
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
