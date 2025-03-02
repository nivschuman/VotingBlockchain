package networking

import (
	"net"
	"sync"
	"time"

	models "github.com/nivschuman/VotingBlockchain/internal/networking/models"
	peer "github.com/nivschuman/VotingBlockchain/internal/networking/peer"
)

type Network struct {
	Listener         *Listener
	Peers            map[net.Addr]*peer.Peer
	BroadcastChannel chan models.Message

	PeersMutex sync.Mutex
}

func (*Network) StartNetwork() {
	//TBD must go over all peers in database and dial them...
	//TBD must start listener
}

// TBD this sucks
// peer should handle this himself
func (network *Network) RemoveInactivePeers() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		<-ticker.C
		now := time.Now()

		network.PeersMutex.Lock()

		for _, peer := range network.Peers {
			if now.Sub(peer.LastMessageTime) > 2*time.Minute {
				network.RemovePeer(peer)
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
		if peer.CompletedHandshake() {
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
	network.Peers[conn.RemoteAddr()] = peer.NewPeer(conn, network.BroadcastChannel, true)
	network.PeersMutex.Unlock()
}

func (network *Network) RemovePeer(peer *peer.Peer) {
	peer.Disconnect()
	delete(network.Peers, peer.Conn.RemoteAddr())
}
