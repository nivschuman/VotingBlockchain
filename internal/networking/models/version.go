package networking_models

import (
	"bytes"
	"encoding/binary"
	"time"

	config "github.com/nivschuman/VotingBlockchain/internal/config"
)

type Version struct {
	ProtocolVersion int32  //protocol version used by node
	NodeType        uint32 //type of node - full node or just transaction sender
	Timestamp       int64  //UNIX timestamp of node
	Nonce           uint64 //Random nonce for version packet
	LastBlockHeight uint32 //Height of last block in active chain of node
}

func (version *Version) AsBytes() []byte {
	buf := new(bytes.Buffer)

	binary.Write(buf, binary.BigEndian, version.ProtocolVersion)
	binary.Write(buf, binary.BigEndian, version.NodeType)
	binary.Write(buf, binary.BigEndian, version.Timestamp)
	binary.Write(buf, binary.BigEndian, version.Nonce)
	binary.Write(buf, binary.BigEndian, version.LastBlockHeight)

	return buf.Bytes()
}

func VersionFromBytes(bytes []byte) *Version {
	if len(bytes) < 28 {
		return nil
	}

	protocolVersion := int32(binary.BigEndian.Uint32(bytes[0:4]))
	nodeType := binary.BigEndian.Uint32(bytes[4:8])
	timestamp := int64(binary.BigEndian.Uint64(bytes[8:16]))
	nonce := binary.BigEndian.Uint64(bytes[16:24])
	lastBlockHeight := binary.BigEndian.Uint32(bytes[24:28])

	return &Version{
		ProtocolVersion: protocolVersion,
		NodeType:        nodeType,
		Timestamp:       timestamp,
		Nonce:           nonce,
		LastBlockHeight: lastBlockHeight,
	}
}

func NewVersionMessage(version *Version) *Message {
	return NewMessage(CommandVersion, version.AsBytes())
}

func MyVersion() *Version {
	//TBD add current block height
	return &Version{
		ProtocolVersion: config.GlobalConfig.NodeConfig.Version,
		NodeType:        config.GlobalConfig.NodeConfig.Type,
		Timestamp:       time.Now().Unix(),
	}
}
