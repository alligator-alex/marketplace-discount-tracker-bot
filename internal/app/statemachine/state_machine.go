package statemachine

import (
	"errors"
	"sync"
)

var ErrUnknownEvent error = errors.New("unknown event")
var ErrUnsupportedTransition error = errors.New("unsupported transition from current state")

type State string
type Event string

const StateIdle State = "Idle"
const EventIdle Event = "Idle"

type Transition struct {
	From []State
	To   State
}

type TransitionsList map[Event]Transition

type StateMachine struct {
	mutex       *sync.Mutex
	initState   State
	currState   State
	Transitions TransitionsList
}

// Create new StateMachine instance.
func NewFSM(initState State, transitions TransitionsList) StateMachine {
	return StateMachine{
		mutex:       new(sync.Mutex),
		initState:   initState,
		currState:   initState,
		Transitions: transitions,
	}
}

// Get current state name.
func (sm *StateMachine) GetCurrentState() State {
	if sm.currState == "" {
		return StateIdle
	}

	return sm.currState
}

// Check if state machine is one of states.
func (sm *StateMachine) IsInOneOfStates(states []State) bool {
	currentState := sm.GetCurrentState()

	for _, state := range states {
		if currentState == state {
			return true
		}
	}

	return false
}

// Trigger an event to make a transition to state.
func (sm *StateMachine) TriggerEvent(event Event) (State, error) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if event == EventIdle {
		return sm.currState, nil
	}

	transition, ok := sm.Transitions[event]
	if !ok || transition.From == nil || len(transition.From) == 0 {
		return sm.currState, ErrUnknownEvent
	}

	for _, state := range transition.From {
		if state != sm.currState {
			continue
		}

		sm.currState = transition.To

		return sm.currState, nil
	}

	return sm.currState, ErrUnsupportedTransition
}

// Reset state machine to it's initial state.
func (sm *StateMachine) Reset() {
	sm.currState = sm.initState
}

// Check if state machine is initialized.
func (sm *StateMachine) IsInitialized() bool {
	return sm.mutex != nil
}
