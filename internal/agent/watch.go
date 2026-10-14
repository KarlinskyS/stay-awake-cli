package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const codexFreshWindow = 20 * time.Minute

type Target struct {
	Kind   Kind
	PID    int
	Detail string
}

type Snapshot struct {
	Working []Target
}

func (s Snapshot) AnyWorking() bool {
	return len(s.Working) > 0
}

func (s Snapshot) Signature() string {
	parts := make([]string, 0, len(s.Working))
	for _, target := range s.Working {
		parts = append(parts, target.String())
	}
	return strings.Join(parts, "; ")
}

func (t Target) String() string {
	if t.PID > 0 {
		return string(t.Kind) + " pid " + itoa(t.PID) + " (" + t.Detail + ")"
	}
	return string(t.Kind) + " (" + t.Detail + ")"
}

type Source struct {
	Home  string
	Now   func() time.Time
	Alive func(int) bool
}

func Inspect(src Source, kinds []Kind) (Snapshot, error) {
	var snap Snapshot
	now := time.Now()
	if src.Now != nil {
		now = src.Now()
	}
	alive := src.Alive
	if alive == nil {
		alive = pidAlive
	}

	for _, kind := range kinds {
		var (
			targets []Target
			err     error
		)
		switch kind {
		case KindClaude:
			targets, err = inspectClaude(src.Home, alive)
		case KindGrok:
			targets, err = inspectGrok(src.Home, alive)
		case KindCodex:
			targets, err = inspectCodex(src.Home, now)
		}
		if err != nil {
			return Snapshot{}, err
		}
		snap.Working = append(snap.Working, targets...)
	}
	return snap, nil
}

type claudeSession struct {
	PID    int    `json:"pid"`
	Status string `json:"status"`
}

func inspectClaude(home string, alive func(int) bool) ([]Target, error) {
	matches, err := filepath.Glob(filepath.Join(home, ".claude", "sessions", "*.json"))
	if err != nil {
		return nil, err
	}
	var working []Target
	for _, path := range matches {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var session claudeSession
		if err := json.Unmarshal(data, &session); err != nil {
			continue
		}
		if session.PID <= 0 || !alive(session.PID) {
			continue
		}
		status := strings.TrimSpace(session.Status)
		if status == "" {
			status = "unknown"
		}
		// Claude is idle only at the prompt. busy, shell, waiting, and other
		// statuses mean a turn is still in flight.
		if strings.EqualFold(status, "idle") {
			continue
		}
		working = append(working, Target{Kind: KindClaude, PID: session.PID, Detail: status})
	}
	return working, nil
}

type grokActiveSession struct {
	SessionID string `json:"session_id"`
	PID       int    `json:"pid"`
}

func inspectGrok(home string, alive func(int) bool) ([]Target, error) {
	data, err := os.ReadFile(filepath.Join(home, ".grok", "active_sessions.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var sessions []grokActiveSession
	if err := json.Unmarshal(data, &sessions); err != nil {
		return nil, nil
	}

	var working []Target
	for _, session := range sessions {
		if session.PID <= 0 || !alive(session.PID) {
			continue
		}
		eventPath := findGrokEvents(home, session.SessionID)
		detail := "active"
		if eventPath != "" {
			event, ok, err := lastJSONLEvent(eventPath)
			if err != nil {
				return nil, err
			}
			if ok {
				if event.Type == "turn_ended" {
					continue
				}
				if event.Type == "phase_changed" && event.Phase != "" {
					detail = event.Phase
				} else if event.Type != "" {
					detail = event.Type
				}
			}
		}
		working = append(working, Target{Kind: KindGrok, PID: session.PID, Detail: detail})
	}
	return working, nil
}

func findGrokEvents(home string, sessionID string) string {
	if sessionID == "" {
		return ""
	}
	matches, err := filepath.Glob(filepath.Join(home, ".grok", "sessions", "*", sessionID, "events.jsonl"))
	if err != nil || len(matches) == 0 {
		return ""
	}
	return matches[0]
}

type jsonlEvent struct {
	Type    string `json:"type"`
	Phase   string `json:"phase"`
	Payload struct {
		Type string `json:"type"`
	} `json:"payload"`
}

func inspectCodex(home string, now time.Time) ([]Target, error) {
	root := filepath.Join(home, ".codex", "sessions")
	var working []Target
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if info.IsDir() || !strings.HasPrefix(info.Name(), "rollout-") || !strings.HasSuffix(info.Name(), ".jsonl") {
			return nil
		}
		if now.Sub(info.ModTime()) > codexFreshWindow {
			return nil
		}
		boundary, ok, err := lastCodexTurnBoundary(path)
		if err != nil {
			return err
		}
		if ok && boundary == "task_started" {
			working = append(working, Target{Kind: KindCodex, Detail: boundary})
		}
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return working, nil
}

func lastCodexTurnBoundary(path string) (string, bool, error) {
	events, err := tailJSONLEvents(path)
	if err != nil {
		return "", false, err
	}
	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]
		if event.Type != "event_msg" {
			continue
		}
		switch event.Payload.Type {
		case "task_started", "task_complete":
			return event.Payload.Type, true, nil
		}
	}
	return "", false, nil
}

func lastJSONLEvent(path string) (jsonlEvent, bool, error) {
	events, err := tailJSONLEvents(path)
	if err != nil {
		return jsonlEvent{}, false, err
	}
	if len(events) == 0 {
		return jsonlEvent{}, false, nil
	}
	return events[len(events)-1], true, nil
}

func tailJSONLEvents(path string) ([]jsonlEvent, error) {
	data, err := readTail(path, 256*1024)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	text := string(data)
	if len(data) == 256*1024 {
		if i := strings.IndexByte(text, '\n'); i >= 0 {
			text = text[i+1:]
		}
	}
	var events []jsonlEvent
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var event jsonlEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}
		events = append(events, event)
	}
	return events, nil
}

func readTail(path string, maxBytes int64) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	size := info.Size()
	start := size - maxBytes
	if start < 0 {
		start = 0
	}
	buf := make([]byte, size-start)
	_, err = file.ReadAt(buf, start)
	return buf, err
}

func pidAlive(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
