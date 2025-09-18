package network

import (
	"log"
	"net"
	"slices"
	"sync"
	"time"

	config "github.com/nivschuman/VotingBlockchain/internal/config"
	"github.com/nivschuman/VotingBlockchain/internal/database/repositories"
	connectors "github.com/nivschuman/VotingBlockchain/internal/networking/connectors"
	models "github.com/nivschuman/VotingBlockchain/internal/networking/models"
	peer "github.com/nivschuman/VotingBlockchain/internal/networking/peer"
	nonce "github.com/nivschuman/VotingBlockchain/internal/networking/utils/nonce"
	structures "github.com/nivschuman/VotingBlockchain/internal/structures"
)

type PeersMap map[string]*peer.Peer

type Network interface {
	Start()
	Stop()
	AddCommandHandler(command [12]byte, handler peer.CommandHandler)
	GetNetworkTime() int64
	BroadcastItemToPeers(msgType uint32, id []byte, exceptPeer *peer.Peer)
	DialAddress(address *models.Address) error
	GetPeers() []*peer.Peer
	RemovePeer(p *peer.Peer)
	GetAddressRepository() repositories.AddressRepository
}

type NetworkImpl struct {
	Listener *connectors.Listener
	Dialer   *connectors.Dialer

	Peers      PeersMap
	PeersMutex sync.RWMutex

	myVersion     models.VersionProvider
	networkConfig *config.NetworkConfig

	addressRepository repositories.AddressRepository

	networkTimeOffset      int64
	networkTimeOffsetMutex sync.Mutex

	commandHandlersMutex sync.Mutex
	commandHandlers      *structures.BytesMap[peer.CommandHandler]

	stopChannel chan bool
	wg          sync.WaitGroup
}

func NewNetworkImpl(addressRepository repositories.AddressRepository, networkConfig *config.NetworkConfig, myVersion models.VersionProvider) *NetworkImpl {
	network := &NetworkImpl{}
	network.Listener = connectors.NewListener(networkConfig.Ip, networkConfig.Port, network.handleConnection)
	network.Dialer = connectors.NewDialer(network.handleConnection)
	network.Peers = make(PeersMap)
	network.stopChannel = make(chan bool)
	network.commandHandlers = structures.NewBytesMap[peer.CommandHandler]()
	network.addressRepository = addressRepository
	network.networkConfig = networkConfig
	network.myVersion = myVersion

	return network
}

func (network *NetworkImpl) Start() {
	log.Print("|Network| Starting")
	network.Listener.Listen(&network.wg)
	network.removePeers()
	if network.networkConfig.Dial {
		network.dialPeers()
	}
}

func (network *NetworkImpl) Stop() {
	log.Print("|Network| Stopping")
	close(network.stopChannel)
	network.Listener.StopListening()
	network.wg.Wait()

	network.PeersMutex.Lock()
	defer network.PeersMutex.Unlock()

	for _, peer := range network.Peers {
		log.Printf("|Network| Removing peer %s", peer.String())
		peer.Disconnect()
	}
	network.Peers = make(PeersMap)
}

func (network *NetworkImpl) AddCommandHandler(command [12]byte, handler peer.CommandHandler) {
	network.commandHandlersMutex.Lock()
	defer network.commandHandlersMutex.Unlock()

	network.commandHandlers.Put(command[:], handler)
}

func (network *NetworkImpl) GetNetworkTime() int64 {
	network.networkTimeOffsetMutex.Lock()
	defer network.networkTimeOffsetMutex.Unlock()

	return time.Now().Add(time.Duration(network.networkTimeOffset) * time.Second).Unix()
}

func (network *NetworkImpl) BroadcastItemToPeers(msgType uint32, id []byte, exceptPeer *peer.Peer) {
	network.PeersMutex.RLock()
	for _, peer := range network.Peers {
		if peer != exceptPeer {
			peer.InventoryToSendMutex.Lock()
			peer.InventoryToSend.AddItem(models.MSG_TX, id)
			peer.InventoryToSendMutex.Unlock()
		}
	}
	network.PeersMutex.RUnlock()
}

func (network *NetworkImpl) DialAddress(address *models.Address) error {
	network.PeersMutex.RLock()
	_, alreadyConnected := network.Peers[address.String()]
	network.PeersMutex.RUnlock()

	if alreadyConnected {
		return nil
	}

	err := network.Dialer.Dial(address.Ip, address.Port)
	if err != nil {
		log.Printf("|Network| Failed to manually dial %s: %v", address.String(), err)

		now := time.Now()
		if err2 := network.addressRepository.UpdateLastFailed(address, &now); err2 != nil {
			log.Printf("|Network| Failed to update last failed for address %s: %v", address.String(), err2)
		}

		return err
	}

	return nil
}

func (network *NetworkImpl) GetPeers() []*peer.Peer {
	network.PeersMutex.RLock()
	defer network.PeersMutex.RUnlock()

	peers := make([]*peer.Peer, 0, len(network.Peers))
	for _, p := range network.Peers {
		peers = append(peers, p)
	}
	return peers
}

