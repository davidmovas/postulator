package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	stderrors "errors"
	"io"
	"net/http"

	port "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	maxEventBytes   = 16 << 20
	eventLineSlack  = 64
	readBufferBytes = 64 << 10

	eventDone       = "[DONE]"
	eventCreated    = "response.created"
	eventInProgress = "response.in_progress"
	eventQueued     = "response.queued"
	eventTextDelta  = "response.output_text.delta"
	eventItemDone   = "response.output_item.done"
	eventCompleted  = "response.completed"
	eventIncomplete = "response.incomplete"
	eventFailed     = "response.failed"
	eventError      = "error"
)

var errEventTooLarge = stderrors.New("a server-sent event is larger than the client reads")

type sseEvent struct {
	name string
	data []byte
}

type sseReader struct {
	reader *bufio.Reader
	limit  int
}

func newSSEReader(body io.Reader, limit int) *sseReader {
	return &sseReader{reader: bufio.NewReaderSize(body, readBufferBytes), limit: limit}
}

func (s *sseReader) next() (sseEvent, error) {
	var (
		event sseEvent
		seen  bool
		lines int
	)
	for {
		line, err := s.line()
		if err != nil {
			if stderrors.Is(err, io.EOF) && seen {
				return event, nil
			}
			return sseEvent{}, err
		}
		if len(line) == 0 {
			if seen {
				return event, nil
			}
			continue
		}
		if line[0] == ':' {
			continue
		}

		field, value, _ := bytes.Cut(line, []byte(":"))
		value = bytes.TrimPrefix(value, []byte(" "))
		switch string(field) {
		case "event":
			event.name, seen = string(value), true
		case "data":
			if lines > 0 {
				event.data = append(event.data, '\n')
			}
			event.data = append(event.data, value...)
			if event.data == nil {
				event.data = []byte{}
			}
			lines++
			seen = true
		}
		if len(event.data) > s.limit {
			return sseEvent{}, errEventTooLarge
		}
	}
}

func (s *sseReader) line() ([]byte, error) {
	var line []byte
	for {
		chunk, err := s.reader.ReadSlice('\n')
		line = append(line, chunk...)
		if len(line) > s.limit+eventLineSlack {
			return nil, errEventTooLarge
		}
		switch {
		case err == nil:
			return bytes.TrimRight(line, "\r\n"), nil
		case stderrors.Is(err, bufio.ErrBufferFull):
			continue
		case stderrors.Is(err, io.EOF) && len(line) > 0:
			return bytes.TrimRight(line, "\r\n"), nil
		default:
			return nil, err
		}
	}
}

type streamState struct {
	delivered map[string]bool
	sentTier  string
	spoke     bool
}

type outcome struct {
	err    error
	deltas []port.Delta
	ended  bool
}

func newStreamState(sentTier string) *streamState {
	return &streamState{delivered: map[string]bool{}, sentTier: sentTier}
}

func admission(event sseEvent) bool {
	switch event.name {
	case eventCreated, eventInProgress, eventQueued:
		return true
	default:
		return bytes.Equal(bytes.TrimSpace(event.data), []byte(eventDone))
	}
}

func (s *streamState) read(event sseEvent) outcome {
	if bytes.Equal(bytes.TrimSpace(event.data), []byte(eventDone)) {
		return outcome{}
	}

	var head struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(event.data, &head); err != nil {
		return outcome{err: unreadableEvent(err), ended: true}
	}
	kind := head.Type
	if kind == "" {
		kind = event.name
	}

	switch kind {
	case eventTextDelta:
		return s.text(event.data)
	case eventItemDone:
		return s.item(event.data)
	case eventCompleted, eventIncomplete:
		return s.finish(event.data, kind)
	case eventFailed:
		return s.failed(event.data)
	case eventError:
		return s.fault(event.data)
	default:
		return outcome{}
	}
}

func (s *streamState) text(data []byte) outcome {
	var delta struct {
		Delta string `json:"delta"`
	}
	if err := json.Unmarshal(data, &delta); err != nil {
		return outcome{err: unreadableEvent(err), ended: true}
	}
	if delta.Delta == "" {
		return outcome{}
	}
	s.spoke = true
	return outcome{deltas: []port.Delta{{Text: delta.Delta}}}
}

func (s *streamState) item(data []byte) outcome {
	var done struct {
		Item wireOutput `json:"item"`
	}
	if err := json.Unmarshal(data, &done); err != nil {
		return outcome{err: unreadableEvent(err), ended: true}
	}
	if done.Item.Type != itemFunctionCall || s.delivered[done.Item.CallID] {
		return outcome{}
	}

	call, err := callOf(done.Item)
	if err != nil {
		return outcome{err: err, ended: true}
	}
	s.delivered[call.ID] = true
	return outcome{deltas: []port.Delta{{Call: &call}}}
}

