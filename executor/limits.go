package executor

import (
	"github.com/RedBird96/b7s/models/execute"
)

type Limiter interface {
	LimitProcess(proc execute.ProcessID) error
	ListProcesses() ([]int, error)
}
