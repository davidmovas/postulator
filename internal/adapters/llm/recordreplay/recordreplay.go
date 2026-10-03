package recordreplay

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	stderrors "errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"

	port "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/settings"
)

const (
	ModeOff    = "off"
	ModeRecord = "record"
	ModeReplay = "replay"

	DefaultDir = "testdata/llm"

	fixtureMode   = 0o600
	directoryMode = 0o700
)

var modeSetting = settings.Enum("llm.recordReplayMode", ModeOff, []string{ModeOff, ModeRecord, ModeReplay})

func Mode(values *settings.Values) string {
	return modeSetting.Get(values)
}

type fixture struct {
	Request  port.Request  `json:"request"`
	Response port.Response `json:"response"`
}

type Client struct {
	next   port.Client
	redact func(args json.RawMessage) json.RawMessage
	mode   string
	dir    string
}

func New(next port.Client, mode, dir string, redact func(args json.RawMessage) json.RawMessage) *Client {
	if mode == "" {
		mode = ModeOff
	}
	if dir == "" {
		dir = DefaultDir
	}
	return &Client{next: next, redact: redact, mode: mode, dir: dir}
}

func (c *Client) Complete(ctx context.Context, req port.Request) (port.Response, error) {
	switch c.mode {
	case ModeReplay:
		stored, err := c.load(req)
		if err != nil {
			return port.Response{}, err
		}
		return stored.Response, nil
	case ModeRecord:
		resp, err := c.next.Complete(ctx, req)
		if err != nil {
			return port.Response{}, err
		}
		if saveErr := c.save(req, resp); saveErr != nil {
			return port.Response{}, saveErr
		}
		return resp, nil
	default:
		return c.next.Complete(ctx, req)
	}
}

func (c *Client) Stream(ctx context.Context, req port.Request) (<-chan port.Delta, error) {
	switch c.mode {
	case ModeReplay:
		stored, err := c.load(req)
		if err != nil {
			return nil, err
		}
		return replay(stored.Response), nil
	case ModeRecord:
		return c.recordStream(ctx, req)
	default:
		return c.next.Stream(ctx, req)
	}
}

func replay(resp port.Response) <-chan port.Delta {
	out := make(chan port.Delta, len(resp.Calls)+2)
	if resp.Text != "" {
		out <- port.Delta{Text: resp.Text}
	}
	for i := range resp.Calls {
		out <- port.Delta{Call: &resp.Calls[i]}
	}
	usage := resp.Usage
	out <- port.Delta{Done: true, Usage: &usage, Finish: resp.FinishReason, Tier: resp.Tier}
	close(out)
	return out
}

func (c *Client) recordStream(ctx context.Context, req port.Request) (<-chan port.Delta, error) {
	deltas, err := c.next.Stream(ctx, req)
	if err != nil {
		return nil, err
	}

	out := make(chan port.Delta)
	go func() {
		defer close(out)

		var (
			heard    port.Response
			text     []byte
			finished bool
			failure  bool
		)
		for delta := range deltas {
			text = append(text, delta.Text...)
			if delta.Call != nil {
				heard.Calls = append(heard.Calls, *delta.Call)
			}
			if delta.Err != nil {
				failure = true
			}
			if delta.Done {
				finished = true
				heard.FinishReason, heard.Tier = delta.Finish, delta.Tier
				if delta.Usage != nil {
					heard.Usage = *delta.Usage
				}
			}
			select {
			case out <- delta:
			case <-ctx.Done():
				return
			}
		}
		if failure || !finished {
			return
		}

		heard.Text = string(text)
		if saveErr := c.save(req, heard); saveErr != nil {
			select {
			case out <- port.Delta{Err: saveErr}:
			case <-ctx.Done():
			}
		}
	}()
	return out, nil
}

func (c *Client) save(req port.Request, resp port.Response) error {
	masked := c.masked(req)
	name, canonical, err := fixtureName(masked)
	if err != nil {
		return err
	}

	resp.Calls = c.maskedCalls(resp.Calls)
	encoded, err := json.MarshalIndent(fixture{Request: masked, Response: resp}, "", "  ")
	if err != nil {
		return errors.Wrap(err, errors.Internal, "encode the llm fixture")
	}
	if err = os.MkdirAll(c.dir, directoryMode); err != nil {
		return errors.Wrap(err, errors.Internal, "create the llm fixture directory")
	}
	if err = os.WriteFile(filepath.Join(c.dir, name), encoded, fixtureMode); err != nil {
		return errors.New(errors.Internal, "write the llm fixture").WithDetail("request", canonical).WithInternal(err)
	}
	return nil
}

func (c *Client) load(req port.Request) (fixture, error) {
	name, canonical, err := fixtureName(c.masked(req))
	if err != nil {
		return fixture{}, err
	}

	path := filepath.Join(c.dir, name)
	raw, err := os.ReadFile(path)
	if err != nil {
		if stderrors.Is(err, fs.ErrNotExist) {
			return fixture{}, errors.New(errors.NotFound, "no recorded llm fixture matches this request; record it first").
				WithDetail("fixture", path).
				WithDetail("request", canonical)
		}
		return fixture{}, errors.New(errors.Internal, "read the llm fixture").WithDetail("fixture", path).WithInternal(err)
	}

	var stored fixture
	if err = json.Unmarshal(raw, &stored); err != nil {
		return fixture{}, errors.New(errors.Internal, "decode the llm fixture").WithDetail("fixture", path).WithInternal(err)
	}
	return stored, nil
}

func (c *Client) masked(req port.Request) port.Request {
	req.Messages = slices.Clone(req.Messages)
	for i := range req.Messages {
		if held := req.Messages[i].Call; held != nil {
			call := *held
			call.Args = c.redact(call.Args)
			req.Messages[i].Call = &call
		}
	}
	return req
}

func (c *Client) maskedCalls(calls []port.ToolCall) []port.ToolCall {
	if len(calls) == 0 {
		return calls
	}
	masked := slices.Clone(calls)
	for i := range masked {
		masked[i].Args = c.redact(masked[i].Args)
	}
	return masked
}

func fixtureName(req port.Request) (name, canonical string, err error) {
	req.Meta = port.CallMeta{}

	encoded, err := json.Marshal(req)
	if err != nil {
		return "", "", errors.Wrap(err, errors.Internal, "encode the llm request")
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]) + ".json", string(encoded), nil
}
