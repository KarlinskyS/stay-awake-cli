package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/ieffai/stay-awake-cli/internal/agent"
)

const (
	DefaultInterval  = 60 * time.Second
	DefaultDistance  = 1
	DefaultWaitStart = 10 * time.Minute
	DefaultWaitIdle  = 60 * time.Second
)

type Config struct {
	Interval  time.Duration
	Distance  int
	Always    bool
	Verbose   bool
	Version   bool
	Wait      []agent.Kind
	WaitStart time.Duration
	WaitIdle  time.Duration
}

func DefaultConfig() Config {
	return Config{
		Interval:  DefaultInterval,
		Distance:  DefaultDistance,
		WaitStart: DefaultWaitStart,
		WaitIdle:  DefaultWaitIdle,
	}
}

func ParseArgs(args []string, out io.Writer, version string) (Config, error) {
	cfg := DefaultConfig()
	fs := flag.NewFlagSet("stay", flag.ContinueOnError)
	fs.SetOutput(out)

	interval := fs.String("interval", cfg.Interval.String(), "check interval, for example 60s, 1m, or 500ms")
	fs.IntVar(&cfg.Distance, "distance", cfg.Distance, "cursor movement distance in display points")
	fs.BoolVar(&cfg.Always, "always", cfg.Always, "move cursor every interval regardless of user activity")
	fs.BoolVar(&cfg.Verbose, "verbose", cfg.Verbose, "print detailed diagnostic logs")
	fs.BoolVar(&cfg.Version, "version", false, "print version and exit")
	waitAgent := fs.Bool("wait-agent", false, "stop when Claude, Grok, and Codex finish their current work")
	waitClaude := fs.Bool("wait-claude", false, "stop when Claude Code finishes its current work")
	waitGrok := fs.Bool("wait-grok", false, "stop when Grok finishes its current work")
	waitCodex := fs.Bool("wait-codex", false, "stop when Codex finishes its current work")
	waitStart := fs.String("wait-start", cfg.WaitStart.String(), "how long to wait for a working agent to appear")
	waitIdle := fs.String("wait-idle", cfg.WaitIdle.String(), "how long agents must stay idle before stay exits")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: stay [options]\n\nOptions:\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return cfg, err
	}
	if fs.NArg() > 0 {
		return cfg, fmt.Errorf("unexpected argument: %s", fs.Arg(0))
	}

	parsedInterval, err := ParseInterval(*interval)
	if err != nil {
		return cfg, err
	}
	cfg.Interval = parsedInterval
	if cfg.Distance < 1 {
		return cfg, errors.New("--distance must be greater than 0")
	}

	cfg.Wait = collectWaitKinds(*waitAgent, *waitClaude, *waitGrok, *waitCodex)
	cfg.WaitStart, err = parseNonNegativeDuration("wait-start", *waitStart)
	if err != nil {
		return cfg, err
	}
	cfg.WaitIdle, err = parseNonNegativeDuration("wait-idle", *waitIdle)
	if err != nil {
		return cfg, err
	}

	return cfg, nil
}

func collectWaitKinds(waitAgent, waitClaude, waitGrok, waitCodex bool) []agent.Kind {
	if waitAgent {
		return agent.AllKinds()
	}
	var kinds []agent.Kind
	if waitClaude {
		kinds = append(kinds, agent.KindClaude)
	}
	if waitGrok {
		kinds = append(kinds, agent.KindGrok)
	}
	if waitCodex {
		kinds = append(kinds, agent.KindCodex)
	}
	return kinds
}

func parseNonNegativeDuration(flagName string, value string) (time.Duration, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, fmt.Errorf("--%s must not be empty", flagName)
	}
	if seconds, err := strconv.Atoi(value); err == nil {
		if seconds < 0 {
			return 0, fmt.Errorf("--%s must not be negative", flagName)
		}
		return time.Duration(seconds) * time.Second, nil
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("invalid --%s %q: use values like 60s, 1m, or 0s", flagName, value)
	}
	if duration < 0 {
		return 0, fmt.Errorf("--%s must not be negative", flagName)
	}
	return duration, nil
}

func ParseInterval(value string) (time.Duration, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, errors.New("--interval must not be empty")
	}

	if seconds, err := strconv.Atoi(value); err == nil {
		if seconds <= 0 {
			return 0, errors.New("--interval must be greater than 0")
		}
		return time.Duration(seconds) * time.Second, nil
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("invalid --interval %q: use values like 60s, 1m, or 500ms", value)
	}
	if duration <= 0 {
		return 0, errors.New("--interval must be greater than 0")
	}
	return duration, nil
}
