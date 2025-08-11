package networking_mocks

import (
	"time"

	models "github.com/nivschuman/VotingBlockchain/internal/networking/models"
	inits "github.com/nivschuman/VotingBlockchain/tests/init"
)

func MockVersionProvider() (*models.Version, error) {
	return &models.Version{
		ProtocolVersion: inits.TestConfig.NodeConfig.Version,
		NodeType:        inits.TestConfig.NodeConfig.Type,
		Timestamp:       time.Now().Unix(),
		Nonce:           0,
	}, nil
}
