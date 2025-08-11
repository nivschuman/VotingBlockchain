package mining

import (
	"bytes"
	"log"
	"slices"
	"sync"
	"time"

	repos "github.com/nivschuman/VotingBlockchain/internal/database/repositories"
	data_models "github.com/nivschuman/VotingBlockchain/internal/models"
)

type BlockHandler func(block *data_models.Block)

type Miner interface {
	AddHandler(blockHandler BlockHandler)
	Start()
	MineBlockTemplate(blockTemplate *data_models.Block)
	CreateBlockTemplate() (*data_models.Block, error)
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

	stopChannel chan bool
	stopOnce    sync.Once
	wg          sync.WaitGroup
}

func NewMinerImpl(
	getNetworkTime func() int64,
	blockRepository repos.BlockRepository,
	transactionRepository repos.TransactionRepository,
	minerProperties MinerProperties) *MinerImpl {
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
	lastLogTime := startTime
	lastLogNonce := uint64(0)

	log.Printf("|Miner| Started mining block")
	blockTemplate.Header.Nonce = 0
	blockTemplate.Header.Timestamp = max(medianPastTime+1, miner.getNetworkTime())
	for !blockTemplate.Header.IsHashBelowTarget() {
		select {
		case <-miner.stopChannel:
			return
		default:
			blockTemplate.Header.Nonce++

			if time.Since(lastLogTime) > 10*time.Minute {
				elapsed := time.Since(startTime).Seconds()
				hashes := blockTemplate.Header.Nonce - lastLogNonce
				hashRate := float64(hashes) / time.Since(lastLogTime).Seconds()

				log.Printf("|Miner| Mining... nonce=%d, %.2f hashes/sec, elapsed=%.0fs", blockTemplate.Header.Nonce, hashRate, elapsed)
				lastLogTime = time.Now()
				lastLogNonce = blockTemplate.Header.Nonce
			}

			if blockTemplate.Header.Nonce&0x3ffff == 0 {
				blockTemplate.Header.Timestamp = max(medianPastTime+1, miner.getNetworkTime())
			}

			if !bytes.Equal(blockTemplate.Header.PreviousBlockId, miner.blockRepository.GetActiveChainTipId()) {
				return
			}
		}
	}

	blockTemplate.Header.SetId()

	duration := time.Since(startTime)
	log.Printf("|Miner| Mined block %x in %s", blockTemplate.Header.Id, duration)

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
