package network

import (
	"bytes"
	"log"
	"net"
	"slices"
	"sync"
	"time"

	config "github.com/nivschuman/VotingBlockchain/internal/config"
	repos "github.com/nivschuman/VotingBlockchain/internal/database/repositories"
	difficulty "github.com/nivschuman/VotingBlockchain/internal/difficulty"
	mining "github.com/nivschuman/VotingBlockchain/internal/mining"
	data_models "github.com/nivschuman/VotingBlockchain/internal/models"
	connectors "github.com/nivschuman/VotingBlockchain/internal/networking/connectors"
	models "github.com/nivschuman/VotingBlockchain/internal/networking/models"
	peer "github.com/nivschuman/VotingBlockchain/internal/networking/peer"
	nonce "github.com/nivschuman/VotingBlockchain/internal/networking/utils/nonce"
	structures "github.com/nivschuman/VotingBlockchain/internal/structures"
)

type PeersMap map[string]*peer.Peer

type Network struct {
	Listener   *connectors.Listener
	Dialer     *connectors.Dialer
	Peers      PeersMap
	PeersMutex sync.RWMutex
	Miner      *mining.Miner

	orphanBlocks      *structures.BytesMap[*data_models.Block]
	orphanBlocksMutex sync.RWMutex

	stopChannel chan bool
	wg          sync.WaitGroup
}

func NewNetwork() *Network {
	network := &Network{}

	ip := config.GlobalConfig.NetworkConfig.Ip
	port := config.GlobalConfig.NetworkConfig.Port

	network.Listener = connectors.NewListener(ip, port, network.handleConnection)
	network.Dialer = connectors.NewDialer()
	network.Peers = make(PeersMap)

	network.Miner = mining.NewMiner()
	network.Miner.AddHandler(network.processMinedBlock)

	network.stopChannel = make(chan bool)
	network.orphanBlocks = structures.NewBytesMap[*data_models.Block]()

	return network
}

func (network *Network) StartNetwork() {
	network.Listener.Listen(&network.wg)
	network.wg.Add(2)
	go network.DialPeers()
	go network.RemovePeers()
}

func (network *Network) StartMiner() {
	network.Miner.StartMiner(&network.wg)
}

func (network *Network) StopNetwork() {
	close(network.stopChannel)
	network.Listener.StopListening()
	network.Miner.StopMiner()

	network.PeersMutex.Lock()
	for _, peer := range network.Peers {
		network.RemovePeer(peer)
	}
	network.Peers = make(PeersMap)
	network.PeersMutex.Unlock()

	network.wg.Wait()
}

func (network *Network) DialPeers() {
	log.Println("Network: stopping dial peers")
	defer network.wg.Done()
	//TBD must go over all peers in database and dial them...
}

func (network *Network) RemovePeers() {
	defer network.wg.Done()
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-network.stopChannel:
			log.Println("Network: stopping remove peers")
			return
		case <-ticker.C:
			network.PeersMutex.RLock()
			toRemove := make([]*peer.Peer, 0)
			for _, peer := range network.Peers {
				if peer.Remove || peer.Disconnected {
					toRemove = append(toRemove, peer)
					continue
				}
				sinceLastPong := time.Since(peer.PingPongDetails.PongTime)
				pongTimeout := time.Duration(config.GlobalConfig.NetworkConfig.PongTimeout) * time.Second
				if sinceLastPong > pongTimeout {
					toRemove = append(toRemove, peer)
					continue
				}
			}
			network.PeersMutex.RUnlock()

			network.PeersMutex.Lock()
			for _, peer := range toRemove {
				network.RemovePeer(peer)
			}
			network.PeersMutex.Unlock()
		}
	}
}

func (network *Network) RemovePeer(peer *peer.Peer) {
	log.Printf("Network: removing peer %s", peer.String())
	peer.Disconnect()
	delete(network.Peers, peer.Conn.RemoteAddr().String())
}

func (network *Network) NetworkTime() int64 {
	network.PeersMutex.RLock()
	offsets := make([]int64, 0, len(network.Peers))

	for _, peer := range network.Peers {
		if peer.CompletedHandshake() {
			offsets = append(offsets, peer.PeerDetails.TimeOffset)
		}
	}
	network.PeersMutex.RUnlock()

	if len(offsets) == 0 {
		return time.Now().Unix()
	}

	slices.Sort(offsets)
	medianOffset := offsets[len(offsets)/2]

	return time.Now().Add(time.Duration(medianOffset) * time.Second).Unix()
}

