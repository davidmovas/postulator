package agent

import (
	"strings"

	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
)

type tally struct {
	used  domainllm.Usage
	usd   float64
	seq   int64
	calls int
}

func (t *tally) next() int64 {
	t.seq++
	return t.seq
}

func (t *tally) add(used domainllm.Usage, usd float64) {
	t.used = t.used.Add(used)
	t.usd += usd
	t.calls++
}

func (t *tally) answered() int {
	return t.calls
}

func (t *tally) spent() float64 {
	return t.usd
}

func (t *tally) total() domainllm.Usage {
	return t.used
}

type answer struct {
	rounds []string
}

func (a *answer) spoke(text string) {
	if strings.TrimSpace(text) == "" {
		return
	}
	a.rounds = append(a.rounds, text)
}

func (a *answer) String() string {
	return strings.Join(a.rounds, "\n\n")
}
