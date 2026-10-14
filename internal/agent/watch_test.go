package agent

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestInspectClaudeBusy(t *testing.T) {
	home := t.TempDir()
	write(t, filepath.Join(home, ".claude", "sessions", "42.json"), `{"pid":42,"status":"busy"}`)

	snap, err := Inspect(Source{Home: home, Alive: aliveSet(42)}, []Kind{KindClaude})
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	if !snap.AnyWorking() {
		t.Fatal("expected claude to be working")
	}
	if snap.Working[0].PID != 42 || snap.Working[0].Detail != "busy" {
		t.Fatalf("working = %+v", snap.Working[0])
	}
}

func TestInspectClaudeIdleIsNotWorking(t *testing.T) {
	home := t.TempDir()
	write(t, filepath.Join(home, ".claude", "sessions", "42.json"), `{"pid":42,"status":"idle"}`)

	snap, err := Inspect(Source{Home: home, Alive: aliveSet(42)}, []Kind{KindClaude})
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	if snap.AnyWorking() {
		t.Fatalf("idle claude reported working: %+v", snap.Working)
	}
}

func TestInspectClaudeIgnoresDeadPID(t *testing.T) {
	home := t.TempDir()
	write(t, filepath.Join(home, ".claude", "sessions", "42.json"), `{"pid":42,"status":"busy"}`)

	snap, err := Inspect(Source{Home: home, Alive: aliveSet()}, []Kind{KindClaude})
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	if snap.AnyWorking() {
		t.Fatalf("dead claude pid reported working: %+v", snap.Working)
	}
}

func TestInspectGrokWorkingFromEvents(t *testing.T) {
	home := t.TempDir()
	write(t, filepath.Join(home, ".grok", "active_sessions.json"), `[{"session_id":"s1","pid":7}]`)
	write(t, filepath.Join(home, ".grok", "sessions", "cwd", "s1", "events.jsonl"), `{"ts":"t0","type":"turn_started"}
{"ts":"t1","type":"phase_changed","phase":"tool_execution"}
`)

	snap, err := Inspect(Source{Home: home, Alive: aliveSet(7)}, []Kind{KindGrok})
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	if !snap.AnyWorking() {
		t.Fatal("expected grok to be working")
	}
	if snap.Working[0].Detail != "tool_execution" {
		t.Fatalf("detail = %q, want tool_execution", snap.Working[0].Detail)
	}
}

func TestInspectGrokTurnEndedIsIdle(t *testing.T) {
	home := t.TempDir()
	write(t, filepath.Join(home, ".grok", "active_sessions.json"), `[{"session_id":"s1","pid":7}]`)
	write(t, filepath.Join(home, ".grok", "sessions", "cwd", "s1", "events.jsonl"), `{"type":"phase_changed","phase":"streaming_text"}
{"type":"turn_ended"}
`)

	snap, err := Inspect(Source{Home: home, Alive: aliveSet(7)}, []Kind{KindGrok})
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	if snap.AnyWorking() {
		t.Fatalf("idle grok reported working: %+v", snap.Working)
	}
}

func TestInspectCodexTaskStartedIsWorking(t *testing.T) {
	home := t.TempDir()
	write(t, filepath.Join(home, ".codex", "sessions", "2026", "10", "14", "rollout-1.jsonl"), `{"type":"event_msg","payload":{"type":"task_started"}}
{"type":"event_msg","payload":{"type":"agent_message"}}
`)

	snap, err := Inspect(Source{Home: home, Now: time.Now}, []Kind{KindCodex})
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	if !snap.AnyWorking() {
		t.Fatal("expected codex to be working")
	}
}

func TestInspectCodexTaskCompleteIsIdle(t *testing.T) {
	home := t.TempDir()
	write(t, filepath.Join(home, ".codex", "sessions", "2026", "10", "14", "rollout-1.jsonl"), `{"type":"event_msg","payload":{"type":"task_started"}}
{"type":"event_msg","payload":{"type":"task_complete"}}
{"type":"event_msg","payload":{"type":"token_count"}}
`)

	snap, err := Inspect(Source{Home: home, Now: time.Now}, []Kind{KindCodex})
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	if snap.AnyWorking() {
		t.Fatalf("completed codex reported working: %+v", snap.Working)
	}
}

func TestInspectClaudeShellStatusIsWorking(t *testing.T) {
	home := t.TempDir()
	write(t, filepath.Join(home, ".claude", "sessions", "42.json"), `{"pid":42,"status":"shell"}`)

	snap, err := Inspect(Source{Home: home, Alive: aliveSet(42)}, []Kind{KindClaude})
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	if !snap.AnyWorking() {
		t.Fatal("expected claude shell status to be working")
	}
}

func TestInspectCodexIgnoresNewerCompletedSession(t *testing.T) {
	home := t.TempDir()
	write(t, filepath.Join(home, ".codex", "sessions", "2026", "10", "14", "rollout-old.jsonl"), `{"type":"event_msg","payload":{"type":"task_started"}}`)
	write(t, filepath.Join(home, ".codex", "sessions", "2026", "10", "14", "rollout-new.jsonl"), `{"type":"event_msg","payload":{"type":"task_complete"}}`)

	snap, err := Inspect(Source{Home: home, Now: time.Now}, []Kind{KindCodex})
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	if !snap.AnyWorking() {
		t.Fatal("expected in-progress codex session to stay working")
	}
}

func TestInspectCodexStaleTaskStartedIsIdle(t *testing.T) {
	home := t.TempDir()
	write(t, filepath.Join(home, ".codex", "sessions", "2026", "10", "14", "rollout-1.jsonl"), `{"type":"event_msg","payload":{"type":"task_started"}}`)
	now := time.Now().Add(30 * time.Minute)

	snap, err := Inspect(Source{Home: home, Now: func() time.Time { return now }}, []Kind{KindCodex})
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	if snap.AnyWorking() {
		t.Fatalf("stale codex task reported working: %+v", snap.Working)
	}
}

func TestInspectEmptyHomeIsIdle(t *testing.T) {
	snap, err := Inspect(Source{Home: t.TempDir()}, AllKinds())
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	if snap.AnyWorking() {
		t.Fatalf("empty home reported working: %+v", snap.Working)
	}
}

func TestInspectOnlyWatchesRequestedKinds(t *testing.T) {
	home := t.TempDir()
	write(t, filepath.Join(home, ".claude", "sessions", "42.json"), `{"pid":42,"status":"busy"}`)

	snap, err := Inspect(Source{Home: home, Alive: aliveSet(42)}, []Kind{KindGrok})
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	if snap.AnyWorking() {
		t.Fatalf("unwatched claude reported working: %+v", snap.Working)
	}
}

func write(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func aliveSet(pids ...int) func(int) bool {
	set := make(map[int]bool, len(pids))
	for _, pid := range pids {
		set[pid] = true
	}
	return func(pid int) bool { return set[pid] }
}
