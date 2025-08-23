package nodes

import (
	"bytes"
	"log"
	"sync"

	repos "github.com/nivschuman/VotingBlockchain/internal/database/repositories"
	difficulty "github.com/nivschuman/VotingBlockchain/internal/difficulty"
	mining "github.com/nivschuman/VotingBlockchain/internal/mining"
	data_models "github.com/nivschuman/VotingBlockchain/internal/models"
	models "github.com/nivschuman/VotingBlockchain/internal/networking/models"
	network "github.com/nivschuman/VotingBlockchain/internal/networking/network"
	peer "github.com/nivschuman/VotingBlockchain/internal/networking/peer"
	structures "github.com/nivschuman/VotingBlockchain/internal/structures"
)

type FullNode struct {
	network network.Network
	miner   mining.Miner

	blockRepository       repos.BlockRepository
	transactionRepository repos.TransactionRepository

	governmentPublicKey []byte

	orphanBlocks      *structures.BytesMap[*data_models.Block]
	orphanBlocksMutex sync.RWMutex

	shutdownHooks      []func() error
	shutdownHooksMutex sync.Mutex

	criticalMutex sync.Mutex
}

func NewFullNode(
	network network.Network,
	miner mining.Miner,
	blockRepository repos.BlockRepository,
	transactionRepository repos.TransactionRepository,
	governmentPublicKey []byte) *FullNode {
	fullNode := &FullNode{
		network:               network,
		miner:                 miner,
		blockRepository:       blockRepository,
		transactionRepository: transactionRepository,
		orphanBlocks:          structures.NewBytesMap[*data_models.Block](),
		governmentPublicKey:   governmentPublicKey,
		shutdownHooks:         make([]func() error, 0),
	}

	fullNode.network.AddCommandHandler(models.CommandGetBlocks, fullNode.processGetBlocks)
	fullNode.network.AddCommandHandler(models.CommandMemPool, fullNode.processMemPool)
	fullNode.network.AddCommandHandler(models.CommandTx, fullNode.processTx)
	fullNode.network.AddCommandHandler(models.CommandGetData, fullNode.processGetData)
	fullNode.network.AddCommandHandler(models.CommandInv, fullNode.processInv)
	fullNode.network.AddCommandHandler(models.CommandBlock, fullNode.processBlock)

	fullNode.miner.AddHandler(fullNode.processMinedBlock)

	return fullNode
}

func (fullNode *FullNode) Start() {
	log.Print("|Node| Starting full node")
	fullNode.network.Start()
	fullNode.miner.Start()
}

func (fullNode *FullNode) Stop() {
	fullNode.shutdownHooksMutex.Lock()
	defer fullNode.shutdownHooksMutex.Unlock()

	log.Print("|Node| Stopping full node")
	fullNode.network.Stop()
	fullNode.miner.Stop()

	for _, hook := range fullNode.shutdownHooks {
		if err := hook(); err != nil {
			log.Printf("shutdown hook error: %v", err)
		}
	}
}

func (fullNode *FullNode) AddShutdownHook(hook func() error) {
	fullNode.shutdownHooksMutex.Lock()
	defer fullNode.shutdownHooksMutex.Unlock()

	fullNode.shutdownHooks = append(fullNode.shutdownHooks, hook)
}

func (fullNode *FullNode) GetMiner() mining.Miner {
	return fullNode.miner
}

func (fullNode *FullNode) GetNetwork() network.Network {
	return fullNode.network
}

func (fullNode *FullNode) GetBlockRepository() repos.BlockRepository {
	return fullNode.blockRepository
}

func (fullNode *FullNode) GetTransactionRepository() repos.TransactionRepository {
	return fullNode.transactionRepository
}

