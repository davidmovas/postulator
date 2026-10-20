package steps

import (
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/run"
)

const (
	checkpointPublish = "publish"

	unitCheck = "check"
	unitRaw   = "raw"
	unitItem  = "item"
	unitStore = "store"
	unitSEO   = "seo"
)

var publishInputKinds = []run.ArtifactKind{
	run.ArtifactDraft, run.ArtifactBodyHTML, run.ArtifactMeta, run.ArtifactFields, run.ArtifactImages,
}

type itemAnswer struct {
	Modified time.Time `json:"modified"`
	Title    string    `json:"title"`
	Slug     string    `json:"slug"`
	Status   string    `json:"status"`
	Link     string    `json:"link"`
	ID       int64     `json:"id"`
	Parent   int64     `json:"parent"`
}

func answerOf(item wp.Item) itemAnswer {
	return itemAnswer{
		Modified: item.Modified, Title: item.Title, Slug: item.Slug, Status: item.Status, Link: item.Link,
		ID: item.ID, Parent: item.Parent,
	}
}

func (a itemAnswer) item() wp.Item {
	return wp.Item{
		Modified: a.Modified, Title: a.Title, Slug: a.Slug, Status: a.Status, Link: a.Link, ID: a.ID, Parent: a.Parent,
	}
}

type productAnswer struct {
	Modified         time.Time `json:"modified"`
	Name             string    `json:"name"`
	Slug             string    `json:"slug"`
	Status           string    `json:"status"`
	Permalink        string    `json:"permalink"`
	ShortDescription string    `json:"shortDescription"`
	ID               int64     `json:"id"`
}

func productAnswerOf(product wp.Product) productAnswer {
	return productAnswer{
		Modified: product.Modified, Name: product.Name, Slug: product.Slug, Status: product.Status,
		Permalink: product.Permalink, ShortDescription: product.ShortDescription, ID: product.ID,
	}
}

func (a productAnswer) product() wp.Product {
	return wp.Product{
		Modified: a.Modified, Name: a.Name, Slug: a.Slug, Status: a.Status, Permalink: a.Permalink,
		ShortDescription: a.ShortDescription, ID: a.ID,
	}
}

type itemPlan struct {
	Excerpt   *string `json:"excerpt,omitempty"`
	Title     string  `json:"title"`
	Slug      string  `json:"slug"`
	Status    string  `json:"status"`
	Parent    int64   `json:"parent"`
	Featured  int64   `json:"featured"`
	ViaRaw    bool    `json:"viaRaw"`
	KeepTitle bool    `json:"keepTitle"`
}

func (p itemPlan) carries() bool {
	return !p.KeepTitle || p.Excerpt != nil || !p.ViaRaw || p.Featured != 0
}

type seoPlan struct {
	Previous  *wp.SEOMeta       `json:"previous,omitempty"`
	Meta      wp.SEOMeta        `json:"meta"`
	Skipped   []string          `json:"skipped"`
	Findings  []content.Finding `json:"findings"`
	Send      bool              `json:"send"`
	Canonical bool              `json:"canonical"`
}

type storePlan struct {
	ShortDescription *string             `json:"shortDescription,omitempty"`
	Attributes       []SnapshotAttribute `json:"attributes,omitempty"`
	Images           []int64             `json:"images,omitempty"`
	Snapshot         ProductSnapshot     `json:"snapshot"`
	Expected         string              `json:"expected"`
	Saved            *productAnswer      `json:"saved,omitempty"`
	AttributesSent   bool                `json:"attributesSent"`
	ImagesSent       bool                `json:"imagesSent"`
}

func (p storePlan) update() wp.UpdateProduct {
	update := wp.UpdateProduct{ShortDescription: p.ShortDescription}
	if p.AttributesSent {
		attributes := ProductSnapshot{Attributes: p.Attributes}.StoreAttributes()
		update.Attributes = &attributes
	}
	if p.ImagesSent {
		images := slices.Clone(p.Images)
		update.Images = &images
	}
	return update
}

type publishPlan struct {
	Store        *storePlan  `json:"store,omitempty"`
	Item         *itemAnswer `json:"item,omitempty"`
	Writes       []string    `json:"writes"`
	Kept         []field.Key `json:"kept"`
	Request      itemPlan    `json:"request"`
	SEO          seoPlan     `json:"seo"`
	Final        string      `json:"final"`
	Stored       *string     `json:"stored,omitempty"`
	Theirs       string      `json:"theirs"`
	Part         string      `json:"part"`
	Previous     string      `json:"previous"`
	SiteHash     string      `json:"siteHash"`
	SiteTitle    string      `json:"siteTitle"`
	Excerpt      string      `json:"excerpt"`
	WPID         int64       `json:"wpId"`
	SiteFeatured int64       `json:"siteFeatured"`
	Create       bool        `json:"create"`
	Plugin       bool        `json:"plugin"`
	Product      bool        `json:"product"`
	Unstable     bool        `json:"unstable"`
}

func (p *publishPlan) kept() keptFields {
	out := keptFields{}
	for _, key := range p.Kept {
		out[key] = true
	}
	return out
}

func (k keptFields) keys() []field.Key {
	out := make([]field.Key, 0, len(k))
	for key, held := range k {
		if held {
			out = append(out, key)
		}
	}
	slices.Sort(out)
	return out
}

type publishState struct {
	Plan     *publishPlan  `json:"plan,omitempty"`
	Inputs   string        `json:"inputs"`
	Done     []string      `json:"done"`
	Result   PublishResult `json:"result"`
	Seq      int           `json:"seq"`
	Finished bool          `json:"finished"`
}

func publishStateOf(sc *run.StepContext) (publishState, error) {
	inputs := publishInputs(sc)
	held, found, err := run.Get[publishState](sc.Check, checkpointPublish)
	if err != nil {
		return publishState{}, err
	}
	if !found || held.Finished || held.Plan == nil || held.Inputs != inputs {
		return publishState{Inputs: inputs, Done: []string{}, Seq: held.Seq}, nil
	}
	return held, nil
}

func publishInputs(sc *run.StepContext) string {
	digest := sha256.New()
	for _, kind := range publishInputKinds {
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

func (s *publishState) next() string {
	for _, write := range s.Plan.Writes {
		if !slices.Contains(s.Done, write) {
			return write
		}
	}
	return ""
}

func (s *publishState) wrote(write string) {
	s.Done = append(s.Done, write)
}

func (s *publishState) carry(result run.Result) (run.Result, error) {
	s.Seq++
	check := run.NewCheckpoint()
	if err := run.Set(check, checkpointPublish, *s); err != nil {
		return run.Result{}, err
	}
	result.Checkpoint = check
	if len(s.Done) == 0 && !s.Finished {
		return result, nil
	}
	blob, err := encode(s.Result, "publish result")
	if err != nil {
		return run.Result{}, err
	}
	result.Artifacts = []run.Artifact{{Kind: run.ArtifactPublishResult, Blob: blob}}
	return result, nil
}

func (s *publishState) wait(message string) (run.Result, error) {
	return s.carry(run.Result{Next: run.TransitionWait, Message: message})
}

func (s *publishState) stopped(result run.Result) (run.Result, error) {
	return s.carry(result)
}

func (s *publishState) failed(cause error) (run.Result, error) {
	carried, err := s.carry(run.Result{})
	if err != nil {
		return run.Result{}, err
	}
	return carried, cause
}

func (s *publishState) finish(message string) (run.Result, error) {
	s.Finished = true
	return s.carry(run.Result{Message: message})
}
