package node

import (
	"github.com/RedBird96/b7s/models/codes"
	"github.com/RedBird96/b7s/models/execute"
)

type ChanData struct {
	res        codes.Code        `json:"res,omitempty"`
	functionId string            `json:"functionId,omitempty"`
	requestId  string            `json:"requestId,omitempty"`
	topic      string            `json:"topic,omitempty"`
	data       execute.ResultMap `json:"data,omitempty"`
}

func (n *Node) CommunicatorAppLayer() chan []byte {
	return n.comChannel
}