func (fullNode *FullNode) ProcessGeneratedTransaction(transaction *data_models.Transaction) {
	valid, err := transaction.IsValid(fullNode.governmentPublicKey)

	if err != nil {
		log.Printf("|Node| Failed to validate generated transaction: %v", err)
		return
	}

	if !valid {
		log.Printf("|Node| Received invalid generated transaction from")
		return
	}

	fullNode.criticalMutex.Lock()
	valid, err = fullNode.transactionRepository.TransactionValidInActiveChain(transaction)
	fullNode.criticalMutex.Unlock()

	if err != nil {
		log.Printf("|Node| Failed validating generated transaction: %v", err)
		return
	}

	if !valid {
		log.Printf("|Node| Received invalid generated transaction")
		return
	}

	err = fullNode.transactionRepository.InsertIfNotExists(transaction)

	if err != nil {
		log.Printf("|Node| Failed to insert generated transaction: %v", err)
	}

	fullNode.network.BroadcastItemToPeers(models.MSG_TX, transaction.Id, nil)
}

func (fullNode *FullNode) processInv(fromPeer *peer.Peer, message *models.Message) {
	inv, err := models.InvFromBytes(message.Payload)

	if err != nil {
		log.Printf("|Node| Failed to parse inv from %s: %v", fromPeer.String(), err)
		return
	}

	getData := models.NewGetData()

	blockHashes := structures.NewBytesSet()
	txHashes := structures.NewBytesSet()

	for _, invItem := range inv.Items {
		switch invItem.Type {
		case models.MSG_BLOCK:
			blockHashes.Add(invItem.Hash)
		case models.MSG_TX:
			txHashes.Add(invItem.Hash)
		}
	}

	missingTransactions, err := fullNode.transactionRepository.GetMissingTransactionIds(txHashes)

	if err != nil {
		log.Printf("|Node| Failed to get missing transactions : %v", err)
		return
	}

	missingBlocks, err := fullNode.blockRepository.GetMissingBlockIds(blockHashes)

	if err != nil {
		log.Printf("|Node| Failed to get missing blocks : %v", err)
		return
	}

	for _, id := range missingTransactions.ToBytesSlice() {
		getData.AddItem(models.MSG_TX, id)
	}

	for _, id := range missingBlocks.ToBytesSlice() {
		getData.AddItem(models.MSG_BLOCK, id)
	}

	getDataMessage, err := models.NewGetDataMessage(getData)

	if err != nil {
		log.Printf("|Node| Failed to make get data message for %s : %v", fromPeer.String(), err)
		return
	}

	select {
	case <-fromPeer.StopChannel:
		return
	case fromPeer.SendChannel <- *getDataMessage:
	}
}

func (fullNode *FullNode) processTx(fromPeer *peer.Peer, message *models.Message) {
	transaction, err := data_models.TransactionFromBytes(message.Payload)

	if err != nil {
		log.Printf("|Node| Failed to parse transaction from %s: %v", fromPeer.String(), err)
		return
	}

	valid, err := transaction.IsValid(fullNode.governmentPublicKey)

	if err != nil {
		log.Printf("|Node| Failed to validate transaction from %s: %v", fromPeer.String(), err)
		return
	}

	if !valid {
		log.Printf("|Node| Received invalid transaction from %s", fromPeer.String())
		return
	}

	fullNode.criticalMutex.Lock()
	valid, err = fullNode.transactionRepository.TransactionValidInActiveChain(transaction)
	fullNode.criticalMutex.Unlock()

	if err != nil {
		log.Printf("|Node| Failed validating transaction from %s: %v", fromPeer.String(), err)
		return
	}

	if !valid {
		log.Printf("|Node| Received invalid transaction from %s", fromPeer.String())
		return
	}

	err = fullNode.transactionRepository.InsertIfNotExists(transaction)

	if err != nil {
		log.Printf("|Node| Failed to insert transaction from %s: %v", fromPeer.String(), err)
	}

	fullNode.network.BroadcastItemToPeers(models.MSG_TX, transaction.Id, fromPeer)
}

