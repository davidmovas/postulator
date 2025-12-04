package history

import (
	"context"
	"sync"
	"time"
)

// SourceKey identifies a unique history source (e.g., sitemap ID).
type SourceKey string

// Manager manages history stacks for multiple sources.
// Each source (e.g., each sitemap) has its own independent stack.
type Manager struct {
	mu      sync.RWMutex
	stacks  map[SourceKey]*stackEntry
	maxSize int
	ttl     time.Duration // Time after which inactive stacks are cleaned up
}

type stackEntry struct {
	stack      *Stack
	lastAccess time.Time
}

// NewManager creates a new history manager.
// maxSize is the maximum number of transactions per stack.
// ttl is how long to keep inactive stacks (0 = forever).
func NewManager(maxSize int, ttl time.Duration) *Manager {
	if maxSize <= 0 {
		maxSize = 25
	}

	m := &Manager{
		stacks:  make(map[SourceKey]*stackEntry),
		maxSize: maxSize,
		ttl:     ttl,
	}

	// Start cleanup goroutine if TTL is set
	if ttl > 0 {
		go m.cleanupLoop()
	}

	return m
}

// cleanupLoop periodically removes stale stacks.
func (m *Manager) cleanupLoop() {
	ticker := time.NewTicker(m.ttl / 2)
	defer ticker.Stop()

	for range ticker.C {
		m.cleanup()
	}
}

// cleanup removes stacks that haven't been accessed within TTL.
func (m *Manager) cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	cutoff := time.Now().Add(-m.ttl)
	for key, entry := range m.stacks {
		if entry.lastAccess.Before(cutoff) {
			delete(m.stacks, key)
		}
	}
}

// getOrCreateStack gets an existing stack or creates a new one.
func (m *Manager) getOrCreateStack(key SourceKey) *Stack {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, exists := m.stacks[key]
	if !exists {
		entry = &stackEntry{
			stack:      NewStack(m.maxSize),
			lastAccess: time.Now(),
		}
		m.stacks[key] = entry
	} else {
		entry.lastAccess = time.Now()
	}
	return entry.stack
}

// getStack gets an existing stack or nil if it doesn't exist.
func (m *Manager) getStack(key SourceKey) *Stack {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entry, exists := m.stacks[key]
	if !exists {
		return nil
	}
	entry.lastAccess = time.Now()
	return entry.stack
}

// Record adds a transaction to the specified source's history.
func (m *Manager) Record(key SourceKey, tx *Transaction) {
	stack := m.getOrCreateStack(key)
	stack.Record(tx)
}

// Undo executes undo for the specified source.
// Returns the description of the undone transaction.
func (m *Manager) Undo(ctx context.Context, key SourceKey) (string, error) {
	stack := m.getStack(key)
	if stack == nil {
		return "", nil
	}
	return stack.Undo(ctx)
}

// Redo executes redo for the specified source.
// Returns the description of the redone transaction.
func (m *Manager) Redo(ctx context.Context, key SourceKey) (string, error) {
	stack := m.getStack(key)
	if stack == nil {
		return "", nil
	}
	return stack.Redo(ctx)
}

// GetState returns the history state for the specified source.
func (m *Manager) GetState(key SourceKey) State {
	stack := m.getStack(key)
	if stack == nil {
		return State{}
	}
	return stack.GetState()
}

// Clear removes all history for the specified source.
func (m *Manager) Clear(key SourceKey) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.stacks, key)
}

// ClearAll removes all history for all sources.
func (m *Manager) ClearAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stacks = make(map[SourceKey]*stackEntry)
}
