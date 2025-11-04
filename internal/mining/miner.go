package mining

import (
	"bytes"
	"log"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	hash "github.com/nivschuman/VotingBlockchain/internal/crypto/hash"
	repos "github.com/nivschuman/VotingBlockchain/internal/database/repositories"
	"github.com/nivschuman/VotingBlockchain/internal/difficulty"
	data_models "github.com/nivschuman/VotingBlockchain/internal/models"
)

type BlockHandler func(block *data_models.Block)

type Miner interface {
	AddHandler(blockHandler BlockHandler)
	Start()
	MineBlockTemplate(blockTemplate *data_models.Block)
	CreateBlockTemplate() (*data_models.Block, error)
	GetMiningStatistics() MiningStatistics
	Stop()
}

type MinerProperties struct {
	NodeVersion    int32
	MinerPublicKey []byte
}

type MinerImpl struct {
	properties MinerProperties

	blockRepository       repos.BlockRepository
	transactionRepository repos.TransactionRepository

	getNetworkTime func() int64

	handlers    []BlockHandler
	handlersMux sync.Mutex

	statistics MiningStatistics

	stopChannel chan bool
	stopOnce    sync.Once
	wg          sync.WaitGroup
}

func NewMinerImpl(getNetworkTime func() int64, blockRepository repos.BlockRepository, transactionRepository repos.TransactionRepository, minerProperties MinerProperties) *MinerImpl {
	return &MinerImpl{
		stopChannel:           make(chan bool),
		getNetworkTime:        getNetworkTime,
		blockRepository:       blockRepository,
		transactionRepository: transactionRepository,
		properties:            minerProperties,
	}
}

func (miner *MinerImpl) AddHandler(blockHandler BlockHandler) {
	miner.handlersMux.Lock()
	defer miner.handlersMux.Unlock()
	miner.handlers = append(miner.handlers, blockHandler)
}

func (miner *MinerImpl) Start() {
	miner.wg.Add(1)
	log.Printf("|Miner| Starting")

	go func() {
		defer miner.wg.Done()
		for {
			select {
			case <-miner.stopChannel:
				return
			default:
				blockTemplate, err := miner.CreateBlockTemplate()
				if err != nil {
					log.Printf("|Miner| Failed to create block template: %v", err)
					continue
				}

				miner.MineBlockTemplate(blockTemplate)
			}
		}
	}()
}

func (miner *MinerImpl) MineBlockTemplate(blockTemplate *data_models.Block) {
	medianPastTime, err := miner.blockRepository.GetMedianTimePast(blockTemplate.Header.PreviousBlockId, 11)
	if err != nil {
		log.Printf("|Miner| Failed to get median time past: %v", err)
		return
	}

	startTime := time.Now()
	atomic.StoreInt64(&miner.statistics.CurrentBlockStart, startTime.UnixNano())
	atomic.StoreUint32(&miner.statistics.CurrentNBits, blockTemplate.Header.NBits)
	atomic.StoreInt64(&miner.statistics.CurrentBlockHashesTried, 0)

	log.Printf("|Miner| Started mining block")
	blockTemplate.Header.Nonce = 0
	blockTemplate.Header.Timestamp = max(medianPastTime+1, miner.getNetworkTime())

	target := blockTemplate.Header.GetTarget()
	blockHeaderBytes := blockTemplate.Header.AsBytes()
	blockHeaderHash := hash.HashBytes(blockHeaderBytes)

	for !difficulty.IsHashBelowTarget(blockHeaderHash, target) {
		select {
		case <-miner.stopChannel:
			return
		default:
			blockTemplate.Header.Nonce++
			atomic.AddInt64(&miner.statistics.CurrentBlockHashesTried, 1)

			if blockTemplate.Header.Nonce&0x3ffff == 0 {
				if !bytes.Equal(blockTemplate.Header.PreviousBlockId, miner.blockRepository.GetActiveChainTipId()) {
					return
				}
				blockTemplate.Header.Timestamp = max(medianPastTime+1, miner.getNetworkTime())
			}

			data_models.UpdateBlockHeaderBytes(blockHeaderBytes, blockTemplate.Header.Timestamp, blockTemplate.Header.Nonce)
			blockHeaderHash = hash.HashBytes(blockHeaderBytes)
		}
	}

	blockTemplate.Header.SetId()
	duration := time.Since(startTime)

	log.Printf("|Miner| Mined block %x in %s", blockTemplate.Header.Id, duration)

	atomic.AddInt64(&miner.statistics.TotalBlocksMined, 1)
	atomic.StoreInt64(&miner.statistics.LastNonce, int64(blockTemplate.Header.Nonce))
	atomic.StoreInt64(&miner.statistics.LastBlockTimeNs, duration.Nanoseconds())

	for _, handler := range miner.handlers {
		handler(blockTemplate)
	}
}

func (miner *MinerImpl) CreateBlockTemplate() (*data_models.Block, error) {
	activeChainTipId := slices.Clone(miner.blockRepository.GetActiveChainTipId())

	txs, err := miner.transactionRepository.GetMempool(10)
	if err != nil {
		return nil, err
	}

	nbits, err := miner.blockRepository.GetNextWorkRequired(activeChainTipId)
	if err != nil {
		return nil, err
	}

	merkleRoot := data_models.TransactionsMerkleRoot(txs)

	templateHeader := data_models.BlockHeader{
		Version:         miner.properties.NodeVersion,
		PreviousBlockId: activeChainTipId,
		MerkleRoot:      merkleRoot,
		NBits:           nbits,
		MinerPublicKey:  miner.properties.MinerPublicKey,
	}

	template := &data_models.Block{
		Header:       templateHeader,
		Transactions: txs,
	}

	return template, nil
}

func (miner *MinerImpl) Stop() {
	miner.stopOnce.Do(func() {
		log.Printf("|Miner| Stopping")
		close(miner.stopChannel)
	})
	miner.wg.Wait()
}

func (miner *MinerImpl) GetMiningStatistics() MiningStatistics {
	return MiningStatistics{
		TotalBlocksMined:        atomic.LoadInt64(&miner.statistics.TotalBlocksMined),
		CurrentBlockHashesTried: atomic.LoadInt64(&miner.statistics.CurrentBlockHashesTried),
		LastNonce:               atomic.LoadInt64(&miner.statistics.LastNonce),
		LastBlockTimeNs:         atomic.LoadInt64(&miner.statistics.LastBlockTimeNs),
		CurrentNBits:            atomic.LoadUint32(&miner.statistics.CurrentNBits),
		CurrentBlockStart:       atomic.LoadInt64(&miner.statistics.CurrentBlockStart),
	}
}
