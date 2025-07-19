package network

import (
	"log"
	"net"
	"slices"
	"sync"
	"sync/atomic"
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

type Network struct {
	Listener *connectors.Listener
	Dialer   *connectors.Dialer

	Peers      PeersMap
	PeersMutex sync.RWMutex

	NetworkTime atomic.Int64

	commandHandlersMutex sync.Mutex
	commandHandlers      *structures.BytesMap[peer.CommandHandler]

	stopChannel chan bool
	wg          sync.WaitGroup
}

func NewNetwork(ip string, port uint16) *Network {
	network := &Network{}
	network.Listener = connectors.NewListener(ip, port, network.handleConnection)
	network.Dialer = connectors.NewDialer()
	network.Peers = make(PeersMap)
	network.stopChannel = make(chan bool)
	network.commandHandlers = structures.NewBytesMap[peer.CommandHandler]()

	return network
}

func (network *Network) Start() {
	log.Print("|Network| Starting")
	network.Listener.Listen(&network.wg)
	network.dialPeers()
	network.removePeers()
}

func (network *Network) Stop() {
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

func (network *Network) AddCommandHandler(command [12]byte, handler peer.CommandHandler) {
	network.commandHandlersMutex.Lock()
	defer network.commandHandlersMutex.Unlock()

	network.commandHandlers.Put(command[:], handler)
}

func (network *Network) handleConnection(conn net.Conn, initializer bool) {
	network.PeersMutex.Lock()
	defer network.PeersMutex.Unlock()

	//check if already connected to peer
	if _, ok := network.Peers[conn.RemoteAddr().String()]; ok {
		return
	}

	//create peer
	p := peer.NewPeer(conn, initializer)
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
		err := repositories.GlobalAddressRepository.UpdateLastFailed(p.Address, &now)
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
	err = repositories.GlobalAddressRepository.UpdateLastSeen(p.Address, &now)
	if err != nil {
		log.Printf("|Network| Failed to update last seen for address %s: %v", p.Address.String(), err)
		p.Disconnect()
		return
	}
	p.LastSeen = &now

	//add peer to map
	network.Peers[conn.RemoteAddr().String()] = p

	//update network time
	network.setNetworkTime()

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

func (network *Network) setNetworkTime() {
	offsets := make([]int64, 0, len(network.Peers))
	for _, peer := range network.Peers {
		offsets = append(offsets, peer.PeerDetails.TimeOffset)
	}

	if len(offsets) == 0 {
		network.NetworkTime.Store(time.Now().Unix())
		return
	}

	slices.Sort(offsets)
	medianOffset := offsets[len(offsets)/2]
	network.NetworkTime.Store(time.Now().Add(time.Duration(medianOffset) * time.Second).Unix())
}

func (network *Network) dialPeers() {
	ticker := time.NewTicker(2 * time.Minute)
	network.wg.Add(1)

	go func() {
		defer network.wg.Done()
		defer ticker.Stop()

		for {
			select {
			case <-network.stopChannel:
				log.Println("|Network| Stopping dial peers")
				return
			case <-ticker.C:
				network.PeersMutex.RLock()
				if len(network.Peers) >= config.GlobalConfig.NetworkConfig.MaxNumberOfConnections {
					network.PeersMutex.RUnlock()
					continue
				}

				neededAddresses := config.GlobalConfig.NetworkConfig.MaxNumberOfConnections - len(network.Peers)
				excludedIps := make([]net.IP, 0)
				excludedPorts := make([]uint16, 0)

				for _, peer := range network.Peers {
					excludedIps = append(excludedIps, peer.Address.Ip)
					excludedPorts = append(excludedPorts, peer.Address.Port)
				}
				network.PeersMutex.RUnlock()

				addresses, err := repositories.GlobalAddressRepository.GetAddresses(neededAddresses, excludedIps, excludedPorts)
				if err != nil {
					log.Printf("|Network| Failed to get addresses: %v", err)
					continue
				}

				for _, address := range addresses {
					err := network.Dialer.Dial(address.Ip, address.Port)
					if err != nil {
						log.Printf("|Network| Failed to dial address %s: %v", address.String(), err)

						now := time.Now()
						err := repositories.GlobalAddressRepository.UpdateLastFailed(address, &now)
						if err != nil {
							log.Printf("|Network| Failed to update last failed for address %s: %v", address.String(), err)
						}
					}
				}
			}
		}
	}()
}

func (network *Network) removePeers() {
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
					pongTimeout := time.Duration(config.GlobalConfig.NetworkConfig.PongTimeout) * time.Second
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
				network.setNetworkTime()
				network.PeersMutex.Unlock()
			}
		}
	}()
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
	if fromPeer.PingPongDetails.Nonce != n {
		return
	}

	latency := time.Since(fromPeer.PingPongDetails.PingTime)
	fromPeer.PingPongDetails.Latency = latency
	fromPeer.PingPongDetails.PongTime = time.Now()
	fromPeer.PingPongDetails.Nonce = 0

	if fromPeer.LastSeen == nil || time.Since(*fromPeer.LastSeen) > 5*time.Minute {
		now := time.Now()
		err := repositories.GlobalAddressRepository.UpdateLastSeen(fromPeer.Address, &now)
		if err != nil {
			log.Printf("|Network| Failed to update last seen for address %s: %v", fromPeer.Address.String(), err)
			return
		}

		fromPeer.LastSeen = &now
	}
}

func (network *Network) processGetAddr(fromPeer *peer.Peer, message *models.Message) {
	excludedIps := []net.IP{fromPeer.Address.Ip}
	excludedPorts := []uint16{fromPeer.Address.Port}
	addresses, err := repositories.GlobalAddressRepository.GetAddresses(models.MAX_ADDR_SIZE, excludedIps, excludedPorts)

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

func (network *Network) processAddr(fromPeer *peer.Peer, message *models.Message) {
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

	for _, address := range addr.Addresses {
		err := network.addAddress(address)
		if err != nil {
			log.Printf("|Network| Failed to insert address %s from peer %s: %v", address.String(), fromPeer.String(), err)
			return
		}
	}
}

func (network *Network) addAddress(address *models.Address) error {
	if address.NodeType != 1 {
		return nil
	}

	if !address.IsValid() || !address.IsRoutable() {
		return nil
	}

	if config.GlobalConfig.NetworkConfig.Ip == address.Ip.String() && config.GlobalConfig.NetworkConfig.Port == address.Port {
		return nil
	}

	return repositories.GlobalAddressRepository.InsertIfNotExists(address)
}
