package fake

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"

	port "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	ToolDirective  = "TOOL:"
	FinalDirective = "FAKE:"
	FailDirective  = "FAIL:"

	callPrefix     = "fake-call-"
	toolCallTokens = 8
	emptyArguments = "{}"
	ranPrefix      = "ran "
	failedSuffix   = " (error)"
	errorField     = "error"
)

var toolPattern = regexp.MustCompile(`TOOL:([A-Za-z0-9_.-]+)(\{.*\})`)

type Turn struct {
	Text string
	Tool string
	Args json.RawMessage
}

type Script func(prompt string) Turn

type scripted struct {
	name string
	args string
}

type plan struct {
	calls   []scripted
	answer  string
	failure errors.Code
}

func planOf(prompt string, written Script) plan {
	var planned plan
	for _, match := range toolPattern.FindAllStringSubmatch(prompt, -1) {
		planned.calls = append(planned.calls, scripted{name: match[1], args: match[2]})
	}
	for line := range strings.SplitSeq(prompt, "\n") {
		trimmed := strings.TrimSpace(line)
		if answer, found := strings.CutPrefix(trimmed, FinalDirective); found {
			planned.answer = strings.TrimSpace(answer)
		}
		if planned.failure == "" {
			planned.failure = failureIn(trimmed)
		}
	}
	if len(planned.calls) > 0 || planned.answer != "" || written == nil {
		return planned
	}

	turn := written(prompt)
	if turn.Tool != "" {
		planned.calls = append(planned.calls, scripted{name: turn.Tool, args: string(turn.Args)})
	}
	planned.answer = turn.Text
	return planned
}

func failureIn(line string) errors.Code {
	for _, directive := range []string{ErrorDirective, FailDirective} {
		if code, found := strings.CutPrefix(line, directive); found {
			return errors.Code(strings.TrimSpace(code))
		}
	}
	return ""
}

func (c *Client) converse(req port.Request) (port.Response, error) {
	asked := lastAsked(req.Messages)
	prompt := ""
	if asked >= 0 {
		prompt = req.Messages[asked].Text
	}
	since := req.Messages[asked+1:]
	answered := results(since)

	planned := planOf(prompt, c.script)
	if planned.failure != "" && len(answered) == 0 {
		return port.Response{}, errors.New(planned.failure, "the scripted model failed on purpose").
			WithDetail("model", req.Ref.String())
	}

	if len(answered) < len(planned.calls) {
		next := planned.calls[len(answered)]
		call := port.ToolCall{
			ID:   callPrefix + strconv.Itoa(turnOf(req.Messages, asked)) + "-" + strconv.Itoa(len(answered)+1),
			Name: next.name,
			Args: arguments(next.args),
		}
		return port.Response{
			Calls: []port.ToolCall{call}, Usage: usageOf(req, toolCallTokens),
			FinishReason: port.FinishStop, Tier: served(req.Tier),
		}, nil
	}

	text := planned.final(since, answered)
	return port.Response{Text: text, Usage: usageOf(req, tokens(text)), FinishReason: port.FinishStop, Tier: served(req.Tier)}, nil
}

func lastAsked(messages []port.Message) int {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == port.RoleUser {
			return i
		}
	}
	return -1
}

func turnOf(messages []port.Message, asked int) int {
	turns := 0
	for i := 0; i <= asked; i++ {
		if messages[i].Role == port.RoleUser {
			turns++
		}
	}
	return turns
}

func results(messages []port.Message) []port.ToolResult {
	var found []port.ToolResult
	for _, message := range messages {
		if message.Result != nil {
			found = append(found, *message.Result)
		}
	}
	return found
}

func arguments(raw string) json.RawMessage {
	if !json.Valid([]byte(raw)) {
		return json.RawMessage(emptyArguments)
	}
	return json.RawMessage(raw)
}

func (p plan) final(since []port.Message, answered []port.ToolResult) string {
	if p.answer != "" {
		return p.answer
	}
	if len(answered) == 0 {
		return defaultAnswer
	}

	called := make(map[string]string, len(answered))
	for _, message := range since {
		if message.Call != nil {
			called[message.Call.ID] = message.Call.Name
		}
	}

	ran := make([]string, 0, len(answered))
	for _, result := range answered {
		named := called[result.CallID]
		if failed(result.Output) {
			named += failedSuffix
		}
		ran = append(ran, named)
	}
	return ranPrefix + strings.Join(ran, ", ")
}

func failed(output json.RawMessage) bool {
	var object map[string]json.RawMessage
	if json.Unmarshal(output, &object) != nil {
		return false
	}
	_, carries := object[errorField]
	return carries
}
