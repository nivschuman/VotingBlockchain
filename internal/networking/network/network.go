package network

import (
	"log"
	"net"
	"slices"
	"sync"
	"time"

	config "github.com/nivschuman/VotingBlockchain/internal/config"
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
	network.Listener.Listen(&network.wg)
	network.dialPeers()
	network.removePeers()
}

func (network *Network) Stop() {
	close(network.stopChannel)
	network.Listener.StopListening()
	network.wg.Wait()

	network.PeersMutex.Lock()
	defer network.PeersMutex.Unlock()

	for _, peer := range network.Peers {
		log.Printf("Network: removing peer %s", peer.String())
		peer.Disconnect()
	}
	network.Peers = make(PeersMap)
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

func (network *Network) AddCommandHandler(command [12]byte, handler peer.CommandHandler) {
	network.commandHandlersMutex.Lock()
	network.commandHandlers.Put(command[:], handler)
	network.commandHandlersMutex.Unlock()
}

func (network *Network) handleConnection(conn net.Conn, initializer bool) {
	network.PeersMutex.Lock()
	defer network.PeersMutex.Unlock()

	if _, ok := network.Peers[conn.RemoteAddr().String()]; ok {
		return
	}

	p := peer.NewPeer(conn, initializer)
	p.Start()
	err := p.WaitForHandshake(time.Second * 10)

	if err != nil {
		log.Printf("Failed to complete handshake with peer %s: %v", p.String(), p.HandshakeDetails.Error)
		p.Disconnect()
		return
	}

	//TBD add peer to database if this is a peer we have never seen before...
	network.Peers[conn.RemoteAddr().String()] = p

	network.commandHandlersMutex.Lock()
	for _, command := range network.commandHandlers.Keys() {
		var c [12]byte
		copy(c[:], command)

		p.AddCommandHandler(c, network.commandHandlers.GetOrDefault(command, nil))
	}
	network.commandHandlersMutex.Unlock()

	p.AddCommandHandler(models.CommandPing, network.processPing)
	p.AddCommandHandler(models.CommandPong, network.processPong)
	p.StartProcessing()
}

func (network *Network) dialPeers() {
	network.wg.Add(1)

	go func() {
		defer network.wg.Done()
		//TBD must go over all peers in database and dial them...
		log.Println("Network: stopping dial peers")
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
					log.Printf("Network: removing peer %s", peer.String())
					peer.Disconnect()
					delete(network.Peers, peer.Conn.RemoteAddr().String())
				}
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
	if fromPeer.PingPongDetails.Nonce == n {
		latency := time.Since(fromPeer.PingPongDetails.PingTime)
		fromPeer.PingPongDetails.Latency = latency
		fromPeer.PingPongDetails.PongTime = time.Now()
		fromPeer.PingPongDetails.Nonce = 0
	}
}
