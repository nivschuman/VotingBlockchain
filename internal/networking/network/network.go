package network

import (
	"bytes"
	"log"
	"net"
	"sync"
	"time"

	config "github.com/nivschuman/VotingBlockchain/internal/config"
	connectors "github.com/nivschuman/VotingBlockchain/internal/networking/connectors"
	models "github.com/nivschuman/VotingBlockchain/internal/networking/models"
	peer "github.com/nivschuman/VotingBlockchain/internal/networking/peer"
	nonce "github.com/nivschuman/VotingBlockchain/internal/networking/utils/nonce"
)

const PING_INTERVAL = 2 * time.Minute
const PONG_TIMEOUT = 20 * time.Minute

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

	network.wg.Add(3)
	go network.DialPeers()
	go network.SendPings()
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
				if sinceLastPong > PONG_TIMEOUT {
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
	log.Printf("Network: removing peer %s", peer.Conn.RemoteAddr().String())
	peer.Disconnect()
	delete(network.Peers, peer.Conn.RemoteAddr().String())
}

func (network *Network) SendPings() {
	defer network.wg.Done()
	ticker := time.NewTicker(PING_INTERVAL)
	defer ticker.Stop()

	for {
		select {
		case <-network.stopChannel:
			log.Println("Network: stopping send pings")
			return
		case <-ticker.C:
			network.PeersMutex.RLock()
			for _, peer := range network.Peers {
				if peer.PingPongDetails.Nonce == 0 {
					n, err := nonce.Generator.GenerateNonce()
					if err != nil {
						continue
					}
					peer.PingPongDetails.PingTime = time.Now()
					peer.PingPongDetails.Nonce = n

					select {
					case <-peer.StopChannel:
						continue
					case peer.SendChannel <- *models.NewMessage(models.CommandPing, nonce.NonceToBytes(n)):
					}
				}
			}
			network.PeersMutex.RUnlock()
		}
	}
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
		log.Printf("Failed to complete handshake with peer %s: %v", p.Conn.RemoteAddr().String(), p.HandshakeDetails.Error)
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
}
