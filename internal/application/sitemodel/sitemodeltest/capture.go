package sitemodeltest

import (
	"embed"
	"encoding/json"
	"path"

	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/domain/render"
	"github.com/davidmovas/postulator/internal/domain/site"
)

//go:embed testdata
var captures embed.FS

func Stands() []string {
	return []string{"replica", "vanilla"}
}

type Capture struct {
	Site     site.Site                 `json:"site"`
	Pages    []pagemap.Page            `json:"pages"`
	Entities []graph.Entity            `json:"entities"`
	Edges    []graph.Edge              `json:"edges"`
	Types    []pagetype.Type           `json:"types"`
	Scopes   []field.Scope             `json:"scopes"`
	Terms    map[string][]pagemap.Term `json:"terms"`
	Fields   map[string][]field.Value  `json:"fields"`
	Bodies   []block.Snapshot          `json:"bodies"`
	Written  []block.Written           `json:"written"`
	Renders  []render.Record           `json:"renders"`
	Links    []pagemap.PageLink        `json:"links"`
	Findings []finding.Finding         `json:"findings"`
}

func Stand(name string) (*Memory, error) {
	held, err := Read(name)
	if err != nil {
		return nil, err
	}
	return held.Memory(), nil
}

func Read(name string) (Capture, error) {
	raw, err := captures.ReadFile(path.Join("testdata", name, "site.json"))
	if err != nil {
		return Capture{}, err
	}
	var held Capture
	if err := json.Unmarshal(raw, &held); err != nil {
		return Capture{}, err
	}
	return held, nil
}

func Decode(raw []byte) (*Memory, error) {
	var held Capture
	if err := json.Unmarshal(raw, &held); err != nil {
		return nil, err
	}
	return held.Memory(), nil
}

func (c Capture) Memory() *Memory {
	return &Memory{
		Site: c.Site, Pages: c.Pages, Entities: c.Entities, Edges: c.Edges,
		Types: pagetype.NewSet(c.Types, c.Scopes), Terms: c.Terms, Fields: c.Fields, Bodies: c.Bodies,
		Written: c.Written, Renders: c.Renders, Links: c.Links, Findings: c.Findings,
	}
}
