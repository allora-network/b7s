package executor

import (
	"github.com/allora-network/b7s/models/execute"
)

type Limiter interface {
	LimitProcess(proc execute.ProcessID) error
	ListProcesses() ([]int, error)
}
