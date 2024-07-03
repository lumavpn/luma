//go:build !ios

package statistic

import (
	"os"

	"github.com/shirou/gopsutil/v4/process"
)

var mProcess *process.Process

func init() {
	mProcess = &process.Process{Pid: int32(os.Getpid())}
}

func (m *Manager) Memory() uint64 {
	m.updateMemory()
	return m.memory
}

func (m *Manager) updateMemory() {
	stat, err := mProcess.MemoryInfo()
	if err != nil {
		return
	}
	m.memory = stat.RSS
}
