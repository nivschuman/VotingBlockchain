package mining

import (
	"bytes"
	"log"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	config "github.com/nivschuman/VotingBlockchain/internal/config"
	repos "github.com/nivschuman/VotingBlockchain/internal/database/repositories"
	data_models "github.com/nivschuman/VotingBlockchain/internal/models"
)

type BlockHandler func(block *data_models.Block)

type Miner struct {
	networkTime *atomic.Int64

	handlers    []BlockHandler
	handlersMux sync.Mutex

	stopChannel chan bool
	stopOnce    sync.Once
	wg          sync.WaitGroup
}

func NewMiner(networkTime *atomic.Int64) *Miner {
	return &Miner{
		stopChannel: make(chan bool),
		networkTime: networkTime,
	}
}

func (miner *Miner) AddHandler(blockHandler BlockHandler) {
	miner.handlersMux.Lock()
	defer miner.handlersMux.Unlock()
	miner.handlers = append(miner.handlers, blockHandler)
}

func (miner *Miner) Start() {
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

func (miner *Miner) MineBlockTemplate(blockTemplate *data_models.Block) {
	medianPastTime, err := repos.GlobalBlockRepository.GetMedianTimePast(blockTemplate.Header.PreviousBlockId, 11)
	if err != nil {
		log.Printf("|Miner| Failed to get median time past: %v", err)
		return
	}

	startTime := time.Now()
	lastLogTime := startTime
	lastLogNonce := uint64(0)

	log.Printf("|Miner| Started mining block")
	blockTemplate.Header.Nonce = 0
	blockTemplate.Header.Timestamp = max(medianPastTime+1, miner.networkTime.Load())
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
				blockTemplate.Header.Timestamp = max(medianPastTime+1, miner.networkTime.Load())
			}

			if !bytes.Equal(blockTemplate.Header.PreviousBlockId, repos.GlobalBlockRepository.ActiveChainTipId) {
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

func (miner *Miner) CreateBlockTemplate() (*data_models.Block, error) {
	activeChainTipId := slices.Clone(repos.GlobalBlockRepository.ActiveChainTipId)

	txs, err := repos.GlobalTransactionRepository.GetMempool(10)
	if err != nil {
		return nil, err
	}

	nbits, err := repos.GlobalBlockRepository.GetNextWorkRequired(activeChainTipId)
	if err != nil {
		return nil, err
	}

	merkleRoot := data_models.TransactionsMerkleRoot(txs)

	templateHeader := data_models.BlockHeader{
		Version:         config.GlobalConfig.NodeConfig.Version,
		PreviousBlockId: activeChainTipId,
		MerkleRoot:      merkleRoot,
		NBits:           nbits,
		MinerPublicKey:  config.GlobalConfig.MinerConfig.PublicKey,
	}

	template := &data_models.Block{
		Header:       templateHeader,
		Transactions: txs,
	}

	return template, nil
}

func (miner *Miner) Stop() {
	miner.stopOnce.Do(func() {
		log.Printf("|Miner| Stopping")
		close(miner.stopChannel)
	})
	miner.wg.Wait()
}