func (network *NetworkImpl) RemovePeer(p *peer.Peer) {
	if p == nil {
		return
	}

	network.PeersMutex.Lock()
	defer network.PeersMutex.Unlock()

	key := p.Conn.RemoteAddr().String()
	if _, exists := network.Peers[key]; !exists {
		return
	}

	log.Printf("|Network| Removing peer %s", p.String())
	p.Disconnect()
	delete(network.Peers, key)

	network.setNetworkTimeOffset()
}

func (network *NetworkImpl) GetAddressRepository() repositories.AddressRepository {
	return network.addressRepository
}

func (network *NetworkImpl) createPeerConfig() peer.PeerConfig {
	sendDataInterval := time.Duration(network.networkConfig.SendDataInterval) * time.Second
	pingInterval := time.Duration(network.networkConfig.PingInterval) * time.Second
	getAddrInterval := time.Duration(network.networkConfig.GetAddrInterval) * time.Second

	return peer.PeerConfig{
		SendDataInterval: sendDataInterval,
		PingInterval:     pingInterval,
		GetAddrInterval:  getAddrInterval,
	}
}

func (network *NetworkImpl) handleConnection(conn net.Conn, initializer bool) {
	network.PeersMutex.Lock()
	defer network.PeersMutex.Unlock()

	//check if already connected to peer
	if _, ok := network.Peers[conn.RemoteAddr().String()]; ok {
		return
	}

	//create peer
	p := peer.NewPeer(conn, initializer, network.createPeerConfig(), network.myVersion)
	err := p.SetPeerAddress()
	if err != nil {
		log.Printf("|Network| Failed to set peer %s address: %v", p.String(), err)
		p.Disconnect()
		return
	}

	//start peer
	p.Start()

	//wait for handshake
	err = p.WaitForHandshake(time.Second * 10)
	if err != nil {
		log.Printf("|Network| Failed to complete handshake with peer %s: %v", p.String(), err)
		p.Disconnect()

		now := time.Now()
		err := network.addressRepository.UpdateLastFailed(p.Address, &now)
		if err != nil {
			log.Printf("|Network| Failed to update last failed for address %s: %v", p.Address.String(), err)
		}

		return
	}

	//update peer address
	err = network.addAddress(p.Address)
	if err != nil {
		log.Printf("|Network| Failed to insert address %s: %v", p.Address.String(), err)
		p.Disconnect()
		return
	}

	now := time.Now()
	err = network.addressRepository.UpdateLastSeen(p.Address, &now)
	if err != nil {
		log.Printf("|Network| Failed to update last seen for address %s: %v", p.Address.String(), err)
		p.Disconnect()
		return
	}
	p.LastSeen = &now

	//add peer to map
	network.Peers[conn.RemoteAddr().String()] = p

	//update network time
	network.setNetworkTimeOffset()

	//attach handlers to peer
	network.commandHandlersMutex.Lock()
	for _, command := range network.commandHandlers.Keys() {
		var c [12]byte
		copy(c[:], command)

		p.AddCommandHandler(c, network.commandHandlers.GetOrDefault(command, nil))
	}
	network.commandHandlersMutex.Unlock()

	p.AddCommandHandler(models.CommandPing, network.processPing)
	p.AddCommandHandler(models.CommandPong, network.processPong)
	p.AddCommandHandler(models.CommandAddr, network.processAddr)
	p.AddCommandHandler(models.CommandGetAddr, network.processGetAddr)

	//start peer processing
	p.StartProcessing()
}

func (network *NetworkImpl) setNetworkTimeOffset() {
	network.networkTimeOffsetMutex.Lock()
	defer network.networkTimeOffsetMutex.Unlock()

	offsets := make([]int64, 0, len(network.Peers))
	for _, peer := range network.Peers {
		offsets = append(offsets, peer.PeerDetails.TimeOffset)
	}

	if len(offsets) == 0 {
		network.networkTimeOffset = 0
		return
	}

	slices.Sort(offsets)
	network.networkTimeOffset = offsets[len(offsets)/2]
}

func (network *NetworkImpl) dialPeers() {
	ticker := time.NewTicker(2 * time.Minute)
	network.wg.Add(1)

	dial := func() {
		network.PeersMutex.RLock()
		if len(network.Peers) >= network.networkConfig.MaxNumberOfConnections {
			network.PeersMutex.RUnlock()
			return
		}

		neededAddresses := network.networkConfig.MaxNumberOfConnections - len(network.Peers)

		excludedAddresses := make([]*models.Address, len(network.Peers))
		idx := 0
		for _, peer := range network.Peers {
			excludedAddresses[idx] = peer.Address
			idx++
		}
		network.PeersMutex.RUnlock()

		addresses, err := network.addressRepository.GetAddresses(neededAddresses, excludedAddresses)
		if err != nil {
			log.Printf("|Network| Failed to get addresses: %v", err)
			return
		}

		for _, address := range addresses {
			err := network.Dialer.Dial(address.Ip, address.Port)
			if err != nil {
				log.Printf("|Network| Failed to dial address %s: %v", address.String(), err)

				now := time.Now()
				err := network.addressRepository.UpdateLastFailed(address, &now)
				if err != nil {
					log.Printf("|Network| Failed to update last failed for address %s: %v", address.String(), err)
				}
			}
		}
	}

	go func() {
		defer network.wg.Done()
		defer ticker.Stop()

		dial()
		for {
			select {
			case <-network.stopChannel:
				log.Println("|Network| Stopping dial peers")
				return
			case <-ticker.C:
				dial()
			}
		}
	}()
}

