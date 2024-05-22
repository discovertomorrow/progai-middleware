package llamacpp

import (
	"sync"
	"time"
)

// DataWithExpiry holds the value and the expiry time
type DataWithExpiry struct {
	Value  string
	Expiry time.Time
}

// ExpiringMap holds the map and the mutex to protect it during concurrent access
type ExpiringMap struct {
	data map[string]DataWithExpiry
	mu   sync.Mutex
}

// NewExpiringMap creates a new expiring map
func NewExpiringMap() *ExpiringMap {
	m := &ExpiringMap{
		data: make(map[string]DataWithExpiry),
	}
	go m.cleanupExpiredEntries()
	return m
}

// Set adds a key-value pair to the map with an expiry time of 6 hours
func (m *ExpiringMap) Set(key, value string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = DataWithExpiry{
		Value:  value,
		Expiry: time.Now().Add(6 * time.Hour),
	}
}

// Get retrieves a value based on the key
func (m *ExpiringMap) Get(key string) (string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	data, exists := m.data[key]
	if !exists || time.Now().After(data.Expiry) {
		return "", false
	}
	return data.Value, true
}

// cleanupExpiredEntries periodically checks and removes expired entries
func (m *ExpiringMap) cleanupExpiredEntries() {
	for {
		time.Sleep(time.Minute * 10) // Cleanup interval; you can adjust this as needed.
		m.mu.Lock()
		for key, data := range m.data {
			if time.Now().After(data.Expiry) {
				delete(m.data, key)
			}
		}
		m.mu.Unlock()
	}
}
