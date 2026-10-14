package cli

import "time"

type WaitDecision string

const (
	WaitContinue WaitDecision = "continue"
	WaitStopIdle WaitDecision = "stop-idle"
	WaitStopNone WaitDecision = "stop-none"
)

type WaitState struct {
	StartedAt   time.Time
	SeenWorking bool
	IdleSince   time.Time
}

func DecideWait(state WaitState, working bool, now time.Time, startGrace time.Duration, idleGrace time.Duration) (WaitState, WaitDecision) {
	if working {
		state.SeenWorking = true
		state.IdleSince = time.Time{}
		return state, WaitContinue
	}

	if !state.SeenWorking {
		if now.Sub(state.StartedAt) >= startGrace {
			return state, WaitStopNone
		}
		return state, WaitContinue
	}

	if state.IdleSince.IsZero() {
		state.IdleSince = now
	}
	if now.Sub(state.IdleSince) >= idleGrace {
		return state, WaitStopIdle
	}
	return state, WaitContinue
}
