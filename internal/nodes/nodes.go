package nodes

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
	CreateNode(NodeType) Node
}

type nodeFactoryImpl struct {
}

var GlobalNodeFactory NodeFactory = &nodeFactoryImpl{}

func (*nodeFactoryImpl) CreateNode(nodeType NodeType) Node {
	switch nodeType {
	case FULL_NODE:
		return NewFullNode(true)
	default:
		return nil
	}
}
