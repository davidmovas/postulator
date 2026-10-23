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
	raw, err := captures.ReadFile(path.Join("testdata", name, "site.json"))
	if err != nil {
		return nil, err
	}
	return Decode(raw)
}

func Decode(raw []byte) (*Memory, error) {
	var held Capture
	if err := json.Unmarshal(raw, &held); err != nil {
		return nil, err
	}
	return &Memory{
		Site: held.Site, Pages: held.Pages, Entities: held.Entities, Edges: held.Edges,
		Types: pagetype.NewSet(held.Types, held.Scopes), Terms: held.Terms, Fields: held.Fields, Bodies: held.Bodies,
		Written: held.Written, Renders: held.Renders, Links: held.Links, Findings: held.Findings,
	}, nil
}
