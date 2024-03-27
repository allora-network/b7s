package node

import "github.com/RedBird96/b7s/models/execute"

type ChanData struct {
	res        string
	functionId string
	requestId  string
	topic      string
	data       execute.ResultMap
}

func (n *Node) CommunicatorAppLayer() chan []byte {
	return n.comChannel
}
