package pipeline

import (
	"fmt"
	"time"
)

type State string

const (
	StateInitialized State = "initialized"

	StateValidating State = "validating"
	StateValidated  State = "validated"

	StateSelectingTopic    State = "selecting_topic"
	StateTopicSelected     State = "topic_selected"
	StateSelectingCategory State = "selecting_category"
	StateCategorySelected  State = "category_selected"

	StateCreatingExecution State = "creating_execution"
	StateExecutionCreated  State = "execution_created"

	StateRenderingPrompt State = "rendering_prompt"
	StatePromptRendered  State = "prompt_rendered"
	StateGenerating      State = "generating"
	StateGenerated       State = "generated"

	StateValidatingOutput State = "validating_output"
	StateOutputValidated  State = "output_validated"

	StatePublishing State = "publishing"
	StatePublished  State = "published"

	StateRecordingStats State = "recording_stats"
	StateMarkingUsed    State = "marking_used"
	StateCompleting     State = "completing"
	StateCompleted      State = "completed"

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
		StateValidating: {
			StateValidated,
			StatePausedNoResources,
			StateFailed,
		},
		StateValidated: {
			StateSelectingTopic,
			StateFailed,
		},
		StateSelectingTopic: {
			StateTopicSelected,
			StatePausedNoResources,
			StateFailed,
		},
		StateTopicSelected: {
			StateSelectingCategory,
			StateFailed,
		},
		StateSelectingCategory: {
			StateCategorySelected,
			StateFailed,
		},
		StateCategorySelected: {
			StateCreatingExecution,
			StateFailed,
		},
		StateCreatingExecution: {
			StateExecutionCreated,
			StateFailed,
		},
		StateExecutionCreated: {
			StateRenderingPrompt,
			StateFailed,
		},
		StateRenderingPrompt: {
			StatePromptRendered,
			StateFailed,
		},
		StatePromptRendered: {
			StateGenerating,
			StateFailed,
		},
		StateGenerating: {
			StateGenerated,
			StateFailed,
		},
		StateGenerated: {
			StateValidatingOutput,
			StateFailed,
		},
		StateValidatingOutput: {
			StateOutputValidated,
			StateFailed,
		},
		StateOutputValidated: {
			StatePublishing,
			StateFailed,
		},
		StatePublishing: {
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
			StateCompleting,
			StateFailed,
		},
		StateMarkingUsed: {
			StateCompleting,
			StateFailed,
		},
		StateCompleting: {
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
