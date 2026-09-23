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

	port "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/domain/llm"
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
	next port.Client
	mode string
	dir  string
}

func New(next port.Client, mode, dir string) *Client {
	if mode == "" {
		mode = ModeOff
	}
	if dir == "" {
		dir = DefaultDir
	}
	return &Client{next: next, mode: mode, dir: dir}
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

		usage := stored.Response.Usage
		out := make(chan port.Delta, 2)
		out <- port.Delta{Text: stored.Response.Text}
		out <- port.Delta{Done: true, Usage: &usage}
		close(out)
		return out, nil
	case ModeRecord:
		return c.recordStream(ctx, req)
	default:
		return c.next.Stream(ctx, req)
	}
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
			text    []byte
			usage   llm.Usage
			failure bool
		)
		for delta := range deltas {
			text = append(text, delta.Text...)
			if delta.Usage != nil {
				usage = *delta.Usage
			}
			if delta.Err != nil {
				failure = true
			}
			select {
			case out <- delta:
			case <-ctx.Done():
				return
			}
		}
		if failure {
			return
		}

		response := port.Response{Text: string(text), Usage: usage, FinishReason: port.FinishStop}
		if saveErr := c.save(req, response); saveErr != nil {
			select {
			case out <- port.Delta{Err: saveErr}:
			case <-ctx.Done():
			}
		}
	}()
	return out, nil
}

func (c *Client) save(req port.Request, resp port.Response) error {
	name, canonical, err := fixtureName(req)
	if err != nil {
		return err
	}

	encoded, err := json.MarshalIndent(fixture{Request: req, Response: resp}, "", "  ")
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
	name, canonical, err := fixtureName(req)
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

func fixtureName(req port.Request) (name, canonical string, err error) {
	req.Meta = port.CallMeta{}

	encoded, err := json.Marshal(req)
	if err != nil {
		return "", "", errors.Wrap(err, errors.Internal, "encode the llm request")
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]) + ".json", string(encoded), nil
}