func (s *streamState) finish(data []byte, kind string) outcome {
	var final struct {
		Response wireResponse `json:"response"`
	}
	if err := json.Unmarshal(data, &final); err != nil {
		return outcome{err: unreadableEvent(err), ended: true}
	}
	answer := final.Response
	if answer.Status == "" && kind == eventIncomplete {
		answer.Status = statusIncomplete
	}

	calls, err := callsOf(answer.Output)
	if err != nil {
		return outcome{err: err, ended: true}
	}

	var deltas []port.Delta
	if text := textOf(answer.Output); text != "" && !s.spoke {
		deltas = append(deltas, port.Delta{Text: text})
	}
	for i := range calls {
		if !s.delivered[calls[i].ID] {
			s.delivered[calls[i].ID] = true
			deltas = append(deltas, port.Delta{Call: &calls[i]})
		}
	}
	usage := usageOf(answer.Usage)
	deltas = append(deltas, port.Delta{
		Done:   true,
		Usage:  &usage,
		Finish: finishOf(answer),
		Tier:   tierOf(answer.ServiceTier, s.sentTier),
	})
	return outcome{deltas: deltas, ended: true}
}

func (s *streamState) failed(data []byte) outcome {
	var final struct {
		Response wireResponse `json:"response"`
	}
	if err := json.Unmarshal(data, &final); err != nil {
		return outcome{err: unreadableEvent(err), ended: true}
	}
	return outcome{err: failedOf(final.Response), ended: true}
}

func (s *streamState) fault(data []byte) outcome {
	var told struct {
		Error   *wireFault `json:"error"`
		Code    string     `json:"code"`
		Message string     `json:"message"`
		Param   string     `json:"param"`
	}
	if err := json.Unmarshal(data, &told); err != nil {
		return outcome{err: unreadableEvent(err), ended: true}
	}
	if told.Error != nil {
		return outcome{err: &refusal{fault: *told.Error}, ended: true}
	}
	return outcome{err: &refusal{fault: wireFault{Code: told.Code, Message: told.Message, Param: told.Param}}, ended: true}
}

func unreadableEvent(err error) error {
	return errors.New(errors.External, "the model provider sent an event that could not be read").WithInternal(err).WithRetry(0)
}

func streamBroken(err error) error {
	switch {
	case stderrors.Is(err, io.EOF):
		return errors.New(errors.External, "the model provider closed the stream before the answer was complete").WithRetry(0)
	case stderrors.Is(err, errEventTooLarge):
		return errors.New(errors.External, "the model provider sent an event larger than the client reads").WithRetry(0)
	default:
		return err
	}
}

type liveStream struct {
	body    io.ReadCloser
	events  *sseReader
	state   *streamState
	release context.CancelFunc
	pending []port.Delta
	ended   bool
}

func (s *liveStream) close() {
	s.release()
	s.body.Close()
}

func (s *liveStream) prime() error {
	for {
		event, err := s.events.next()
		if err != nil {
			return streamBroken(err)
		}
		if admission(event) {
			continue
		}
		got := s.state.read(event)
		if got.err != nil {
			return got.err
		}
		s.pending, s.ended = got.deltas, got.ended
		return nil
	}
}

func (c *Client) Stream(ctx context.Context, req port.Request) (<-chan port.Delta, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	call, cancel := c.withTimeout(ctx)
	ex, err := c.prepare(call, req, true)
	if err != nil {
		cancel()
		return nil, err
	}

	live, err := c.open(call, ex)
	if err != nil {
		cancel()
		return nil, c.classify(ctx, ex, err)
	}

	out := make(chan port.Delta)
	go func() {
		defer cancel()
		defer close(out)
		defer live.close()
		c.pump(ctx, ex, live, out)
	}()
	return out, nil
}

func (c *Client) open(call context.Context, ex exchange) (*liveStream, error) {
	attempt, release := context.WithCancel(call)
	live, err := c.openOnce(attempt, release, ex)
	if err != nil {
		release()
		return nil, err
	}
	return live, nil
}

func (c *Client) openOnce(ctx context.Context, release context.CancelFunc, ex exchange) (*liveStream, error) {
	resp, err := c.post(ctx, ex)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		payload, readErr := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
		if readErr != nil {
			return nil, readErr
		}
		return nil, refusalOf(resp.StatusCode, resp.Header, payload)
	}

	live := &liveStream{
		body:    resp.Body,
		events:  newSSEReader(resp.Body, maxEventBytes),
		state:   newStreamState(ex.body.ServiceTier),
		release: release,
	}
	if err = live.prime(); err != nil {
		resp.Body.Close()
		return nil, err
	}
	return live, nil
}

func (c *Client) pump(caller context.Context, ex exchange, live *liveStream, out chan<- port.Delta) {
	for _, delta := range live.pending {
		if !send(caller, out, delta) {
			return
		}
	}
	for !live.ended {
		event, err := live.events.next()
		if err != nil {
			c.fail(caller, ex, out, streamBroken(err))
			return
		}

		got := live.state.read(event)
		for _, delta := range got.deltas {
			if !send(caller, out, delta) {
				return
			}
		}
		if got.err != nil {
			c.fail(caller, ex, out, got.err)
			return
		}
		live.ended = got.ended
	}
}

func (c *Client) fail(caller context.Context, ex exchange, out chan<- port.Delta, err error) {
	if caller.Err() != nil {
		return
	}
	send(caller, out, port.Delta{Err: c.classify(caller, ex, err)})
}

func send(ctx context.Context, out chan<- port.Delta, delta port.Delta) bool {
	if ctx.Err() != nil {
		return false
	}
	select {
	case out <- delta:
		return true
	case <-ctx.Done():
		return false
	}
}