func (network *Network) handleConnection(conn net.Conn, initializer bool) {
	//already connected to peer
	network.PeersMutex.RLock()
	if _, ok := network.Peers[conn.RemoteAddr().String()]; ok {
		network.PeersMutex.RUnlock()
		return
	}
	network.PeersMutex.RUnlock()

	p := peer.NewPeer(conn, initializer)
	p.StartPeer()
	err := p.WaitForHandshake(time.Second * 10)

	if err != nil {
		log.Printf("Failed to complete handshake with peer %s: %v", p.String(), p.HandshakeDetails.Error)
		p.Disconnect()
		return
	}

	//TBD add peer to database if this is a peer we have never seen before...

	network.PeersMutex.Lock()
	network.Peers[conn.RemoteAddr().String()] = p
	network.PeersMutex.Unlock()

	p.AddCommandHandler(models.CommandPing, network.processPing)
	p.AddCommandHandler(models.CommandPong, network.processPong)
	p.AddCommandHandler(models.CommandBlock, network.processBlock)
	p.AddCommandHandler(models.CommandGetData, network.processGetData)
	p.AddCommandHandler(models.CommandInv, network.processInv)
	p.AddCommandHandler(models.CommandMemPool, network.processMemPool)
	p.AddCommandHandler(models.CommandTx, network.processTx)
	p.AddCommandHandler(models.CommandGetBlocks, network.processGetBlocks)

	p.StartProcessing()
}

func (network *Network) processPing(fromPeer *peer.Peer, message *models.Message) {
	select {
	case <-fromPeer.StopChannel:
		return
	case fromPeer.SendChannel <- *models.NewMessage(models.CommandPong, message.Payload):
	}
}

func (network *Network) processPong(fromPeer *peer.Peer, message *models.Message) {
	n := nonce.NonceFromBytes(message.Payload)
	if fromPeer.PingPongDetails.Nonce == n {
		latency := time.Since(fromPeer.PingPongDetails.PingTime)
		fromPeer.PingPongDetails.Latency = latency
		fromPeer.PingPongDetails.PongTime = time.Now()
		fromPeer.PingPongDetails.Nonce = 0
	}
}

func (network *Network) processInv(fromPeer *peer.Peer, message *models.Message) {
	inv, err := models.InvFromBytes(message.Payload)

	if err != nil {
		log.Printf("Failed to parse inv from %s: %v", fromPeer.String(), err)
		return
	}

	getData := models.NewGetData()

	blockHashes := structures.NewBytesSet()
	txHashes := structures.NewBytesSet()

	for _, invItem := range inv.Items {
		if invItem.Type == models.MSG_BLOCK {
			blockHashes.Add(invItem.Hash)
		} else if invItem.Type == models.MSG_TX {
			txHashes.Add(invItem.Hash)
		}
	}

	missingTransactions, err := repos.GlobalTransactionRepository.GetMissingTransactionIds(txHashes)

	if err != nil {
		log.Printf("Failed to get missing transactions : %v", err)
		return
	}

	missingBlocks, err := repos.GlobalBlockRepository.GetMissingBlockIds(blockHashes)

	if err != nil {
		log.Printf("Failed to get missing blocks : %v", err)
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
		log.Printf("Failed to make get data message for %s : %v", fromPeer.String(), err)
		return
	}

	select {
	case <-fromPeer.StopChannel:
		return
	case fromPeer.SendChannel <- *getDataMessage:
	}
}

func (network *Network) processTx(fromPeer *peer.Peer, message *models.Message) {
	transaction, err := data_models.TransactionFromBytes(message.Payload)

	if err != nil {
		log.Printf("Failed to parse transaction from %s: %v", fromPeer.String(), err)
		return
	}

	valid, err := transaction.IsValid()

	if err != nil {
		log.Printf("Failed to validate transaction from %s: %v", fromPeer.String(), err)
		return
	}

	if !valid {
		log.Printf("Received invalid transaction from %s", fromPeer.String())
		return
	}

	valid, err = repos.GlobalTransactionRepository.TransactionValidInActiveChain(transaction)

	if err != nil {
		log.Printf("Failed validating transaction from %s: %v", fromPeer.String(), err)
		return
	}

	if !valid {
		log.Printf("Received invalid transaction from %s", fromPeer.String())
		return
	}

	err = repos.GlobalTransactionRepository.InsertIfNotExists(transaction)

	if err != nil {
		log.Printf("Failed to insert transaction from %s: %v", fromPeer.String(), err)
	}

	network.PeersMutex.RLock()
	for _, peer := range network.Peers {
		if peer != fromPeer {
			peer.InventoryToSendMutex.Lock()
			peer.InventoryToSend.AddItem(models.MSG_TX, transaction.Id)
			peer.InventoryToSendMutex.Unlock()
		}
	}
	network.PeersMutex.RUnlock()
}

