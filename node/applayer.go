package node

import (
	"github.com/RedBird96/b7s/models/codes"
	"github.com/RedBird96/b7s/models/execute"
)

type ChanData struct {
	Res        codes.Code        `json:"res,omitempty"`
	FunctionId string            `json:"functionId,omitempty"`
	RequestId  string            `json:"requestId,omitempty"`
	Topic      string            `json:"topic,omitempty"`
	Data       execute.ResultMap `json:"data,omitempty"`
}

func (n *Node) CommunicatorAppLayer() chan []byte {
	return n.comChannel
}
