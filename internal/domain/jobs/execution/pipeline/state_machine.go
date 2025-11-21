package pipeline

import (
	"fmt"
	"time"
)

type State string

const (
	StateInitialized      State = "initialized"
	StateValidated        State = "validated"
	StateTopicSelected    State = "topic_selected"
	StateCategorySelected State = "category_selected"
	StateExecutionCreated State = "execution_created"
	StatePromptRendered   State = "prompt_rendered"
	StateGenerated        State = "generated"
	StateOutputValidated  State = "output_validated"
	StatePublished        State = "published"
	StateRecordingStats   State = "recording_stats"
	StateMarkingUsed      State = "marking_used"
	StateCompleted        State = "completed"

	StatePausedForValidation State = "paused_for_validation"
	StatePausedNoResources   State = "paused_no_resources"
	StateFailed              State = "failed"
)

type StateMachine struct {
	currentState State
	history      []StateTransition
	transitions  map[State][]State
}

type StateTransition struct {
	From      State
	To        State
	Timestamp time.Time
	Reason    string
}

func NewStateMachine(initialState State) *StateMachine {
	sm := &StateMachine{
		currentState: initialState,
		history:      make([]StateTransition, 0),
		transitions:  make(map[State][]State),
	}

	sm.defineTransitions()
	return sm
}

func (sm *StateMachine) defineTransitions() {
	sm.transitions = map[State][]State{
		StateInitialized: {
			StateValidated,
			StateFailed,
		},
		StateValidated: {
			StateTopicSelected,
			StateFailed,
		},
		StateTopicSelected: {
			StateCategorySelected,
			StateFailed,
		},
		StateCategorySelected: {
			StateExecutionCreated,
			StateFailed,
		},
		StateExecutionCreated: {
			StatePromptRendered,
			StateFailed,
		},
		StatePromptRendered: {
			StateGenerated,
			StateFailed,
		},
		StateGenerated: {
			StateOutputValidated,
			StateFailed,
		},
		StateOutputValidated: {
			StatePublished,
			StatePausedForValidation,
			StateFailed,
		},
		StatePublished: {
			StateRecordingStats,
			StateFailed,
		},
		StateRecordingStats: {
			StateMarkingUsed,
			StateFailed,
		},
		StateMarkingUsed: {
			StateCompleted,
			StateFailed,
		},
		StateCompleted:           {},
		StateFailed:              {},
		StatePausedForValidation: {},
		StatePausedNoResources:   {},
	}
}

func (sm *StateMachine) CurrentState() State {
	return sm.currentState
}

func (sm *StateMachine) CanTransition(to State) bool {
	allowedStates, exists := sm.transitions[sm.currentState]
	if !exists {
		return false
	}

	for _, allowed := range allowedStates {
		if allowed == to {
			return true
		}
	}

	return false
}

func (sm *StateMachine) Transition(to State, reason string) error {
	if !sm.CanTransition(to) {
		return fmt.Errorf("invalid state transition from %s to %s", sm.currentState, to)
	}

	transition := StateTransition{
		From:      sm.currentState,
		To:        to,
		Timestamp: time.Now(),
		Reason:    reason,
	}

	sm.history = append(sm.history, transition)
	sm.currentState = to

	return nil
}

func (sm *StateMachine) History() []StateTransition {
	return sm.history
}

func (sm *StateMachine) IsFinalState() bool {
	return sm.currentState == StateCompleted ||
		sm.currentState == StateFailed ||
		sm.currentState == StatePausedForValidation ||
		sm.currentState == StatePausedNoResources
}

func (sm *StateMachine) IsErrorState() bool {
	return sm.currentState == StateFailed
}

func (sm *StateMachine) IsPausedState() bool {
	return sm.currentState == StatePausedForValidation ||
		sm.currentState == StatePausedNoResources
}
