package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ieffai/stay-awake-cli/internal/agent"
)

type fakeActivity struct {
	idleFor time.Duration
	err     error
	trusted bool
}

func (f fakeActivity) LastInputAge() (time.Duration, error) {
	if f.err != nil {
		return 0, f.err
	}
	return f.idleFor, nil
}

func (f fakeActivity) AccessibilityTrusted() bool {
	return f.trusted
}

type fakeMover struct {
	calls    int
	distance int
	err      error
}

func (f *fakeMover) MoveAndReturn(ctx context.Context, distance int) error {
	f.calls++
	f.distance = distance
	if f.err != nil {
		return f.err
	}
	return ctx.Err()
}

func TestRunnerTickMovesWhenIdle(t *testing.T) {
	var out bytes.Buffer
	mover := &fakeMover{}
	now := time.Date(2026, 7, 21, 13, 0, 0, 0, time.UTC)
	runner := appRunner{
		cfg:      Config{Interval: time.Minute, Distance: 2},
		out:      &out,
		activity: fakeActivity{idleFor: time.Minute, trusted: true},
		mover:    mover,
		now:      func() time.Time { return now },
	}

	if err := runner.tick(context.Background()); err != nil {
		t.Fatalf("tick() error = %v", err)
	}
	if mover.calls != 1 {
		t.Fatalf("mover calls = %d, want 1", mover.calls)
	}
	if mover.distance != 2 {
		t.Fatalf("distance = %d, want 2", mover.distance)
	}
	if runner.lastSyntheticAt != now {
		t.Fatalf("lastSyntheticAt = %s, want %s", runner.lastSyntheticAt, now)
	}
	if !strings.Contains(out.String(), "idle for 60s - cursor moved") {
		t.Fatalf("log = %q, want move log", out.String())
	}
}

func TestRunnerTickSkipsWhenUserWasActive(t *testing.T) {
	var out bytes.Buffer
	mover := &fakeMover{}
	runner := appRunner{
		cfg:      Config{Interval: time.Minute, Distance: 1, Verbose: true},
		out:      &out,
		activity: fakeActivity{idleFor: 10 * time.Second, trusted: true},
		mover:    mover,
		now:      func() time.Time { return time.Date(2026, 7, 21, 13, 0, 0, 0, time.UTC) },
	}

	if err := runner.tick(context.Background()); err != nil {
		t.Fatalf("tick() error = %v", err)
	}
	if mover.calls != 0 {
		t.Fatalf("mover calls = %d, want 0", mover.calls)
	}
	if !strings.Contains(out.String(), "user activity detected after 10s idle - skipped") {
		t.Fatalf("log = %q, want verbose skip log", out.String())
	}
}

func TestRunnerTickSkipsUserActivityAfterSynthetic(t *testing.T) {
	mover := &fakeMover{}
	now := time.Date(2026, 7, 21, 13, 1, 0, 0, time.UTC)
	runner := appRunner{
		cfg:             Config{Interval: time.Minute, Distance: 1},
		out:             &bytes.Buffer{},
		activity:        fakeActivity{idleFor: 59*time.Second + 900*time.Millisecond, trusted: true},
		mover:           mover,
		now:             func() time.Time { return now },
		lastSyntheticAt: now.Add(-time.Minute),
	}

	if err := runner.tick(context.Background()); err != nil {
		t.Fatalf("tick() error = %v", err)
	}
	if mover.calls != 0 {
		t.Fatalf("mover calls = %d, want 0", mover.calls)
	}
}

func TestRunnerTickReturnsMoverError(t *testing.T) {
	moverErr := errors.New("boom")
	runner := appRunner{
		cfg:      Config{Interval: time.Minute, Distance: 1},
		out:      &bytes.Buffer{},
		activity: fakeActivity{idleFor: time.Minute, trusted: true},
		mover:    &fakeMover{err: moverErr},
		now:      func() time.Time { return time.Date(2026, 7, 21, 13, 0, 0, 0, time.UTC) },
	}

	err := runner.tick(context.Background())
	if !errors.Is(err, moverErr) {
		t.Fatalf("tick() error = %v, want mover error", err)
	}
}

type scriptedInspector struct {
	snaps []agent.Snapshot
	i     int
}

func (s *scriptedInspector) Snapshot(kinds []agent.Kind) (agent.Snapshot, error) {
	if s.i < len(s.snaps) {
		snap := s.snaps[s.i]
		s.i++
		return snap, nil
	}
	return s.snaps[len(s.snaps)-1], nil
}

func TestRunnerStopsWhenWatchedAgentsFinish(t *testing.T) {
	var out bytes.Buffer
	now := time.Date(2026, 7, 21, 13, 0, 0, 0, time.UTC)
	runner := appRunner{
		cfg: Config{
			Interval:  time.Millisecond,
			Distance:  1,
			Wait:      []agent.Kind{agent.KindClaude},
			WaitStart: time.Minute,
			WaitIdle:  0,
		},
		out:      &out,
		activity: fakeActivity{idleFor: time.Minute, trusted: true},
		mover:    &fakeMover{},
		inspector: &scriptedInspector{snaps: []agent.Snapshot{
			{Working: []agent.Target{{Kind: agent.KindClaude, PID: 42, Detail: "busy"}}},
			{},
		}},
		now: func() time.Time { return now },
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	code, err := runner.run(ctx)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if !strings.Contains(out.String(), "waiting on claude pid 42 (busy)") {
		t.Fatalf("log = %q, want working agent log", out.String())
	}
	if !strings.Contains(out.String(), "agents finished - stopping") {
		t.Fatalf("log = %q, want finished log", out.String())
	}
}

func TestRunnerStopsWhenNoWorkingAgentAppears(t *testing.T) {
	var out bytes.Buffer
	now := time.Date(2026, 7, 21, 13, 0, 0, 0, time.UTC)
	runner := appRunner{
		cfg: Config{
			Interval:  time.Minute,
			Distance:  1,
			Wait:      []agent.Kind{agent.KindClaude},
			WaitStart: 0,
			WaitIdle:  0,
		},
		out:       &out,
		activity:  fakeActivity{idleFor: time.Minute, trusted: true},
		mover:     &fakeMover{},
		inspector: &scriptedInspector{snaps: []agent.Snapshot{{}}},
		now:       func() time.Time { return now },
	}

	code, err := runner.run(context.Background())
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if !strings.Contains(out.String(), "no working agent found - stopping") {
		t.Fatalf("log = %q, want no-agent log", out.String())
	}
}

func TestRunnerRunRejectsMissingAccessibilityPermission(t *testing.T) {
	runner := appRunner{
		cfg:      Config{Interval: time.Minute, Distance: 1},
		out:      &bytes.Buffer{},
		activity: fakeActivity{trusted: false},
		mover:    &fakeMover{},
		now:      func() time.Time { return time.Date(2026, 7, 21, 13, 0, 0, 0, time.UTC) },
	}

	code, err := runner.run(context.Background())
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if err == nil || !strings.Contains(err.Error(), "Accessibility permission") {
		t.Fatalf("error = %v, want Accessibility permission error", err)
	}
}