func (fullNode *FullNode) processMemPool(fromPeer *peer.Peer, _ *models.Message) {
	transactions, err := fullNode.transactionRepository.GetMempool(10)

	if err != nil {
		log.Printf("|Node| Failed to get mempool for %s: %v", fromPeer.String(), err)
		return
	}

	inv := models.NewInv()

	for _, tx := range transactions {
		inv.AddItem(models.MSG_TX, tx.Id)
	}

	mempoolMessage, err := models.NewInvMessage(inv)

	if err != nil {
		log.Printf("|Node| Failed to create mempool inv message for %s: %v", fromPeer.String(), err)
		return
	}

	select {
	case <-fromPeer.StopChannel:
		return
	case fromPeer.SendChannel <- *mempoolMessage:
	}
}

func (fullNode *FullNode) processGetData(fromPeer *peer.Peer, message *models.Message) {
	getData, err := models.GetDataFromBytes(message.Payload)

	if err != nil {
		log.Printf("|Node| Failed to parse getdata from %s: %v", fromPeer.String(), err)
		return
	}

	blockHashes := structures.NewBytesSet()
	txHashes := structures.NewBytesSet()

	for _, item := range getData.Items() {
		switch item.Type {
		case models.MSG_BLOCK:
			blockHashes.Add(item.Hash)
		case models.MSG_TX:
			txHashes.Add(item.Hash)
		}
	}

	transactions, err := fullNode.transactionRepository.GetTransactions(txHashes)

	if err != nil {
		log.Printf("|Node| Failed to get transactions : %v", err)
		return
	}

	blocks, err := fullNode.blockRepository.GetBlocks(blockHashes)

	if err != nil {
		log.Printf("|Node| Failed to get blocks : %v", err)
		return
	}

	for _, tx := range transactions {
		msg := models.NewMessage(models.CommandTx, tx.AsBytes())

		select {
		case <-fromPeer.StopChannel:
			return
		case fromPeer.SendChannel <- *msg:
		}
	}

	for _, block := range blocks {
		msg := models.NewMessage(models.CommandBlock, block.AsBytes())

		select {
		case <-fromPeer.StopChannel:
			return
		case fromPeer.SendChannel <- *msg:
		}
	}
}

func (fullNode *FullNode) processGetBlocks(fromPeer *peer.Peer, message *models.Message) {
	getBlocks, err := models.GetBlocksFromBytes(message.Payload)

	if err != nil {
		log.Printf("|Node| Failed to parse getblocks from %s: %v", fromPeer.String(), err)
		return
	}

	blocksIds, err := fullNode.blockRepository.GetNextBlocksIds(getBlocks.BlockLocator, getBlocks.StopHash, 500)

	if err != nil {
		log.Printf("|Node| Failed to parse get next blocks for %s: %v", fromPeer.String(), err)
		return
	}

	inv := models.NewInv()

	for _, blockId := range blocksIds.ToBytesSlice() {
		inv.AddItem(models.MSG_BLOCK, blockId)
	}

	invMessage, err := models.NewInvMessage(inv)

	if err != nil {
		log.Printf("|Node| Failed to create get blocks inv message for %s: %v", fromPeer.String(), err)
		return
	}

	select {
	case <-fromPeer.StopChannel:
		return
	case fromPeer.SendChannel <- *invMessage:
	}
}

