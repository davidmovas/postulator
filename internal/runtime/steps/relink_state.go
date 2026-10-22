package steps

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	checkpointRelinkNeighbors = "relink_neighbors.state"
	checkpointRelinkPage      = "relink_page.state"
)

type relinkWrite struct {
	Outcome NeighborResult `json:"outcome"`
	After   string         `json:"after"`
	Ours    []string       `json:"ours"`
	Planned bool           `json:"planned"`
}

type relinkState struct {
	wake     time.Time
	Writes   []relinkWrite    `json:"writes"`
	Inputs   string           `json:"inputs"`
	Result   RelinkResult     `json:"result"`
	Page     RelinkPageResult `json:"page"`
	Publish  PublishResult    `json:"publish"`
	Done     int              `json:"done"`
	Seq      int              `json:"seq"`
	Planned  bool             `json:"planned"`
	Finished bool             `json:"finished"`
}

func relinkInputs(sc *run.StepContext, kinds ...run.ArtifactKind) string {
	digest := sha256.New()
	for _, kind := range kinds {
		hash := ""
		if artifact, held := sc.Artifacts[kind]; held {
			hash = artifact.Hash
			if hash == "" {
				hash = run.HashBlob(artifact.Blob)
			}
		}
		digest.Write([]byte(string(kind) + ":" + hash))
		digest.Write([]byte{0})
	}
	return hex.EncodeToString(digest.Sum(nil))
}

func relinkStateOf(sc *run.StepContext, wake time.Time, key string, kinds ...run.ArtifactKind) (relinkState, error) {
	inputs := relinkInputs(sc, kinds...)
	held, found, err := run.Get[relinkState](sc.Check, key)
	if err != nil {
		return relinkState{}, err
	}
	if !found || held.Finished || !held.Planned || held.Inputs != inputs {
		return relinkState{wake: wake, Inputs: inputs, Writes: []relinkWrite{}, Seq: held.Seq}, nil
	}
	held.wake = wake
	return held, nil
}

func (s *relinkState) checkpoint(key string) (run.Checkpoint, error) {
	s.Seq++
	check := run.NewCheckpoint()
	if err := run.Set(check, key, *s); err != nil {
		return nil, err
	}
	return check, nil
}

func (s *relinkState) wait(key, message string) (run.Result, error) {
	check, err := s.checkpoint(key)
	if err != nil {
		return run.Result{}, err
	}
	return run.Result{Next: run.TransitionWait, WakeAt: s.wake, Checkpoint: check, Message: message}, nil
}

type landing struct {
	content string
	hash    string
	held    string
	written bool
}

func landRelinkedBody(ctx context.Context, client *wp.Client, page pagemap.Page, after, beforeHash string) (landing, error) {
	written, err := client.PutRaw(ctx, onSiteType(page), *page.WPID, after, beforeHash)
	if err == nil {
		return landing{content: written.Content, hash: written.Hash, written: true}, nil
	}
	if !errors.IsCode(err, errors.Conflict) {
		return landing{}, err
	}
	stored, readErr := client.GetRaw(ctx, onSiteType(page), *page.WPID)
	if readErr != nil {
		return landing{}, readErr
	}
	if !block.SameRender(stored.Content, after) {
		return landing{held: stored.ContentHash}, nil
	}
	return landing{content: stored.Content, hash: stored.ContentHash, written: true}, nil
}

func (w relinkWrite) stored(content string) splicedBody {
	composed, held := block.Parse(w.After).Blocks, block.Parse(content).Blocks
	at := block.Align(w.After, content, w.Ours)
	ours := make([]string, 0, len(w.Ours))
	for i := range composed {
		if at[i] < 0 || !slices.Contains(w.Ours, composed[i].Hash) {
			continue
		}
		if hash := held[at[i]].Hash; !slices.Contains(ours, hash) {
			ours = append(ours, hash)
		}
	}
	return splicedBody{raw: content, ours: ours}
}

func (s *relinkState) bodies(index pagemap.Index) []relinkedBody {
	out := make([]relinkedBody, 0, len(s.Writes))
	for i := range s.Writes {
		page, ok := index.ByID(s.Writes[i].Outcome.PageID)
		if !ok {
			continue
		}
		out = append(out, relinkedBody{page: page, before: s.Writes[i].Outcome.Before.HTML, after: s.Writes[i].After})
	}
	return out
}
