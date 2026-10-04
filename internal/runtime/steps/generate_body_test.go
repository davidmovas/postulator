package steps_test

import (
	"context"
	"strings"
	"sync"
	"testing"

	port "github.com/davidmovas/postulator/internal/application/llm"
	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

type answer string

const (
	answerDraft     answer = "draft"
	answerTruncated answer = "truncated"
	answerThrottled answer = "throttled"
	answerProse     answer = "prose"
)

type scriptedWriter struct {
	mu       sync.Mutex
	script   []answer
	ceilings []int
}

func (w *scriptedWriter) Complete(_ context.Context, req port.Request) (port.Response, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	next := answerDraft
	if len(w.ceilings) < len(w.script) {
		next = w.script[len(w.ceilings)]
	}
	w.ceilings = append(w.ceilings, req.MaxTokens)

	used := domainllm.Usage{Input: 100, Output: req.MaxTokens, Total: 100 + req.MaxTokens}
	switch next {
	case answerTruncated:
		return port.Response{FinishReason: port.FinishLength, Usage: used}, nil
	case answerThrottled:
		return port.Response{}, errors.New(errors.RateLimited, "slow down")
	case answerProse:
		return port.Response{Text: "sure thing", FinishReason: port.FinishStop, Usage: used}, nil
	default:
		return port.Response{Text: goodDraft, FinishReason: port.FinishStop, Usage: domainllm.Usage{Input: 100, Output: 50, Total: 150}}, nil
	}
}

func (w *scriptedWriter) Stream(context.Context, port.Request) (<-chan port.Delta, error) {
	return nil, errors.New(errors.Internal, "the scripted writer does not stream")
}

func (w *scriptedWriter) asked() []int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]int(nil), w.ceilings...)
}

func chain(t *testing.T, def run.StepDef, sc *run.StepContext) (run.Result, int, error) {
	t.Helper()

	for attempt := 0; ; attempt++ {
		sc.Item.Attempts = attempt
		result, err := def.Run(t.Context(), sc)
		if err == nil || run.Classify(err).Action != run.ActionRetry || attempt+1 >= def.Retry.Max {
			return result, attempt + 1, err
		}
	}
}

func TestTheWriterIsPaidForAtMostTwoTruncatedAnswers(t *testing.T) {
	t.Parallel()

	long := spec()
	long.Sections = []template.Section{{Heading: "Overview", TargetWords: 1800, Required: true}}
	room := 1800*3 + 1024

	cases := []struct {
		name     string
		script   []answer
		ceilings []int
		paused   bool
		failed   errors.Code
	}{
		{name: "a clean answer is written at once", ceilings: []int{room}},
		{
			name:     "one truncation is tried again once with double the room",
			script:   []answer{answerTruncated},
			ceilings: []int{room, 2 * room},
		},
		{
			name:     "a second truncation pauses the page for a person",
			script:   []answer{answerTruncated, answerTruncated},
			ceilings: []int{room, 2 * room},
			paused:   true,
		},
		{
			name:     "a throttled call keeps the step's retry",
			script:   []answer{answerThrottled},
			ceilings: []int{room, 2 * room},
		},
		{
			name:     "a truncation and a throttled call still leave room for the answer",
			script:   []answer{answerTruncated, answerThrottled},
			ceilings: []int{room, 2 * room, 2 * room},
		},
		{
			name:     "a truncation at double the room pauses however the chain got there",
			script:   []answer{answerThrottled, answerTruncated},
			ceilings: []int{room, 2 * room},
			paused:   true,
		},
		{
			name:     "throttled every time fails the step once its retries run out",
			script:   []answer{answerThrottled, answerThrottled, answerThrottled},
			ceilings: []int{room, 2 * room, 2 * room},
			failed:   errors.RateLimited,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			deps := unitDeps()
			sc := unitContext(t, map[run.ArtifactKind][]byte{run.ArtifactLinkContext: linkContextBlob(t, deps)})
			sc.Spec = long
			writer := &scriptedWriter{script: tc.script}
			deps.LLM = writer

			result, _, err := chain(t, steps.GenerateBody(deps), sc)

			asked := writer.asked()
			if len(asked) != len(tc.ceilings) {
				t.Fatalf("the writer was asked %d times with %v, want %v", len(asked), asked, tc.ceilings)
			}
			for i := range asked {
				if asked[i] != tc.ceilings[i] {
					t.Fatalf("the writer was given %v tokens of room, want %v", asked, tc.ceilings)
				}
			}

			switch {
			case tc.failed != "":
				if !errors.IsCode(err, tc.failed) {
					t.Fatalf("the chain ended on %v, want %s", err, tc.failed)
				}
			case tc.paused:
				if err != nil {
					t.Fatalf("the chain ended on %v, want a pause", err)
				}
				if result.Next != run.TransitionPause || result.Reason != run.PauseNeedsHuman {
					t.Fatalf("result = %+v, want the page paused for a person", result)
				}
				if !strings.Contains(result.Message, steps.ReasonWriterOutOfRoom) || !strings.Contains(result.Message, sc.Page.Path) {
					t.Fatalf("message = %q, want the page and what to change", result.Message)
				}
				if len(result.Artifacts) != 0 || result.Tokens != 100+2*room {
					t.Fatalf("result = %+v, want no artifact and the tokens the last truncated answer spent", result)
				}
			default:
				if err != nil || result.Next == run.TransitionPause || len(result.Artifacts) != 2 {
					t.Fatalf("result = %+v, %v; want the body written", result, err)
				}
			}
		})
	}
}

func TestAMalformedAnswerKeepsTheRetryAndNeverPauses(t *testing.T) {
	t.Parallel()

	deps := unitDeps()
	sc := unitContext(t, map[run.ArtifactKind][]byte{run.ArtifactLinkContext: linkContextBlob(t, deps)})
	deps.LLM = &scriptedWriter{script: []answer{answerProse, answerProse, answerProse, answerProse, answerProse, answerProse}}

	result, attempts, err := chain(t, steps.GenerateBody(deps), sc)
	if !errors.IsCode(err, errors.External) || result.Next == run.TransitionPause {
		t.Fatalf("the chain ended on %+v, %v; want the malformed answer handed to the engine's retry", result, err)
	}
	if attempts != steps.GenerateBody(deps).Retry.Max {
		t.Fatalf("the chain stopped after %d attempts, want every retry the step declares", attempts)
	}
}
