package nodes

import (
	"bytes"
	"log"
	"sync"

	config "github.com/nivschuman/VotingBlockchain/internal/config"
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
	network *network.Network
	miner   *mining.Miner
	mine    bool

	orphanBlocks      *structures.BytesMap[*data_models.Block]
	orphanBlocksMutex sync.RWMutex
}

func NewFullNode(mine bool) *FullNode {
	fullNode := &FullNode{}

	ip := config.GlobalConfig.NetworkConfig.Ip
	port := config.GlobalConfig.NetworkConfig.Port
	fullNode.network = network.NewNetwork(ip, port)
	fullNode.network.AddPeerEventHandler(fullNode.handlePeerEvent)
	fullNode.network.AddCommandHandler(models.CommandGetBlocks, fullNode.processGetBlocks)
	fullNode.network.AddCommandHandler(models.CommandMemPool, fullNode.processMemPool)
	fullNode.network.AddCommandHandler(models.CommandTx, fullNode.processTx)
	fullNode.network.AddCommandHandler(models.CommandGetData, fullNode.processGetData)
	fullNode.network.AddCommandHandler(models.CommandInv, fullNode.processInv)
	fullNode.network.AddCommandHandler(models.CommandBlock, fullNode.processBlock)

	fullNode.miner = mining.NewMiner()
	fullNode.miner.AddHandler(fullNode.processMinedBlock)
	fullNode.mine = mine

	fullNode.orphanBlocks = structures.NewBytesMap[*data_models.Block]()

	return fullNode
}

func (fullNode *FullNode) Start() {
	log.Print("|Node| Starting full node")
	fullNode.network.Start()
	if fullNode.mine {
		fullNode.miner.Start()
	}
}

func (fullNode *FullNode) Stop() {
	log.Print("|Node| Stopping full node")
	fullNode.network.Stop()
	if fullNode.mine {
		fullNode.miner.Stop()
	}
}

func (fullNode *FullNode) handlePeerEvent(eventType network.PeerEventType, peer *peer.Peer) {
	switch eventType {
	case network.PeerConnected:
		if fullNode.miner != nil && fullNode.network != nil {
			fullNode.miner.NetworkTime = fullNode.network.NetworkTime()
		}
	}
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

	missingTransactions, err := repos.GlobalTransactionRepository.GetMissingTransactionIds(txHashes)

	if err != nil {
		log.Printf("|Node| Failed to get missing transactions : %v", err)
		return
	}

	missingBlocks, err := repos.GlobalBlockRepository.GetMissingBlockIds(blockHashes)

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

	valid, err := transaction.IsValid()

	if err != nil {
		log.Printf("|Node| Failed to validate transaction from %s: %v", fromPeer.String(), err)
		return
	}

	if !valid {
		log.Printf("|Node| Received invalid transaction from %s", fromPeer.String())
		return
	}

	valid, err = repos.GlobalTransactionRepository.TransactionValidInActiveChain(transaction)

	if err != nil {
		log.Printf("|Node| Failed validating transaction from %s: %v", fromPeer.String(), err)
		return
	}

	if !valid {
		log.Printf("|Node| Received invalid transaction from %s", fromPeer.String())
		return
	}

	err = repos.GlobalTransactionRepository.InsertIfNotExists(transaction)

	if err != nil {
		log.Printf("|Node| Failed to insert transaction from %s: %v", fromPeer.String(), err)
	}

	fullNode.network.PeersMutex.RLock()
	for _, peer := range fullNode.network.Peers {
		if peer != fromPeer {
			peer.InventoryToSendMutex.Lock()
			peer.InventoryToSend.AddItem(models.MSG_TX, transaction.Id)
			peer.InventoryToSendMutex.Unlock()
		}
	}
	fullNode.network.PeersMutex.RUnlock()
}

