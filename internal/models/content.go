package models

import (
	"github.com/nivschuman/VotingBlockchain/internal/crypto/hash"
)

type Byteable interface {
	AsBytes() []byte
}

type Content interface {
	hash.Hashable
	Byteable
}
