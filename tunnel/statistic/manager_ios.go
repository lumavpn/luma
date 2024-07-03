//go:build ios

package statistic

func (m *Manager) Memory() uint64 {
	return m.memory
}
