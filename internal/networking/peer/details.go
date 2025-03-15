package networking_peer

import (
	"time"

	models "github.com/nivschuman/VotingBlockchain/internal/networking/models"
	ip "github.com/nivschuman/VotingBlockchain/internal/networking/utils/ip"
)

type PeerDetails struct {
	ProtocolVersion int32  //protocol version of peer
	NodeType        uint32 //node type of peer
	Ip              string //Ip of peer
	Port            uint16 //port of peer
	TimeOffset      int64  //difference between local time and peer time in seconds
	BlockHeight     uint32 //estimate of peers current block height
}

func NewPeerDetailsFromVersion(version *models.Version) *PeerDetails {
	return &PeerDetails{
		ProtocolVersion: version.ProtocolVersion,
		NodeType:        version.NodeType,
		Ip:              ip.Uint32ToIPv4(version.Ip),
		Port:            version.Port,
		TimeOffset:      time.Now().Unix() - version.Timestamp,
		BlockHeight:     version.LastBlockHeight,
	}
}