func (network *NetworkImpl) removePeers() {
	ticker := time.NewTicker(30 * time.Second)
	network.wg.Add(1)

	go func() {
		defer network.wg.Done()
		defer ticker.Stop()

		for {
			select {
			case <-network.stopChannel:
				log.Println("|Network| Stopping remove peers")
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
					pongTimeout := time.Duration(network.networkConfig.PongTimeout) * time.Second
					if sinceLastPong > pongTimeout {
						toRemove = append(toRemove, peer)
						continue
					}
				}
				network.PeersMutex.RUnlock()

				network.PeersMutex.Lock()
				for _, peer := range toRemove {
					log.Printf("|Network| Removing peer %s", peer.String())
					peer.Disconnect()
					delete(network.Peers, peer.Conn.RemoteAddr().String())
				}
				network.setNetworkTimeOffset()
				network.PeersMutex.Unlock()
			}
		}
	}()
}

func (network *NetworkImpl) processPing(fromPeer *peer.Peer, message *models.Message) {
	select {
	case <-fromPeer.StopChannel:
		return
	case fromPeer.SendChannel <- *models.NewMessage(models.CommandPong, message.Payload):
	}
}

func (network *NetworkImpl) processPong(fromPeer *peer.Peer, message *models.Message) {
	n := nonce.NonceFromBytes(message.Payload)
	if fromPeer.PingPongDetails.Nonce != n {
		return
	}

	latency := time.Since(fromPeer.PingPongDetails.PingTime)
	fromPeer.PingPongDetails.Latency = latency
	fromPeer.PingPongDetails.PongTime = time.Now()
	fromPeer.PingPongDetails.Nonce = 0

	if fromPeer.LastSeen == nil || time.Since(*fromPeer.LastSeen) > 5*time.Minute {
		now := time.Now()
		err := network.addressRepository.UpdateLastSeen(fromPeer.Address, &now)
		if err != nil {
			log.Printf("|Network| Failed to update last seen for address %s: %v", fromPeer.Address.String(), err)
			return
		}

		fromPeer.LastSeen = &now
	}
}

func (network *NetworkImpl) processGetAddr(fromPeer *peer.Peer, message *models.Message) {
	excludedAddresses := []*models.Address{fromPeer.Address}
	addresses, err := network.addressRepository.GetAddresses(models.MAX_ADDR_SIZE, excludedAddresses)

	if err != nil {
		log.Printf("|Network| Failed to get addresses for peer %s: %v", fromPeer.String(), err)
		return
	}

	addr := models.NewAddr()
	for _, address := range addresses {
		addr.AddAddress(address)
	}

	addrMessage, err := models.NewAddrMessage(addr)
	if err != nil {
		log.Printf("|Network| Failed to create addr message for peer %s: %v", fromPeer.String(), err)
		return
	}

	select {
	case <-fromPeer.StopChannel:
		return
	case fromPeer.SendChannel <- *addrMessage:
	}
}

func (network *NetworkImpl) processAddr(fromPeer *peer.Peer, message *models.Message) {
	fromPeer.SentGetAddrMutex.Lock()
	defer fromPeer.SentGetAddrMutex.Unlock()

	addr, err := models.AddrFromBytes(message.Payload)
	if err != nil {
		log.Printf("|Network| Failed to parse addr message of peer %s: %v", fromPeer.String(), err)
		return
	}

	if addr.Count > models.MAX_ADDR_SIZE {
		log.Printf("|Network| Received more than %d addresses from peer %s", models.MAX_ADDR_SIZE, fromPeer.String())
		return
	}

	if !fromPeer.SentGetAddr {
		log.Printf("|Network| Peer %s sent addr without get addr request", fromPeer.String())
		return
	}

	fromPeer.SentGetAddr = false

	for _, address := range addr.Addresses {
		err := network.addAddress(address)
		if err != nil {
			log.Printf("|Network| Failed to insert address %s from peer %s: %v", address.String(), fromPeer.String(), err)
			return
		}
	}
}

func (network *NetworkImpl) addAddress(address *models.Address) error {
	if address.NodeType != 1 {
		return nil
	}

	if !address.IsValid() || !address.IsRoutable() {
		return nil
	}

	if network.networkConfig.Ip.Equal(address.Ip) && network.networkConfig.Port == address.Port {
		return nil
	}

	return network.addressRepository.InsertIfNotExists(address)
}
