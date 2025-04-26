package repositories

import (
	"bytes"
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
	ActiveChainTipId      []byte
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

func (blockRepository *BlockRepository) InsertIfNotExists(block *models.Block) error {
	return blockRepository.db.Transaction(func(tx *gorm.DB) error {
		var existing db_models.BlockDB
		result := tx.Where("block_header_id = ?", block.Header.Id).Find(&existing)

		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected > 0 {
			return nil
		}

		blockDB := mapping.BlockToBlockDB(block)
		blockWork := types.NewBigInt(block.GetBlockWork())

		if block.Header.PreviousBlockId == nil {
			blockDB.Height = 0
			blockDB.InActiveChain = true
			blockDB.CumulativeWork = blockWork
		} else {
			var prevBlockDB db_models.BlockDB
			result := tx.Where("block_header_id = ?", block.Header.PreviousBlockId).Find(&prevBlockDB)

			if result.Error != nil {
				return result.Error
			}

			if result.RowsAffected > 0 {
				blockDB.Height = prevBlockDB.Height + 1
				blockDB.InActiveChain = bytes.Equal(block.Header.PreviousBlockId, blockRepository.ActiveChainTipId)
				blockDB.CumulativeWork = blockWork.Add(prevBlockDB.CumulativeWork)
			} else {
				return fmt.Errorf("orphan")
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
			if err := blockRepository.TransactionRepository.insertIfNotExistsTransactional(transaction, tx); err != nil {
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

		if blockDB.InActiveChain {
			blockRepository.ActiveChainTipId = blockDB.BlockHeaderId
			return nil
		}

		var activeTip db_models.BlockDB
		if err := tx.Where("block_header_id = ?", blockRepository.ActiveChainTipId).First(&activeTip).Error; err != nil {
			return fmt.Errorf("active chain tip not found: %v", err)
		}

		if blockDB.CumulativeWork.Cmp(activeTip.CumulativeWork) > 0 {
			if err := blockRepository.reorganizeChain(tx, block.Header.Id); err != nil {
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

func (blockRepository *BlockRepository) SetActiveChainTipId() error {
	var tipId []byte

	subQuery := blockRepository.db.
		Table("blocks AS b2").
		Select("b2.block_header_id").
		Joins("JOIN block_headers AS bh2 ON b2.block_header_id = bh2.id").
		Where("b2.in_active_chain = ?", true).
		Where("bh2.previous_block_header_id = b.block_header_id")

	err := blockRepository.db.
		Table("blocks AS b").
		Select("b.block_header_id").
		Where("b.in_active_chain = ?", true).
		Where("NOT EXISTS (?)", subQuery).
		Limit(1).
		Row().
		Scan(&tipId)

	if err != nil {
		return err
	}

	if len(tipId) == 0 {
		return fmt.Errorf("no chain tip found")
	}

	blockRepository.ActiveChainTipId = tipId
	return nil
}

func (blockRepository *BlockRepository) reorganizeChain(tx *gorm.DB, newTipId []byte) error {
	var forkPoint []byte
	oldTipId := blockRepository.ActiveChainTipId

	curId := newTipId

	for {
		var block db_models.BlockDB
		if err := tx.Preload("BlockHeader").Where("block_header_id = ?", curId).First(&block).Error; err != nil {
			return err
		}

		if block.InActiveChain {
			forkPoint = block.BlockHeaderId
			break
		}

		if err := tx.Model(&db_models.BlockDB{}).
			Where("block_header_id = ?", block.BlockHeaderId).
			Update("in_active_chain", true).Error; err != nil {
			return err
		}

		curId = *block.BlockHeader.PreviousBlockHeaderId
	}

	for {
		if bytes.Equal(oldTipId, forkPoint) {
			break
		}

		if err := tx.Model(&db_models.BlockDB{}).
			Where("block_header_id = ?", oldTipId).
			Update("in_active_chain", false).Error; err != nil {
			return err
		}

		var oldBlock db_models.BlockDB
		if err := tx.Preload("BlockHeader").Where("block_header_id = ?", oldTipId).First(&oldBlock).Error; err != nil {
			return err
		}

		oldTipId = *oldBlock.BlockHeader.PreviousBlockHeaderId
	}

	blockRepository.ActiveChainTipId = newTipId
	return nil
}
