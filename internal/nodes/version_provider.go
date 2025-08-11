package nodes

import (
	"time"

	config "github.com/nivschuman/VotingBlockchain/internal/config"
	repositories "github.com/nivschuman/VotingBlockchain/internal/database/repositories"
	networking_models "github.com/nivschuman/VotingBlockchain/internal/networking/models"
)

type VersionProvider struct {
	blockRepo  repositories.BlockRepository
	nodeConfig config.NodeConfig
}

func NewVersionProvider(blockRepo repositories.BlockRepository, nodeConfig config.NodeConfig) *VersionProvider {
	return &VersionProvider{blockRepo: blockRepo, nodeConfig: nodeConfig}
}

func (vp *VersionProvider) GetVersion() (*networking_models.Version, error) {
	now := time.Now().Unix()
	lastBlockHeight, err := vp.blockRepo.GetActiveChainHeight()
	if err != nil {
		return nil, err
	}

	version := &networking_models.Version{
		ProtocolVersion: vp.nodeConfig.Version,
		NodeType:        vp.nodeConfig.Type,
		Timestamp:       now,
		Nonce:           0,
		LastBlockHeight: uint32(lastBlockHeight),
	}
	return version, nil
}
