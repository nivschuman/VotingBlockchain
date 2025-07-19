package networking_peer

import (
	"time"

	models "github.com/nivschuman/VotingBlockchain/internal/networking/models"
)

type PeerDetails struct {
	ProtocolVersion int32  //protocol version of peer
	TimeOffset      int64  //difference between local time and peer time in seconds
	BlockHeight     uint32 //peers block height
}

func (peer *Peer) SetPeerDetails(version *models.Version) {
	peer.PeerDetails = &PeerDetails{
		ProtocolVersion: version.ProtocolVersion,
		TimeOffset:      time.Now().Unix() - version.Timestamp,
		BlockHeight:     version.LastBlockHeight,
	}
}
