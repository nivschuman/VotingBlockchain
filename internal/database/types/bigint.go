package db_types

import (
	"database/sql/driver"
	"fmt"
	"math/big"
)

type BigInt big.Int

func (bi *BigInt) Scan(value any) error {
	if value == nil {
		*bi = BigInt(*big.NewInt(0))
		return nil
	}

	switch v := value.(type) {
	case []byte:
		*bi = BigInt(*new(big.Int).SetBytes(v))
		return nil
	default:
		return fmt.Errorf("failed to scan CumulativeWork: unsupported type %T", value)
	}
}

func (bi BigInt) Value() (driver.Value, error) {
	return []byte((*big.Int)(&bi).Bytes()), nil
}

func (bi BigInt) Add(other BigInt) BigInt {
	result := new(big.Int).Add((*big.Int)(&bi), (*big.Int)(&other))
	return BigInt(*result)
}

func (bi BigInt) Cmp(other BigInt) int {
	return (*big.Int)(&bi).Cmp((*big.Int)(&other))
}

func NewBigInt(value *big.Int) BigInt {
	return BigInt(*value)
}
