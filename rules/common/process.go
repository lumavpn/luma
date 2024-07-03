package common

import (
	"strings"

	M "github.com/lumavpn/luma/metadata"
	R "github.com/lumavpn/luma/rule"
)

type Process struct {
	*Base
	adapter  string
	process  string
	nameOnly bool
}

func (ps *Process) RuleType() R.RuleType {
	if ps.nameOnly {
		return R.Process
	}

	return R.ProcessPath
}

func (ps *Process) Match(metadata *M.Metadata) (bool, string) {
	if ps.nameOnly {
		return strings.EqualFold(metadata.Process, ps.process), ps.adapter
	}

	return strings.EqualFold(metadata.ProcessPath, ps.process), ps.adapter
}

func (ps *Process) Adapter() string {
	return ps.adapter
}

func (ps *Process) Payload() string {
	return ps.process
}

func (ps *Process) ShouldFindProcess() bool {
	return true
}

func NewProcess(process string, adapter string, nameOnly bool) (*Process, error) {
	return &Process{
		Base:     &Base{},
		adapter:  adapter,
		process:  process,
		nameOnly: nameOnly,
	}, nil
}