func (network *Network) processMemPool(fromPeer *peer.Peer, _ *models.Message) {
	transactions, err := repos.GlobalTransactionRepository.GetMempool(10)

	if err != nil {
		log.Printf("Failed to get mempool for %s: %v", fromPeer.String(), err)
		return
	}

	inv := models.NewInv()

	for _, tx := range transactions {
		inv.AddItem(models.MSG_TX, tx.Id)
	}

	mempoolMessage, err := models.NewInvMessage(inv)

	if err != nil {
		log.Printf("Failed to create mempool inv message for %s: %v", fromPeer.String(), err)
		return
	}

	select {
	case <-fromPeer.StopChannel:
		return
	case fromPeer.SendChannel <- *mempoolMessage:
	}
}

func (network *Network) processGetData(fromPeer *peer.Peer, message *models.Message) {
	getData, err := models.GetDataFromBytes(message.Payload)

	if err != nil {
		log.Printf("Failed to parse getdata from %s: %v", fromPeer.String(), err)
		return
	}

	blockHashes := structures.NewBytesSet()
	txHashes := structures.NewBytesSet()

	for _, item := range getData.Items() {
		if item.Type == models.MSG_BLOCK {
			blockHashes.Add(item.Hash)
		} else if item.Type == models.MSG_TX {
			txHashes.Add(item.Hash)
		}
	}

	transactions, err := repos.GlobalTransactionRepository.GetTransactions(txHashes)

	if err != nil {
		log.Printf("Failed to get transactions : %v", err)
		return
	}

	blocks, err := repos.GlobalBlockRepository.GetBlocks(blockHashes)

	if err != nil {
		log.Printf("Failed to get blocks : %v", err)
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

func (network *Network) processGetBlocks(fromPeer *peer.Peer, message *models.Message) {
	getBlocks, err := models.GetBlocksFromBytes(message.Payload)

	if err != nil {
		log.Printf("Failed to parse getblocks from %s: %v", fromPeer.String(), err)
		return
	}

	blocksIds, err := repos.GlobalBlockRepository.GetNextBlocksIds(getBlocks.BlockLocator, getBlocks.StopHash, 500)

	if err != nil {
		log.Printf("Failed to parse get next blocks for %s: %v", fromPeer.String(), err)
		return
	}

	inv := models.NewInv()

	for _, blockId := range blocksIds.ToBytesSlice() {
		inv.AddItem(models.MSG_BLOCK, blockId)
	}

	invMessage, err := models.NewInvMessage(inv)

	if err != nil {
		log.Printf("Failed to create get blocks inv message for %s: %v", fromPeer.String(), err)
		return
	}

	select {
	case <-fromPeer.StopChannel:
		return
	case fromPeer.SendChannel <- *invMessage:
	}
}

func (network *Network) processBlock(fromPeer *peer.Peer, message *models.Message) {
	//Parse block
	block, err := data_models.BlockFromBytes(message.Payload)

	if err != nil {
		log.Printf("Failed to parse block from %s: %v", fromPeer.String(), err)
		return
	}

	//Check if already have block
	if network.orphanBlocks.ContainsKey(block.Header.Id) {
		return
	}

	exists, err := repos.GlobalBlockRepository.HaveBlock(block.Header.Id)

	if err != nil {
		log.Printf("Failed to check if block exists from %s is orphan: %v", fromPeer.String(), err)
		return
	}

	if exists {
		return
	}

	//Check if block is orphan
	isOrphan, err := repos.GlobalBlockRepository.BlockIsOrphan(block)

	if err != nil {
		log.Printf("Failed to check if block from %s is orphan: %v", fromPeer.String(), err)
	}

	//Check block
	isValid, err := network.checkBlock(block)

	if err != nil {
		log.Printf("Failed to check block from %s: %v", fromPeer.String(), err)
		return
	}

	if !isValid {
		log.Printf("Received invalid block from %s", fromPeer.String())
		return
	}

	//No further processing for orphan
	if isOrphan {
		network.orphanBlocksMutex.Lock()
		network.orphanBlocks.Put(block.Header.Id, block)
		network.orphanBlocksMutex.Unlock()

		//Ask for block
		blockLocator, err := repos.GlobalBlockRepository.GetActiveChainBlockLocator()

		if err != nil {
			log.Printf("Failed to check active chain block locator for %s: %v", fromPeer.String(), err)
			return
		}

		orphanRoot := network.getOrphanRoot(block)
		getBlocks := models.NewGetBlocks(blockLocator, orphanRoot.Header.Id)

		msg, err := models.NewGetBlocksMessage(getBlocks)
		if err != nil {
			log.Printf("Failed to make get blocks message for %s: %v", fromPeer.String(), err)
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
	isValid, err = network.validateBlock(block)

	if err != nil {
		log.Printf("Failed to validate block from %s: %v", fromPeer.String(), err)
		return
	}

	if !isValid {
		log.Printf("Received invalid block from %s", fromPeer.String())
		return
	}

	//Insert block
	err = repos.GlobalBlockRepository.InsertIfNotExists(block)

	if err != nil {
		log.Printf("Failed to insert block from %s: %v", fromPeer.String(), err)
		return
	}

	//Send block to peers
	network.PeersMutex.RLock()
	for _, peer := range network.Peers {
		if peer != fromPeer {
			peer.InventoryToSendMutex.Lock()
			peer.InventoryToSend.AddItem(models.MSG_BLOCK, block.Header.Id)
			peer.InventoryToSendMutex.Unlock()
		}
	}
	network.PeersMutex.RUnlock()

	//Process dependent orphans recursively
	queue := []*data_models.Block{block}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		children := network.getConnectedOrphans(current.Header.Id)

		for _, child := range children {
			isValid, err := network.validateBlock(child)

			if err != nil {
				log.Printf("Failed to validate orphan child %x: %v", child.Header.Id, err)
				continue
			}

			if !isValid {
				log.Printf("Invalid orphan child block %x", child.Header.Id)
				continue
			}

			err = repos.GlobalBlockRepository.InsertIfNotExists(child)

			if err != nil {
				log.Printf("Failed to insert orphan child block %x: %v", child.Header.Id, err)
				continue
			}

			network.PeersMutex.RLock()
			for _, peer := range network.Peers {
				if peer != fromPeer {
					peer.InventoryToSendMutex.Lock()
					peer.InventoryToSend.AddItem(models.MSG_BLOCK, child.Header.Id)
					peer.InventoryToSendMutex.Unlock()
				}
			}
			network.PeersMutex.RUnlock()

			network.orphanBlocksMutex.Lock()
			network.orphanBlocks.Remove(child.Header.Id)
			network.orphanBlocksMutex.Unlock()

			queue = append(queue, child)
		}
	}
}

func (network *Network) checkBlock(block *data_models.Block) (bool, error) {
	//Timestamp must be less than the network adjusted time +2 hours.
	if block.Header.Timestamp > network.NetworkTime()+2*60*60 {
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

	//TBD check merkle root

	return true, nil
}

func (network *Network) validateBlock(block *data_models.Block) (bool, error) {
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

func (network *Network) getConnectedOrphans(blockId []byte) []*data_models.Block {
	blocks := make([]*data_models.Block, 0)

	network.orphanBlocksMutex.RLock()
	orphanBlocks := network.orphanBlocks.Values()
	network.orphanBlocksMutex.RUnlock()

	for _, orphanBlock := range orphanBlocks {
		if bytes.Equal(orphanBlock.Header.PreviousBlockId, blockId) {
			blocks = append(blocks, orphanBlock)
		}
	}

	return blocks
}

func (network *Network) getOrphanRoot(orphanBlock *data_models.Block) *data_models.Block {
	network.orphanBlocksMutex.RLock()
	for {
		prevId := orphanBlock.Header.PreviousBlockId

		orphanParent, exists := network.orphanBlocks.Get(prevId)
		if !exists {
			break
		}

		orphanBlock = orphanParent
	}
	network.orphanBlocksMutex.RUnlock()

	return orphanBlock
}

func (network *Network) processMinedBlock(block *data_models.Block) {
	//Check block
	isValid, err := network.checkBlock(block)

	if err != nil {
		log.Printf("Failed to check block mined block: %v", err)
		return
	}

	if !isValid {
		log.Print("Received invalid block from miner")
		return
	}

	//Validate block
	isValid, err = network.validateBlock(block)

	if err != nil {
		log.Printf("Failed to validate block miner: %v", err)
		return
	}

	if !isValid {
		log.Print("Received invalid block from miner")
		return
	}

	//Insert block
	err = repos.GlobalBlockRepository.InsertIfNotExists(block)

	if err != nil {
		log.Printf("Failed to insert block from miner: %v", err)
		return
	}

	//Send block to peers
	network.PeersMutex.RLock()
	for _, peer := range network.Peers {
		peer.InventoryToSendMutex.Lock()
		peer.InventoryToSend.AddItem(models.MSG_BLOCK, block.Header.Id)
		peer.InventoryToSendMutex.Unlock()
	}
	network.PeersMutex.RUnlock()
}
