package mining

import "github.com/nivschuman/VotingBlockchain/internal/models"

type DisabledMiner struct{}

func NewDisabledMiner() *DisabledMiner {
	return &DisabledMiner{}
}

func (m *DisabledMiner) AddHandler(blockHandler BlockHandler) {
	// no-op
}

func (m *DisabledMiner) Start() {
	// no-op
}

func (m *DisabledMiner) MineBlockTemplate(blockTemplate *models.Block) {
	// no-op
}

func (m *DisabledMiner) CreateBlockTemplate() (*models.Block, error) {
	// Return empty block or nil
	return nil, nil
}

func (m *DisabledMiner) Stop() {
	// no-op
}

func (m *DisabledMiner) GetMiningStatistics() MiningStatistics {
	return MiningStatistics{
		TotalBlocksMined:        0,
		CurrentBlockHashesTried: 0,
		LastNonce:               0,
		LastBlockTimeNs:         0,
		CurrentNBits:            0,
		CurrentBlockStart:       0,
	}
}
