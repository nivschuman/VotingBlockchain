package difficulty

import "math/big"

func GetTargetFromNBits(nBits uint32) *big.Int {
	// Extract exponent (first byte)
	exponent := nBits >> 24

	// Extract coefficient (lower 3 bytes)
	coefficient := nBits & 0x00FFFFFF

	// Initialize a big integer for the target
	target := big.NewInt(int64(coefficient))

	// Shift the coefficient by (exponent - 3) to adjust the target
	if exponent > 3 {
		// Left shift by (exponent - 3) bytes, equivalent to multiplying by 256^(exponent-3)
		target.Lsh(target, uint(8*(exponent-3)))
	}

	// Return the target as a big integer
	return target
}

func IsHashBelowTarget(hash []byte, target *big.Int) bool {
	blockBigInt := new(big.Int).SetBytes(hash)
	return blockBigInt.Cmp(blockBigInt) <= 0
}

func CalculateWork(nBits uint32) *big.Int {
	target := GetTargetFromNBits(nBits)

	one := big.NewInt(1)
	denominator := new(big.Int).Add(target, one)

	numerator := new(big.Int).Lsh(one, 256) // 2^256

	return new(big.Int).Div(numerator, denominator)
}
