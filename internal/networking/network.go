package networking

import (
	"net"
	"sync"
	"time"

	networking_models "github.com/nivschuman/VotingBlockchain/internal/networking/connection"
	models "github.com/nivschuman/VotingBlockchain/internal/networking/models"
)

type Network struct {
	Listener         *Listener
	Peers            map[net.Addr]networking_models.Peer
	BroadcastChannel chan models.Message

	PeersMutex sync.Mutex
}

func (*Network) StartNetwork() {
	//TBD must go over all peers in database and dial them...
	//TBD must start listener
}

func (network *Network) RemoveInactivePeers() {
	ticker := time.NewTicker(30 * time.Second) // Run every 30s
	defer ticker.Stop()

	for {
		<-ticker.C
		now := time.Now()

		network.PeersMutex.Lock()

		for addr, peer := range network.Peers {
			if now.Sub(peer.LastMessageTime) > 2*time.Minute {
				peer.Disconnect()
				delete(network.Peers, addr)
			}
		}

		network.PeersMutex.Unlock()
	}
}

func (network *Network) ListenToBroadcasts() {
	for msg := range network.BroadcastChannel {
		network.BroadcastMessage(msg)
	}
}

func (network *Network) BroadcastMessage(msg models.Message) {
	network.PeersMutex.Lock()

	for _, peer := range network.Peers {
		if peer.CompletedHandshake {
			peer.SendChannel <- msg
		}
	}

	network.PeersMutex.Unlock()
}

func (network *Network) handleConnection(conn net.Conn) {
	//already connected to peer
	if _, ok := network.Peers[conn.RemoteAddr()]; ok {
		return
	}

	//TBD add peer to database if this is a peer we have never seen before...

	network.PeersMutex.Lock()
	network.Peers[conn.RemoteAddr()] = *networking_models.NewPeer(conn, 10, 10, network.BroadcastChannel)
	network.PeersMutex.Unlock()
}
