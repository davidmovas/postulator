package tools

import (
	"strings"

	"github.com/davidmovas/postulator/internal/application/pagetypes"
)

func rewrittenFields(kind pagetypes.Type) []string {
	out := make([]string, 0, len(kind.Scopes))
	for i := range kind.Scopes {
		if kind.Scopes[i].Rewrites && kind.Scopes[i].Enabled && kind.Scopes[i].AIWrites {
			out = append(out, kind.Scopes[i].Field)
		}
	}
	return out
}

func whoseScopeRewrites(kind pagetypes.Type, fields []string) string {
	return kind.Name + ", whose scope rewrites its " + strings.Join(fields, ", ")
}
