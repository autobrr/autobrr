package featureflags

import "sync"

type Manager struct {
	mu    sync.RWMutex
	flags map[string]bool
}

var defaultManager = Manager{
	flags: map[string]bool{},
}

func (m *Manager) IsEnabled(flag string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	v, ok := m.flags[flag]
	if !ok {
		return false
	}
	return v
}

func (m *Manager) SetEnabled(flag string, enabled bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.flags[flag] = enabled
}

func (m *Manager) Register(flag string, defaultValue bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.flags[flag]; !exists {
		m.flags[flag] = defaultValue
	}
}

func (m *Manager) Exists(flag string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.flags[flag]
	return exists
}

func (m *Manager) GetAll() map[string]bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	res := make(map[string]bool, len(m.flags))
	for k, v := range m.flags {
		res[k] = v
	}
	return res
}

func IsEnabled(flag string) bool {
	return defaultManager.IsEnabled(flag)
}

func SetEnabled(flag string, enabled bool) {
	defaultManager.SetEnabled(flag, enabled)
}

func Register(flag string, defaultValue bool) {
	defaultManager.Register(flag, defaultValue)
}

func Exists(flag string) bool {
	return defaultManager.Exists(flag)
}

func GetAll() map[string]bool {
	return defaultManager.GetAll()
}
