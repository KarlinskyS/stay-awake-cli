package cli

import (
	"io"
	"reflect"
	"testing"
	"time"

	"github.com/ieffai/stay-awake-cli/internal/agent"
)

func TestParseArgsDefaults(t *testing.T) {
	cfg, err := ParseArgs(nil, io.Discard, "test")
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	if cfg.Interval != 60*time.Second {
		t.Fatalf("Interval = %s, want 60s", cfg.Interval)
	}
	if cfg.Distance != 1 {
		t.Fatalf("Distance = %d, want 1", cfg.Distance)
	}
	if cfg.Always {
		t.Fatal("Always = true, want false")
	}
	if cfg.Verbose {
		t.Fatal("Verbose = true, want false")
	}
	if len(cfg.Wait) != 0 {
		t.Fatalf("Wait = %v, want empty", cfg.Wait)
	}
	if cfg.WaitStart != DefaultWaitStart {
		t.Fatalf("WaitStart = %s, want %s", cfg.WaitStart, DefaultWaitStart)
	}
	if cfg.WaitIdle != DefaultWaitIdle {
		t.Fatalf("WaitIdle = %s, want %s", cfg.WaitIdle, DefaultWaitIdle)
	}
}

func TestParseArgsOptions(t *testing.T) {
	cfg, err := ParseArgs([]string{"--interval", "2m", "--distance", "3", "--always", "--verbose"}, io.Discard, "test")
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	if cfg.Interval != 2*time.Minute {
		t.Fatalf("Interval = %s, want 2m", cfg.Interval)
	}
	if cfg.Distance != 3 {
		t.Fatalf("Distance = %d, want 3", cfg.Distance)
	}
	if !cfg.Always {
		t.Fatal("Always = false, want true")
	}
	if !cfg.Verbose {
		t.Fatal("Verbose = false, want true")
	}
}

func TestParseIntervalBareSeconds(t *testing.T) {
	got, err := ParseInterval("60")
	if err != nil {
		t.Fatalf("ParseInterval() error = %v", err)
	}
	if got != time.Minute {
		t.Fatalf("ParseInterval() = %s, want 1m", got)
	}
}

func TestParseArgsWaitFlags(t *testing.T) {
	cases := []struct {
		args []string
		want []agent.Kind
	}{
		{args: []string{"--wait-agent"}, want: agent.AllKinds()},
		{args: []string{"--wait-claude"}, want: []agent.Kind{agent.KindClaude}},
		{args: []string{"--wait-codex"}, want: []agent.Kind{agent.KindCodex}},
		{args: []string{"--wait-grok"}, want: []agent.Kind{agent.KindGrok}},
		{args: []string{"--wait-claude", "--wait-grok"}, want: []agent.Kind{agent.KindClaude, agent.KindGrok}},
		{args: []string{"--wait-agent", "--wait-claude"}, want: agent.AllKinds()},
	}
	for _, tc := range cases {
		cfg, err := ParseArgs(tc.args, io.Discard, "test")
		if err != nil {
			t.Fatalf("ParseArgs(%v) error = %v", tc.args, err)
		}
		if !reflect.DeepEqual(cfg.Wait, tc.want) {
			t.Fatalf("ParseArgs(%v) Wait = %v, want %v", tc.args, cfg.Wait, tc.want)
		}
	}
}

func TestParseArgsWaitDurations(t *testing.T) {
	cfg, err := ParseArgs([]string{"--wait-claude", "--wait-start", "2m", "--wait-idle", "15s"}, io.Discard, "test")
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}
	if cfg.WaitStart != 2*time.Minute {
		t.Fatalf("WaitStart = %s, want 2m", cfg.WaitStart)
	}
	if cfg.WaitIdle != 15*time.Second {
		t.Fatalf("WaitIdle = %s, want 15s", cfg.WaitIdle)
	}
}

func TestParseArgsAllowsZeroWaitDurations(t *testing.T) {
	cfg, err := ParseArgs([]string{"--wait-agent", "--wait-start", "0s", "--wait-idle", "0"}, io.Discard, "test")
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}
	if cfg.WaitStart != 0 {
		t.Fatalf("WaitStart = %s, want 0", cfg.WaitStart)
	}
	if cfg.WaitIdle != 0 {
		t.Fatalf("WaitIdle = %s, want 0", cfg.WaitIdle)
	}
}

func TestParseArgsRejectsInvalidValues(t *testing.T) {
	cases := [][]string{
		{"--interval", "0s"},
		{"--interval", "soon"},
		{"--distance", "0"},
		{"--wait-start", "-1s"},
		{"--wait-idle", "soon"},
		{"extra"},
	}
	for _, args := range cases {
		if _, err := ParseArgs(args, io.Discard, "test"); err == nil {
			t.Fatalf("ParseArgs(%v) error = nil, want error", args)
		}
	}
}
