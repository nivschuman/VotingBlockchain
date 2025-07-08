package nodes

import (
	"fmt"

	config "github.com/nivschuman/VotingBlockchain/internal/config"
)

type NodeType int

const (
	FULL_NODE NodeType = iota + 1
	SPV_CLIENT
)

type Node interface {
	Start()
	Stop()
}

type NodeFactory interface {
	CreateNode(NodeType) (Node, error)
}

type nodeFactoryImpl struct {
}

var GlobalNodeFactory NodeFactory = &nodeFactoryImpl{}

func (*nodeFactoryImpl) CreateNode(nodeType NodeType) (Node, error) {
	switch nodeType {
	case FULL_NODE:
		return NewFullNode(config.GlobalConfig.MinerConfig.Enabled), nil
	default:
		return nil, fmt.Errorf("unsupported node type: %v", nodeType)
	}
}
