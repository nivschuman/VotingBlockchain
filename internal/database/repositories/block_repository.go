package repositories

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"slices"
	"time"

	db_models "github.com/nivschuman/VotingBlockchain/internal/database/models"
	types "github.com/nivschuman/VotingBlockchain/internal/database/types"
	difficulty "github.com/nivschuman/VotingBlockchain/internal/difficulty"
	mapping "github.com/nivschuman/VotingBlockchain/internal/mapping"
	models "github.com/nivschuman/VotingBlockchain/internal/models"
	structures "github.com/nivschuman/VotingBlockchain/internal/structures"
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

func (repo *BlockRepository) HaveBlock(blockId []byte) (bool, error) {
	var count int64
	result := repo.db.Table("block_headers").Where("block_headers.id = ?", blockId).Count(&count)

	if result.Error != nil {
		return false, result.Error
	}

	return count > 0, nil
}

func (repo *BlockRepository) BlockIsOrphan(block *models.Block) (bool, error) {
	var count int64
	result := repo.db.Table("block_headers").Where("block_headers.id = ?", block.Header.PreviousBlockId).Count(&count)

	if result.Error != nil {
		return true, result.Error
	}

	return count <= 0, nil
}

func (repo *BlockRepository) GetMedianTimePast(startBlockId []byte, numberOfBlocks int) (int64, error) {
	times := make([]int64, 0, numberOfBlocks)
	currentId := startBlockId

	for range numberOfBlocks {
		var blockHeader db_models.BlockHeaderDB
		result := repo.db.Where("block_headers.id = ?", currentId).Limit(1).Find(&blockHeader)

		if result.Error != nil {
			return -1, result.Error
		}

		if result.RowsAffected == 0 {
			break
		}

		times = append(times, blockHeader.Timestamp)

		if blockHeader.PreviousBlockHeaderId == nil {
			break
		}

		currentId = *blockHeader.PreviousBlockHeaderId
	}

	slices.Sort(times)
	medianTime := times[len(times)/2]

	return medianTime, nil
}

func (repo *BlockRepository) GetNextBlocksIds(blockLocator *structures.BlockLocator, stopHash []byte, limit int) (*structures.BytesSet, error) {
	ids := blockLocator.Ids()

	var currentId []byte
	for _, id := range ids {
		var count int64
		err := repo.db.Table("blocks").Where("block_header_id = ?", id).Count(&count).Error

		if err != nil {
			return nil, err
		}

		if count > 0 {
			currentId = slices.Clone(id)
			break
		}
	}

	blocksIds := structures.NewBytesSet()

	if currentId == nil {
		return blocksIds, nil
	}

	for range limit {
		var nextId []byte
		err := repo.db.Table("block_headers").
			Select("block_headers.id").
			Joins("JOIN blocks ON blocks.block_header_id = block_headers.id").
			Where("block_headers.previous_block_header_id = ? AND blocks.in_active_chain = ?", currentId, true).
			Limit(1).Row().Scan(&nextId)

		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				break
			}

			return nil, err
		}

		if stopHash != nil && bytes.Equal(nextId, stopHash) {
			break
		}

		blocksIds.Add(nextId)
		currentId = nextId
	}

	return blocksIds, nil
}

func (repo *BlockRepository) GetActiveChainBlockLocator() (*structures.BlockLocator, error) {
	locator := structures.NewBlockLocator()

	var height uint64
	err := repo.db.Table("blocks").
		Select("height").
		Where("block_header_id = ?", repo.ActiveChainTipId).
		Pluck("height", &height).Error

	if err != nil {
		return nil, err
	}

	currentId := repo.ActiveChainTipId
	step := uint64(1)

	for {
		locator.Add(currentId)

		if height == 0 {
			break
		}

		if locator.Length() >= 10 {
			step *= 2
		}

		if step > height {
			height = 0
		} else {
			height = height - step
		}

		var prevId []byte
		err = repo.db.Table("blocks").
			Select("block_header_id").
			Where("in_active_chain = ?", true).
			Where("height = ?", height).
			Row().Scan(&prevId)

		if err != nil {
			return nil, err
		}

		currentId = slices.Clone(prevId)
	}

	return locator, nil
}

