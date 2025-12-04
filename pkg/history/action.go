package history

import "context"

// Action represents a single undoable operation.
// Actions must be self-contained and able to reverse themselves.
type Action interface {
	// Do executes the action. Called during redo.
	Do(ctx context.Context) error

	// Undo reverses the action.
	Undo(ctx context.Context) error

	// Description returns a human-readable description of the action.
	Description() string
}

// Transaction groups multiple actions that should be undone/redone together.
// For example, AI generation creates many nodes but should undo as one operation.
type Transaction struct {
	// Actions in execution order (undo will reverse this order)
	Actions []Action

	// Description for the entire transaction (shown in UI)
	Label string
}

// NewTransaction creates a new transaction with the given label.
func NewTransaction(label string) *Transaction {
	return &Transaction{
		Label:   label,
		Actions: make([]Action, 0),
	}
}

// Add appends an action to the transaction.
func (t *Transaction) Add(action Action) {
	t.Actions = append(t.Actions, action)
}

// IsEmpty returns true if the transaction has no actions.
func (t *Transaction) IsEmpty() bool {
	return len(t.Actions) == 0
}

// Do executes all actions in order.
func (t *Transaction) Do(ctx context.Context) error {
	for _, action := range t.Actions {
		if err := action.Do(ctx); err != nil {
			return err
		}
	}
	return nil
}

// Undo reverses all actions in reverse order.
func (t *Transaction) Undo(ctx context.Context) error {
	for i := len(t.Actions) - 1; i >= 0; i-- {
		if err := t.Actions[i].Undo(ctx); err != nil {
			return err
		}
	}
	return nil
}

// Description returns the transaction label or generates one from actions.
func (t *Transaction) Description() string {
	if t.Label != "" {
		return t.Label
	}
	if len(t.Actions) == 1 {
		return t.Actions[0].Description()
	}
	return "Multiple actions"
}
