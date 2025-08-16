package nodes

import (
	"fmt"

	config "github.com/nivschuman/VotingBlockchain/internal/config"
	database "github.com/nivschuman/VotingBlockchain/internal/database/connection"
	"github.com/nivschuman/VotingBlockchain/internal/database/repositories"
	"github.com/nivschuman/VotingBlockchain/internal/mining"
	"github.com/nivschuman/VotingBlockchain/internal/networking/network"
	"gorm.io/gorm"
)

const FULL_NODE = uint32(1)

type Node interface {
	Start()
	Stop()
	AddShutdownHook(func() error)
	GetMiner() mining.Miner
}

type NodeBuilder interface {
	BuildNode() (Node, error)
	GetDatabase() *gorm.DB
}

type NodeBuilderImpl struct {
	db                    *gorm.DB
	blockRepository       repositories.BlockRepository
	transactionRepository repositories.TransactionRepository
	addressRepository     repositories.AddressRepository
	miner                 mining.Miner
	network               *network.NetworkImpl
	config                *config.Config
}

func NewNodeBuilderImpl(config *config.Config) (*NodeBuilderImpl, error) {
	db, err := database.GetDatabaseConnection(config.DatabaseConfig.File)
	if err != nil {
		return nil, err
	}

	transactionRepository := repositories.NewTransactionRepositoryImpl(db)
	blockRepository := repositories.NewBlockRepositoryImpl(db, transactionRepository)
	if err := blockRepository.Initialize(); err != nil {
		return nil, err
	}

	addressRepository := repositories.NewAddressRepositoryImpl(db)
	versionProvider := NewVersionProvider(blockRepository, config.NodeConfig)
	netwrk := network.NewNetworkImpl(addressRepository, &config.NetworkConfig, versionProvider.GetVersion)

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
		db:                    db,
		blockRepository:       blockRepository,
		transactionRepository: transactionRepository,
		addressRepository:     addressRepository,
		miner:                 miner,
		network:               netwrk,
		config:                config,
	}, nil
}

func (nodeBuilder *NodeBuilderImpl) BuildNode() (Node, error) {
	var node Node

	nodeType := nodeBuilder.config.NodeConfig.Type
	switch nodeType {
	case FULL_NODE:
		node = NewFullNode(nodeBuilder.network, nodeBuilder.miner, nodeBuilder.blockRepository, nodeBuilder.transactionRepository, nodeBuilder.config.GovernmentConfig.PublicKey)
	default:
		return nil, fmt.Errorf("unsupported node type: %v", nodeType)
	}

	node.AddShutdownHook(func() error {
		return database.CloseDatabaseConnection(nodeBuilder.db)
	})

	return node, nil
}

func (nodeBuilder *NodeBuilderImpl) GetDatabase() *gorm.DB {
	return nodeBuilder.db
}