func (fullNode *FullNode) processMemPool(fromPeer *peer.Peer, _ *models.Message) {
	transactions, err := repos.GlobalTransactionRepository.GetMempool(10)

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

	transactions, err := repos.GlobalTransactionRepository.GetTransactions(txHashes)

	if err != nil {
		log.Printf("|Node| Failed to get transactions : %v", err)
		return
	}

	blocks, err := repos.GlobalBlockRepository.GetBlocks(blockHashes)

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

	blocksIds, err := repos.GlobalBlockRepository.GetNextBlocksIds(getBlocks.BlockLocator, getBlocks.StopHash, 500)

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
	if fullNode.orphanBlocks.ContainsKey(block.Header.Id) {
		return
	}

	exists, err := repos.GlobalBlockRepository.HaveBlock(block.Header.Id)

	if err != nil {
		log.Printf("|Node| Failed to check if block exists from %s is orphan: %v", fromPeer.String(), err)
		return
	}

	if exists {
		return
	}

	//Check if block is orphan
	isOrphan, err := repos.GlobalBlockRepository.BlockIsOrphan(block)

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
		blockLocator, err := repos.GlobalBlockRepository.GetActiveChainBlockLocator()

		if err != nil {
			log.Printf("|Node| Failed to check active chain block locator for %s: %v", fromPeer.String(), err)
			return
		}

		orphanRoot := fullNode.getOrphanRoot(block)
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
	err = repos.GlobalBlockRepository.InsertIfNotExists(block)

	if err != nil {
		log.Printf("|Node| Failed to insert block from %s: %v", fromPeer.String(), err)
		return
	}

	//Send block to peers
	fullNode.network.PeersMutex.RLock()
	for _, peer := range fullNode.network.Peers {
		if peer != fromPeer {
			peer.InventoryToSendMutex.Lock()
			peer.InventoryToSend.AddItem(models.MSG_BLOCK, block.Header.Id)
			peer.InventoryToSendMutex.Unlock()
		}
	}
	fullNode.network.PeersMutex.RUnlock()

	//Process dependent orphans recursively
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

			err = repos.GlobalBlockRepository.InsertIfNotExists(child)

			if err != nil {
				log.Printf("|Node| Failed to insert orphan child block %x: %v", child.Header.Id, err)
				continue
			}

			fullNode.network.PeersMutex.RLock()
			for _, peer := range fullNode.network.Peers {
				if peer != fromPeer {
					peer.InventoryToSendMutex.Lock()
					peer.InventoryToSend.AddItem(models.MSG_BLOCK, child.Header.Id)
					peer.InventoryToSendMutex.Unlock()
				}
			}
			fullNode.network.PeersMutex.RUnlock()

			fullNode.orphanBlocksMutex.Lock()
			fullNode.orphanBlocks.Remove(child.Header.Id)
			fullNode.orphanBlocksMutex.Unlock()

			queue = append(queue, child)
		}
	}
}

func (fullNode *FullNode) checkBlock(block *data_models.Block) (bool, error) {
	//Timestamp must be less than the network adjusted time +2 hours.
	if block.Header.Timestamp > fullNode.network.NetworkTime()+2*60*60 {
		return false, nil
	}

	//Proof of Work must be valid
	if !block.Header.IsHashBelowTarget() {
		return false, nil
	}

	//NBits must not be below minimum work
	target := difficulty.GetTargetFromNBits(block.Header.NBits)
	if target.Cmp(difficulty.GetTargetFromNBits(difficulty.MINIMUM_DIFFICULTY)) > 0 {
		return false, nil
	}

	//Block transactions must be valid
	for _, tx := range block.Transactions {
		valid, err := tx.IsValid()

		if err != nil {
			return false, err
		}

		if !valid {
			return false, nil
		}
	}

	//Check merkle root
	merkleRoot := data_models.TransactionsMerkleRoot(block.Transactions)
	if !bytes.Equal(block.Header.MerkleRoot, merkleRoot) {
		return false, nil
	}

	return true, nil
}

func (fullNode *FullNode) validateBlock(block *data_models.Block) (bool, error) {
	//Timestamp must be greater than the median time of the last 11 blocks
	medianTimePast, err := repos.GlobalBlockRepository.GetMedianTimePast(block.Header.PreviousBlockId, 11)
	if err != nil {
		return false, err
	}

	if block.Header.Timestamp < medianTimePast {
		return false, nil
	}

	//Validate transactions on this blocks chain
	valid, err := repos.GlobalTransactionRepository.TransactionsValidInChain(block.Header.PreviousBlockId, block.Transactions)
	if err != nil {
		return false, err
	}

	if !valid {
		return false, nil
	}

	//Validate work
	requiredNBits, err := repos.GlobalBlockRepository.GetNextWorkRequired(block.Header.PreviousBlockId)
	if err != nil {
		return false, err
	}

	if block.Header.NBits != requiredNBits {
		return false, nil
	}

	return true, nil
}

func (fullNode *FullNode) getConnectedOrphans(blockId []byte) []*data_models.Block {
	blocks := make([]*data_models.Block, 0)

	fullNode.orphanBlocksMutex.RLock()
	defer fullNode.orphanBlocksMutex.RUnlock()

	for _, orphanBlock := range fullNode.orphanBlocks.Values() {
		if bytes.Equal(orphanBlock.Header.PreviousBlockId, blockId) {
			blocks = append(blocks, orphanBlock)
		}
	}

	return blocks
}

func (fullNode *FullNode) getOrphanRoot(orphanBlock *data_models.Block) *data_models.Block {
	fullNode.orphanBlocksMutex.RLock()
	defer fullNode.orphanBlocksMutex.RUnlock()

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
	err = repos.GlobalBlockRepository.InsertIfNotExists(block)

	if err != nil {
		log.Printf("|Node| Failed to insert block from miner: %v", err)
		return
	}

	//Send block to peers
	fullNode.network.PeersMutex.RLock()
	for _, peer := range fullNode.network.Peers {
		peer.InventoryToSendMutex.Lock()
		peer.InventoryToSend.AddItem(models.MSG_BLOCK, block.Header.Id)
		peer.InventoryToSendMutex.Unlock()
	}
	fullNode.network.PeersMutex.RUnlock()
}
