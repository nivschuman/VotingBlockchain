package difficulty

import (
	"math/big"
)

// Minimum nBits difficulty allowed for block
var MINIMUM_DIFFICULTY = uint32(0x1d00ffff)

// Total time expected for interval in seconds
var TARGET_TIMESPAN = int64(14 * 24 * 60 * 60)

// Time expected per block in seconds
var TARGET_SPACING = int64(10 * 60)

// Number of blocks between difficulty adjustments
var INTERVAL = TARGET_TIMESPAN / TARGET_SPACING

// Minimum timespan allowed
var MIN_TIMESPAN = TARGET_TIMESPAN / 4

// Maximum timespan allowed
var MAX_TIMESPAN = TARGET_TIMESPAN * 4

func GetTargetFromNBits(nBits uint32) *big.Int {
	exponent := nBits >> 24
	coefficient := nBits & 0x00FFFFFF
	target := big.NewInt(int64(coefficient))

	if exponent > 3 {
		target.Lsh(target, uint(8*(exponent-3)))
	} else if exponent < 3 {
		target.Rsh(target, uint(8*(3-exponent)))
	}

	return target
}

func TargetToNBits(target *big.Int) uint32 {
	if target.Sign() == 0 {
		return 0
	}

	tmp := new(big.Int).Set(target)
	targetBytes := tmp.Bytes()

	if targetBytes[0]&0x80 != 0 {
		targetBytes = append([]byte{0x00}, targetBytes...)
	}

	size := len(targetBytes)
	coefficient := uint32(0)
	if size >= 1 {
		coefficient |= uint32(targetBytes[0]) << 16
	}
	if size >= 2 {
		coefficient |= uint32(targetBytes[1]) << 8
	}
	if size >= 3 {
		coefficient |= uint32(targetBytes[2])
	}

	nBits := uint32(size<<24) | coefficient
	return nBits
}

func IsHashBelowTarget(hash []byte, target *big.Int) bool {
	blockBigInt := new(big.Int).SetBytes(hash)
	return blockBigInt.Cmp(target) <= 0
}

func CalculateWork(nBits uint32) *big.Int {
	target := GetTargetFromNBits(nBits)

	one := big.NewInt(1)
	denominator := new(big.Int).Add(target, one)
	numerator := new(big.Int).Lsh(one, 256)

	return new(big.Int).Div(numerator, denominator)
}
