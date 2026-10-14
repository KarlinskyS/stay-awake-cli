package cli

import (
	"testing"
	"time"
)

func TestDecideWait(t *testing.T) {
	start := time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC)
	state := WaitState{StartedAt: start}

	state, got := DecideWait(state, true, start, 10*time.Minute, time.Minute)
	if got != WaitContinue {
		t.Fatalf("working at start = %s, want continue", got)
	}
	if !state.SeenWorking {
		t.Fatal("SeenWorking = false, want true")
	}

	idleAt := start.Add(3 * time.Hour)
	state, got = DecideWait(state, false, idleAt, 10*time.Minute, time.Minute)
	if got != WaitContinue {
		t.Fatalf("first idle tick = %s, want continue", got)
	}
	if state.IdleSince != idleAt {
		t.Fatalf("IdleSince = %s, want %s", state.IdleSince, idleAt)
	}

	_, got = DecideWait(state, false, idleAt.Add(time.Minute), 10*time.Minute, time.Minute)
	if got != WaitStopIdle {
		t.Fatalf("idle for grace = %s, want stop-idle", got)
	}
}

func TestDecideWaitStopsWhenNoAgentAppears(t *testing.T) {
	start := time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC)
	state := WaitState{StartedAt: start}

	state, got := DecideWait(state, false, start.Add(9*time.Minute), 10*time.Minute, time.Minute)
	if got != WaitContinue {
		t.Fatalf("before start grace = %s, want continue", got)
	}

	_, got = DecideWait(state, false, start.Add(10*time.Minute), 10*time.Minute, time.Minute)
	if got != WaitStopNone {
		t.Fatalf("after start grace = %s, want stop-none", got)
	}
}

func TestDecideWaitZeroIdleGraceStopsImmediately(t *testing.T) {
	start := time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC)
	state := WaitState{StartedAt: start, SeenWorking: true}

	_, got := DecideWait(state, false, start, 10*time.Minute, 0)
	if got != WaitStopIdle {
		t.Fatalf("zero idle grace = %s, want stop-idle", got)
	}
}

func TestDecideWaitWorkingResetsIdleClock(t *testing.T) {
	start := time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC)
	state := WaitState{StartedAt: start, SeenWorking: true, IdleSince: start}

	state, got := DecideWait(state, true, start.Add(30*time.Second), 10*time.Minute, time.Minute)
	if got != WaitContinue {
		t.Fatalf("working again = %s, want continue", got)
	}
	if !state.IdleSince.IsZero() {
		t.Fatalf("IdleSince = %s, want zero", state.IdleSince)
	}
}
