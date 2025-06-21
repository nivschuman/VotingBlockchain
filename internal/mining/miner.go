package mining

import (
	"bytes"
	"log"
	"slices"
	"sync"
	"time"

	config "github.com/nivschuman/VotingBlockchain/internal/config"
	repos "github.com/nivschuman/VotingBlockchain/internal/database/repositories"
	data_models "github.com/nivschuman/VotingBlockchain/internal/models"
)

type BlockHandler func(block *data_models.Block)

type Miner struct {
	NetworkTime int64

	handlers    []BlockHandler
	handlersMux sync.Mutex

	stopChannel chan bool
	stopOnce    sync.Once
}

func NewMiner() *Miner {
	return &Miner{
		stopChannel: make(chan bool),
	}
}

func (miner *Miner) AddHandler(blockHandler BlockHandler) {
	miner.handlersMux.Lock()
	defer miner.handlersMux.Unlock()
	miner.handlers = append(miner.handlers, blockHandler)
}

func (miner *Miner) StartMiner(wg *sync.WaitGroup) {
	wg.Add(1)
	log.Printf("Started miner")

	go func() {
		defer wg.Done()
		for {
			select {
			case <-miner.stopChannel:
				return
			default:
				blockTemplate, err := miner.CreateBlockTemplate()
				if err != nil {
					log.Printf("Failed to create block template: %v", err)
				}

				miner.MineBlockTemplate(blockTemplate)
			}
		}
	}()
}

func (miner *Miner) MineBlockTemplate(blockTemplate *data_models.Block) {
	medianPastTime, err := repos.GlobalBlockRepository.GetMedianTimePast(blockTemplate.Header.PreviousBlockId, 11)
	if err != nil {
		log.Printf("Failed to get median time past: %v", err)
		return
	}

	startTime := time.Now()
	log.Printf("Started mining block")

	blockTemplate.Header.Nonce = 0
	blockTemplate.Header.Timestamp = max(medianPastTime+1, miner.NetworkTime)
	for !blockTemplate.Header.IsHashBelowTarget() {
		select {
		case <-miner.stopChannel:
			return
		default:
			blockTemplate.Header.Nonce++

			if blockTemplate.Header.Nonce&0x3ffff == 0 {
				blockTemplate.Header.Timestamp = max(medianPastTime+1, miner.NetworkTime)
			}

			if !bytes.Equal(blockTemplate.Header.PreviousBlockId, repos.GlobalBlockRepository.ActiveChainTipId) {
				return
			}
		}
	}

	blockTemplate.Header.SetId()

	duration := time.Since(startTime)
	log.Printf("Mined block %x in %s", blockTemplate.Header.Id, duration)

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

	//TBD calculate merkle root
	merkleRoot := make([]byte, 32)

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

func (miner *Miner) StopMiner() {
	miner.stopOnce.Do(func() {
		log.Printf("Stopped miner")
		close(miner.stopChannel)
	})
}
