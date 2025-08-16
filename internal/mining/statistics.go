package mining

import (
	"math/big"
	"time"

	"github.com/nivschuman/VotingBlockchain/internal/difficulty"
)

type MiningStatistics struct {
	TotalBlocksMined        int64
	CurrentBlockHashesTried int64
	LastNonce               int64
	LastBlockTimeNs         int64 // store as nanoseconds
	CurrentNBits            uint32
	CurrentBlockStart       int64 // Unix nano timestamp
}

func (s *MiningStatistics) LastBlockTime() time.Duration {
	return time.Duration(s.LastBlockTimeNs)
}

func (s *MiningStatistics) CurrentBlockStartTime() time.Time {
	if s.CurrentBlockStart == 0 {
		return time.Time{}
	}

	return time.Unix(0, s.CurrentBlockStart)
}

func (s *MiningStatistics) CurrentHashRate() float64 {
	if s.CurrentBlockStart == 0 {
		return 0
	}

	elapsed := time.Since(time.Unix(0, s.CurrentBlockStart)).Seconds()
	if elapsed < 1e-6 {
		return 0
	}

	return float64(s.CurrentBlockHashesTried) / elapsed
}

func (s *MiningStatistics) Difficulty() float64 {
	currentTarget := difficulty.GetTargetFromNBits(s.CurrentNBits)
	minTarget := difficulty.GetTargetFromNBits(difficulty.MINIMUM_DIFFICULTY)

	r := new(big.Rat).SetFrac(minTarget, currentTarget)
	f, _ := r.Float64()
	return f
}
