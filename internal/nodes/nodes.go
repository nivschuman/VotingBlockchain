package nodes

import (
	"fmt"

	config "github.com/nivschuman/VotingBlockchain/internal/config"
	"github.com/nivschuman/VotingBlockchain/internal/database/repositories"
	"github.com/nivschuman/VotingBlockchain/internal/mining"
	"github.com/nivschuman/VotingBlockchain/internal/networking/network"
	"gorm.io/gorm"
)

const FULL_NODE = uint32(1)

type Node interface {
	Start()
	Stop()
}

type NodeBuilder interface {
	BuildNode() (Node, error)
}

type NodeBuilderImpl struct {
	blockRepository       repositories.BlockRepository
	transactionRepository repositories.TransactionRepository
	addressRepository     repositories.AddressRepository
	miner                 mining.Miner
	network               *network.NetworkImpl
	config                *config.Config
}

func NewNodeBuilderImpl(db *gorm.DB, config *config.Config) (*NodeBuilderImpl, error) {
	transactionRepository := repositories.NewTransactionRepositoryImpl(db)
	blockRepository := repositories.NewBlockRepositoryImpl(db, transactionRepository)
	if err := blockRepository.Initialize(); err != nil {
		return nil, err
	}

	addressRepository := repositories.NewAddressRepositoryImpl(db)
	versionProvider := NewVersionProvider(blockRepository, config.NodeConfig)
	netwrk := network.NewNetworkImpl(
		config.NetworkConfig.Ip,
		config.NetworkConfig.Port,
		addressRepository,
		&config.NetworkConfig,
		versionProvider.GetVersion,
	)

	minerProps := mining.MinerProperties{
		NodeVersion:    config.NodeConfig.Version,
		MinerPublicKey: config.GovernmentConfig.PublicKey,
	}

	var miner mining.Miner
	miner = mining.NewMinerImpl(netwrk.GetNetworkTime, blockRepository, transactionRepository, minerProps)
	if !config.MinerConfig.Enabled {
		miner = mining.NewDisabledMiner()
	}

	return &NodeBuilderImpl{
		blockRepository:       blockRepository,
		transactionRepository: transactionRepository,
		addressRepository:     addressRepository,
		miner:                 miner,
		network:               netwrk,
		config:                config,
	}, nil
}

func (nodeBuilder *NodeBuilderImpl) BuildNode() (Node, error) {
	nodeType := nodeBuilder.config.NodeConfig.Type

	switch nodeType {
	case FULL_NODE:
		return NewFullNode(nodeBuilder.network, nodeBuilder.miner, nodeBuilder.blockRepository, nodeBuilder.transactionRepository, nodeBuilder.config.GovernmentConfig.PublicKey), nil
	default:
		return nil, fmt.Errorf("unsupported node type: %v", nodeType)
	}
}
