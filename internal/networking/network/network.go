package network

import (
	"bytes"
	"log"
	"net"
	"sync"
	"time"

	config "github.com/nivschuman/VotingBlockchain/internal/config"
	repos "github.com/nivschuman/VotingBlockchain/internal/database/repositories"
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

	stopChannel chan bool      // Channel to signal shutdown
	wg          sync.WaitGroup // WaitGroup to track running goroutines
}

func NewNetwork() *Network {
	network := &Network{}

	ip := config.GlobalConfig.NetworkConfig.Ip
	port := config.GlobalConfig.NetworkConfig.Port

	network.Listener = connectors.NewListener(ip, port, network.handleConnection)
	network.Dialer = connectors.NewDialer()
	network.Peers = make(PeersMap)
	network.stopChannel = make(chan bool)

	return network
}

func (network *Network) StartNetwork() {
	network.Listener.Listen(&network.wg)

	network.wg.Add(2)
	go network.DialPeers()
	go network.RemovePeers()
}

func (network *Network) StopNetwork() {
	close(network.stopChannel)
	network.Listener.StopListening()

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

func (network *Network) BroadcastMessage(msg *models.Message) {
	network.PeersMutex.RLock()

	for _, peer := range network.Peers {
		if peer.CompletedHandshake() && !peer.Remove && !peer.Disconnected {
			select {
			case <-peer.StopChannel:
				continue
			case peer.SendChannel <- *msg:
			}
		}
	}

	network.PeersMutex.RUnlock()
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

	p.StartProcessing(network.processMessage)
}

func (network *Network) processMessage(fromPeer *peer.Peer, message *models.Message) {
	//ping
	if bytes.Equal(message.MessageHeader.Command[:], models.CommandPing[:]) {
		select {
		case <-fromPeer.StopChannel:
			return
		case fromPeer.SendChannel <- *models.NewMessage(models.CommandPong, message.Payload):
		}
	}

	//pong
	if bytes.Equal(message.MessageHeader.Command[:], models.CommandPong[:]) {
		n := nonce.NonceFromBytes(message.Payload)
		if fromPeer.PingPongDetails.Nonce == n {
			latency := time.Since(fromPeer.PingPongDetails.PingTime)
			fromPeer.PingPongDetails.Latency = latency
			fromPeer.PingPongDetails.PongTime = time.Now()
			fromPeer.PingPongDetails.Nonce = 0
		}
	}

	//inv
	if bytes.Equal(message.MessageHeader.Command[:], models.CommandInv[:]) {
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

		for _, id := range missingTransactions.ToBytesSlice() {
			getData.AddItem(models.MSG_TX, id)
		}

		//TBD handle blocks

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

	//tx
	if bytes.Equal(message.MessageHeader.Command[:], models.CommandTx[:]) {
		transaction, err := data_models.TransactionFromBytes(message.Payload)

		if err != nil {
			log.Printf("Failed to parse transaction from %s: %v", fromPeer.String(), err)
			return
		}

		valid, err := repos.GlobalTransactionRepository.TransactionIsValid(transaction)

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

	//mempool
	if bytes.Equal(message.MessageHeader.Command[:], models.CommandMemPool[:]) {
		transactions, err := repos.GlobalTransactionRepository.GetMempool(10)

		if err != nil {
			log.Printf("Failed to get mempool for %s: %v", fromPeer.String(), err)
			return
		}

		inv := models.NewInv()

		for _, tx := range transactions {
			inv.AddItem(models.MSG_TX, tx.Id)
		}

		invBytes, err := inv.AsBytes()

		if err != nil {
			log.Printf("Failed to get mempool inv bytes for %s: %v", fromPeer.String(), err)
			return
		}

		mempoolMessage := models.NewMessage(models.CommandInv, invBytes)
		select {
		case <-fromPeer.StopChannel:
			return
		case fromPeer.SendChannel <- *mempoolMessage:
		}
	}
}