func (fullNode *FullNode) processBlock(fromPeer *peer.Peer, message *models.Message) {
	//Parse block
	block, err := data_models.BlockFromBytes(message.Payload)

	if err != nil {
		log.Printf("|Node| Failed to parse block from %s: %v", fromPeer.String(), err)
		return
	}

	//Check if already have block
	fullNode.orphanBlocksMutex.RLock()
	haveOrphan := fullNode.orphanBlocks.ContainsKey(block.Header.Id)
	fullNode.orphanBlocksMutex.RUnlock()

	if haveOrphan {
		return
	}

	exists, err := fullNode.blockRepository.HaveBlock(block.Header.Id)

	if err != nil {
		log.Printf("|Node| Failed to check if block exists from %s is orphan: %v", fromPeer.String(), err)
		return
	}

	if exists {
		return
	}

	//Check if block is orphan
	isOrphan, err := fullNode.blockRepository.BlockIsOrphan(block)

	if err != nil {
		log.Printf("|Node| Failed to check if block from %s is orphan: %v", fromPeer.String(), err)
	}

	//Check block
	isValid, err := fullNode.checkBlock(block)

	if err != nil {
		log.Printf("|Node| Failed to check block from %s: %v", fromPeer.String(), err)
		return
	}

	if !isValid {
		log.Printf("|Node| Received invalid block from %s", fromPeer.String())
		return
	}

	//No further processing for orphan
	if isOrphan {
		fullNode.orphanBlocksMutex.Lock()
		fullNode.orphanBlocks.Put(block.Header.Id, block)
		fullNode.orphanBlocksMutex.Unlock()

		//Ask for block
		fullNode.criticalMutex.Lock()
		blockLocator, err := fullNode.blockRepository.GetActiveChainBlockLocator()
		fullNode.criticalMutex.Unlock()

		if err != nil {
			log.Printf("|Node| Failed to check active chain block locator for %s: %v", fromPeer.String(), err)
			return
		}

		fullNode.orphanBlocksMutex.RLock()
		orphanRoot := fullNode.getOrphanRoot(block)
		fullNode.orphanBlocksMutex.RUnlock()

		getBlocks := models.NewGetBlocks(blockLocator, orphanRoot.Header.Id)

		msg, err := models.NewGetBlocksMessage(getBlocks)
		if err != nil {
			log.Printf("|Node| Failed to make get blocks message for %s: %v", fromPeer.String(), err)
			return
		}

		select {
		case <-fromPeer.StopChannel:
			return
		case fromPeer.SendChannel <- *msg:
		}

		return
	}

	//Validate block
	fullNode.criticalMutex.Lock()
	defer fullNode.criticalMutex.Unlock()

	isValid, err = fullNode.validateBlock(block)

	if err != nil {
		log.Printf("|Node| Failed to validate block from %s: %v", fromPeer.String(), err)
		return
	}

	if !isValid {
		log.Printf("|Node| Received invalid block from %s", fromPeer.String())
		return
	}

	//Insert block
	err = fullNode.blockRepository.InsertIfNotExists(block)

	if err != nil {
		log.Printf("|Node| Failed to insert block from %s: %v", fromPeer.String(), err)
		return
	}

	//Send block to peers
	fullNode.network.BroadcastItemToPeers(models.MSG_BLOCK, block.Header.Id, fromPeer)

	//Process dependent orphans recursively
	fullNode.orphanBlocksMutex.Lock()
	defer fullNode.orphanBlocksMutex.Unlock()

	queue := []*data_models.Block{block}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		children := fullNode.getConnectedOrphans(current.Header.Id)

		for _, child := range children {
			isValid, err := fullNode.validateBlock(child)

			if err != nil {
				log.Printf("|Node| Failed to validate orphan child %x: %v", child.Header.Id, err)
				continue
			}

			if !isValid {
				log.Printf("|Node| Invalid orphan child block %x", child.Header.Id)
				continue
			}

			err = fullNode.blockRepository.InsertIfNotExists(child)

			if err != nil {
				log.Printf("|Node| Failed to insert orphan child block %x: %v", child.Header.Id, err)
				continue
			}

			fullNode.network.BroadcastItemToPeers(models.MSG_BLOCK, child.Header.Id, fromPeer)

			fullNode.orphanBlocks.Remove(child.Header.Id)
			queue = append(queue, child)
		}
	}
}

