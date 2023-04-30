package ptproc

type Node interface {
	isNode()

	Text() string
}

type node struct {
	text string
}

func (*node) isNode() {}

func (n *node) Text() string {
	return n.text
}

type MapFileNode struct {
	Node
	ImportFile string
}

type MapFileEndNode struct {
	Node
}