func (repo *BlockRepository) GetBlockLocator(startBlockId []byte) (*structures.BlockLocator, error) {
	locator := structures.NewBlockLocator()

	var height uint64
	err := repo.db.Table("blocks").
		Select("height").
		Where("block_header_id = ?", startBlockId).
		Pluck("height", &height).Error

	if err != nil {
		return nil, err
	}

	currentId := startBlockId
	step := uint64(1)

	for {
		locator.Add(currentId)

		if height == 0 {
			break
		}

		if locator.Length() >= 10 {
			step *= 2
		}

		toHeight := height - step
		if step > height {
			toHeight = 0
		}

		for height > toHeight {
			var prevId []byte

			err = repo.db.Table("block_headers").
				Select("previous_block_header_id").
				Where("id = ?", currentId).
				Row().Scan(&prevId)

			if err != nil {
				return nil, err
			}

			height--
			currentId = slices.Clone(prevId)
		}
	}

	return locator, nil
}

func (repo *BlockRepository) GetBlock(blockId []byte) (*models.Block, error) {
	var blockDB db_models.BlockDB
	err := repo.db.Preload("BlockHeader").
		Where("block_header_id = ?", blockId).
		Find(&blockDB).Error

	if err != nil {
		return nil, err
	}

	var txsDB []db_models.TransactionBlockDB
	err = repo.db.Preload("Transaction").
		Where("block_header_id = ?", blockId).
		Order("block_header_id, `order` ASC").
		Find(&txsDB).Error

	if err != nil {
		return nil, err
	}

	txs := make([]*models.Transaction, len(txsDB))
	for idx, txDB := range txsDB {
		txs[idx] = mapping.TransactionDBToTransaction(&txDB.Transaction)
	}

	block := &models.Block{
		Header:       *mapping.BlockHeaderDBToBlockHeader(&blockDB.BlockHeader),
		Transactions: txs,
	}

	return block, nil
}

func (repo *BlockRepository) GetBlocks(ids *structures.BytesSet) ([]*models.Block, error) {
	var blocksDB []db_models.BlockDB
	err := repo.db.Preload("BlockHeader").
		Where("block_header_id IN (?)", ids.ToBytesSlice()).
		Find(&blocksDB).Error

	if err != nil {
		return nil, err
	}

	var txBlocks []db_models.TransactionBlockDB
	err = repo.db.Preload("Transaction").
		Where("block_header_id IN (?)", ids.ToBytesSlice()).
		Order("block_header_id, `order` ASC").
		Find(&txBlocks).Error

	if err != nil {
		return nil, err
	}

	txsByBlock := make(map[string][]*models.Transaction)
	for _, tb := range txBlocks {
		blockIdStr := string(tb.BlockHeaderId)
		txsByBlock[blockIdStr] = append(txsByBlock[blockIdStr], mapping.TransactionDBToTransaction(&tb.Transaction))
	}

	blocks := make([]*models.Block, len(blocksDB))
	for i, bdb := range blocksDB {
		blocks[i] = &models.Block{
			Header:       *mapping.BlockHeaderDBToBlockHeader(&bdb.BlockHeader),
			Transactions: txsByBlock[string(bdb.BlockHeaderId)],
		}
	}

	return blocks, nil
}

func (repo *BlockRepository) GetMissingBlockIds(ids *structures.BytesSet) (*structures.BytesSet, error) {
	blockIds := ids.ToBytesSlice()

	if len(blockIds) == 0 {
		return nil, nil
	}

	var existingBlocks []db_models.BlockHeaderDB
	if err := repo.db.Where("id IN (?)", blockIds).Find(&existingBlocks).Error; err != nil {
		return nil, err
	}

	existingIds := structures.NewBytesSet()
	for _, blockHeader := range existingBlocks {
		existingIds.Add(blockHeader.Id)
	}

	missingIds := structures.NewBytesSet()

	for _, id := range blockIds {
		if !existingIds.Contains(id) {
			missingIds.Add(id)
		}
	}

	return missingIds, nil
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
		var count int64
		err := tx.Table("blocks").Where("block_header_id = ?", block.Header.Id).Count(&count).Error

		if err != nil {
			return err
		}

		if count > 0 {
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
		NBits:           difficulty.MINIMUM_DIFFICULTY,
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
