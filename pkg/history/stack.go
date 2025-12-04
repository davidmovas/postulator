package history

import (
	"context"
	"sync"
)

// Stack maintains undo/redo stacks for a single source (e.g., one sitemap).
type Stack struct {
	mu       sync.RWMutex
	undo     []*Transaction
	redo     []*Transaction
	maxSize  int
	isLocked bool // Prevents recording during undo/redo
}

// NewStack creates a new history stack with the given max size.
func NewStack(maxSize int) *Stack {
	if maxSize <= 0 {
		maxSize = 25
	}
	return &Stack{
		undo:    make([]*Transaction, 0, maxSize),
		redo:    make([]*Transaction, 0, maxSize),
		maxSize: maxSize,
	}
}

// Record adds a transaction to the undo stack.
// Clears the redo stack (new action invalidates redo history).
// Does nothing if stack is locked (during undo/redo operations).
func (s *Stack) Record(tx *Transaction) {
	if tx == nil || tx.IsEmpty() {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isLocked {
		return
	}

	// Add to undo stack
	s.undo = append(s.undo, tx)

	// Enforce max size (remove oldest)
	if len(s.undo) > s.maxSize {
		s.undo = s.undo[1:]
	}

	// Clear redo stack
	s.redo = s.redo[:0]
}

// Undo pops the last transaction and executes its undo.
// Returns the transaction description or empty string if nothing to undo.
func (s *Stack) Undo(ctx context.Context) (string, error) {
	s.mu.Lock()
	if len(s.undo) == 0 {
		s.mu.Unlock()
		return "", nil
	}

	// Pop from undo
	tx := s.undo[len(s.undo)-1]
	s.undo = s.undo[:len(s.undo)-1]

	// Lock to prevent recording during undo
	s.isLocked = true
	s.mu.Unlock()

	// Execute undo
	err := tx.Undo(ctx)

	s.mu.Lock()
	s.isLocked = false
	if err == nil {
		// Push to redo
		s.redo = append(s.redo, tx)
	}
	s.mu.Unlock()

	if err != nil {
		return "", err
	}
	return tx.Description(), nil
}

// Redo pops the last undone transaction and re-executes it.
// Returns the transaction description or empty string if nothing to redo.
// If a transaction cannot be redone (e.g., batch operations), it is silently discarded.
func (s *Stack) Redo(ctx context.Context) (string, error) {
	s.mu.Lock()
	if len(s.redo) == 0 {
		s.mu.Unlock()
		return "", nil
	}

	// Pop from redo
	tx := s.redo[len(s.redo)-1]
	s.redo = s.redo[:len(s.redo)-1]

	// Lock to prevent recording during redo
	s.isLocked = true
	s.mu.Unlock()

	// Execute do
	err := tx.Do(ctx)

	s.mu.Lock()
	s.isLocked = false
	if err == nil {
		// Push back to undo
		s.undo = append(s.undo, tx)
	}
	// If redo fails (e.g., batch operations that can't be redone),
	// the transaction is simply discarded - not returned to redo stack
	s.mu.Unlock()

	if err != nil {
		// Don't return error - just indicate nothing was redone
		return "", nil
	}
	return tx.Description(), nil
}

// CanUndo returns true if there are actions to undo.
func (s *Stack) CanUndo() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.undo) > 0
}

// CanRedo returns true if there are actions to redo.
func (s *Stack) CanRedo() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.redo) > 0
}

// UndoCount returns the number of undoable transactions.
func (s *Stack) UndoCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.undo)
}

// RedoCount returns the number of redoable transactions.
func (s *Stack) RedoCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.redo)
}

// LastActionDescription returns the description of the last undoable action.
func (s *Stack) LastActionDescription() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.undo) == 0 {
		return ""
	}
	return s.undo[len(s.undo)-1].Description()
}

// Clear removes all undo/redo history.
func (s *Stack) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.undo = s.undo[:0]
	s.redo = s.redo[:0]
}

// State returns the current state of the stack.
type State struct {
	CanUndo    bool   `json:"canUndo"`
	CanRedo    bool   `json:"canRedo"`
	UndoCount  int    `json:"undoCount"`
	RedoCount  int    `json:"redoCount"`
	LastAction string `json:"lastAction,omitempty"`
}

// GetState returns the current state for UI consumption.
func (s *Stack) GetState() State {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state := State{
		CanUndo:   len(s.undo) > 0,
		CanRedo:   len(s.redo) > 0,
		UndoCount: len(s.undo),
		RedoCount: len(s.redo),
	}
	if len(s.undo) > 0 {
		state.LastAction = s.undo[len(s.undo)-1].Description()
	}
	return state
}
