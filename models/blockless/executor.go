package blockless

import (
	"github.com/allora-network/b7s/models/execute"
)

type Executor interface {
	ExecuteFunction(requestID string, request execute.Request) (execute.Result, error)
}
