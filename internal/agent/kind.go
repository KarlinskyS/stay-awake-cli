package agent

import "strings"

type Kind string

const (
	KindClaude Kind = "claude"
	KindGrok   Kind = "grok"
	KindCodex  Kind = "codex"
)

func AllKinds() []Kind {
	return []Kind{KindClaude, KindGrok, KindCodex}
}

func JoinKinds(kinds []Kind) string {
	parts := make([]string, len(kinds))
	for i, kind := range kinds {
		parts[i] = string(kind)
	}
	return strings.Join(parts, ", ")
}