func (fullNode *FullNode) checkBlock(block *data_models.Block) (bool, error) {
	//Timestamp must be less than the network adjusted time +2 hours.
	if block.Header.Timestamp > fullNode.network.GetNetworkTime()+2*60*60 {
		log.Printf("|Node| Invalid block %x: timestamp too far in the future (%d)", block.Header.Id, block.Header.Timestamp)
		return false, nil
	}

	//Proof of Work must be valid
	if !block.Header.IsHashBelowTarget() {
		log.Printf("|Node| Invalid block %x: hash does not satisfy target", block.Header.Id)
		return false, nil
	}

	//NBits must not be below minimum work
	target := difficulty.GetTargetFromNBits(block.Header.NBits)
	if target.Cmp(difficulty.GetTargetFromNBits(difficulty.MINIMUM_DIFFICULTY)) > 0 {
		log.Printf("|Node| Invalid block %x: NBits below minimum difficulty (%d)", block.Header.Id, block.Header.NBits)
		return false, nil
	}

	//Block transactions must be valid
	for i, tx := range block.Transactions {
		valid, err := tx.IsValid(fullNode.governmentPublicKey)

		if err != nil {
			return false, err
		}

		if !valid {
			log.Printf("|Node| Block %x: transaction %d is invalid", block.Header.Id, i)
			return false, nil
		}
	}

	//Check merkle root
	merkleRoot := data_models.TransactionsMerkleRoot(block.Transactions)
	if !bytes.Equal(block.Header.MerkleRoot, merkleRoot) {
		log.Printf("|Node| Block %x: merkle root mismatch", block.Header.Id)
		return false, nil
	}

	return true, nil
}

func (fullNode *FullNode) validateBlock(block *data_models.Block) (bool, error) {
	//Timestamp must be greater than the median time of the last 11 blocks
	medianTimePast, err := fullNode.blockRepository.GetMedianTimePast(block.Header.PreviousBlockId, 11)
	if err != nil {
		return false, err
	}

	if block.Header.Timestamp < medianTimePast {
		log.Printf("|Node| Block %x: timestamp %d < median past %d", block.Header.Id, block.Header.Timestamp, medianTimePast)
		return false, nil
	}

	//Validate transactions on this blocks chain
	valid, err := fullNode.transactionRepository.TransactionsValidInChain(block.Header.PreviousBlockId, block.Transactions)
	if err != nil {
		return false, err
	}

	if !valid {
		log.Printf("|Node| Block %x: transactions invalid in chain", block.Header.Id)
		return false, nil
	}

	//Validate work
	requiredNBits, err := fullNode.blockRepository.GetNextWorkRequired(block.Header.PreviousBlockId)
	if err != nil {
		return false, err
	}

	if block.Header.NBits != requiredNBits {
		log.Printf("|Node| Block %x: NBits %d != required %d", block.Header.Id, block.Header.NBits, requiredNBits)
		return false, nil
	}

	return true, nil
}

func (fullNode *FullNode) getConnectedOrphans(blockId []byte) []*data_models.Block {
	blocks := make([]*data_models.Block, 0)
	for _, orphanBlock := range fullNode.orphanBlocks.Values() {
		if bytes.Equal(orphanBlock.Header.PreviousBlockId, blockId) {
			blocks = append(blocks, orphanBlock)
		}
	}

	return blocks
}

func (fullNode *FullNode) getOrphanRoot(orphanBlock *data_models.Block) *data_models.Block {
	for {
		prevId := orphanBlock.Header.PreviousBlockId

		orphanParent, exists := fullNode.orphanBlocks.Get(prevId)
		if !exists {
			break
		}

		orphanBlock = orphanParent
	}

	return orphanBlock
}

func (fullNode *FullNode) processMinedBlock(block *data_models.Block) {
	//Check block
	isValid, err := fullNode.checkBlock(block)

	if err != nil {
		log.Printf("|Node| Failed to check block mined block: %v", err)
		return
	}

	if !isValid {
		log.Print("|Node| Received invalid block from miner")
		return
	}

	//Validate block
	fullNode.criticalMutex.Lock()
	defer fullNode.criticalMutex.Unlock()

	isValid, err = fullNode.validateBlock(block)

	if err != nil {
		log.Printf("|Node| Failed to validate block miner: %v", err)
		return
	}

	if !isValid {
		log.Print("|Node| Received invalid block from miner")
		return
	}

	//Insert block
	err = fullNode.blockRepository.InsertIfNotExists(block)

	if err != nil {
		log.Printf("|Node| Failed to insert block from miner: %v", err)
		return
	}

	//Send block to peers
	fullNode.network.BroadcastItemToPeers(models.MSG_BLOCK, block.Header.Id, nil)
}
