package networking_models

import (
	"bytes"
	"encoding/binary"
	"time"

	config "github.com/nivschuman/VotingBlockchain/internal/config"
	ip "github.com/nivschuman/VotingBlockchain/internal/networking/utils/ip"
)

type Version struct {
	ProtocolVersion int32  //protocol version used by node
	NodeType        uint32 //type of node - full node or just transaction sender
	Timestamp       int64  //UNIX timestamp of node
	Ip              uint32 //Ipv4 of node to use for future connections and in database
	Port            uint16 //Port of node to use for future connections and in database
	Nonce           uint64 //Random nonce for version packet
	LastBlockHeight uint32 //Height of last block in active chain of node
}

func (version *Version) AsBytes() []byte {
	buf := new(bytes.Buffer)

	binary.Write(buf, binary.BigEndian, version.ProtocolVersion)
	binary.Write(buf, binary.BigEndian, version.NodeType)
	binary.Write(buf, binary.BigEndian, version.Timestamp)
	binary.Write(buf, binary.BigEndian, version.Ip)
	binary.Write(buf, binary.BigEndian, version.Port)
	binary.Write(buf, binary.BigEndian, version.Nonce)
	binary.Write(buf, binary.BigEndian, version.LastBlockHeight)

	return buf.Bytes()
}

func VersionFromBytes(bytes []byte) *Version {
	if len(bytes) < 34 {
		return nil
	}

	protocolVersion := int32(binary.BigEndian.Uint32(bytes[0:4]))
	nodeType := binary.BigEndian.Uint32(bytes[4:8])
	timestamp := int64(binary.BigEndian.Uint64(bytes[8:16]))
	ip := binary.BigEndian.Uint32(bytes[16:20])
	port := binary.BigEndian.Uint16(bytes[20:22])
	nonce := binary.BigEndian.Uint64(bytes[22:30])
	lastBlockHeight := binary.BigEndian.Uint32(bytes[30:34])

	return &Version{
		ProtocolVersion: protocolVersion,
		NodeType:        nodeType,
		Timestamp:       timestamp,
		Ip:              ip,
		Port:            port,
		Nonce:           nonce,
		LastBlockHeight: lastBlockHeight,
	}
}

func NewVersionMessage(version *Version) *Message {
	return NewMessage(CommandVersion, version.AsBytes())
}

func MyVersion() (*Version, error) {
	uint32Ip, err := ip.Ipv4ToUint32(config.GlobalConfig.NetworkConfig.Ip)

	if err != nil {
		return nil, err
	}

	//TBD add current block height
	version := &Version{
		ProtocolVersion: config.GlobalConfig.NodeConfig.Version,
		NodeType:        config.GlobalConfig.NodeConfig.Type,
		Timestamp:       time.Now().Unix(),
		Ip:              uint32Ip,
		Port:            config.GlobalConfig.NetworkConfig.Port,
	}

	return version, nil
}
