package tools

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/davidmovas/postulator/internal/application/runs"
	"github.com/davidmovas/postulator/internal/application/sitemodel"
	"github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	pagesFieldsName = "pages_fields"
	opGet           = "get"
	opSet           = "set"
	quotedRunes     = 120
)

type pagesFieldsArgs struct {
	Op     string   `json:"op" enum:"get,set" description:"set writes one field"`
	PageID string   `json:"pageId,omitempty" description:"Page id, or give path"`
	Path   string   `json:"path,omitempty" description:"Page path"`
	Keys   []string `json:"keys,omitempty" description:"get: only these keys"`
	Key    string   `json:"key,omitempty" description:"set: field key"`
	Value  string   `json:"value,omitempty" description:"set: new value"`
	Hash   string   `json:"hash,omitempty" description:"set: hash from get, empty if the field was"`
	Cursor string   `json:"cursor,omitempty" description:"get: nextCursor of the last call"`
	Limit  int      `json:"limit,omitempty" description:"get: default 50"`
}

type PagesFieldsResponse struct {
	Fields *sitemodel.FieldsView `json:"fields,omitempty"`
	Edit   *runs.EditResponse    `json:"edit,omitempty"`
}

func pagesFields(deps Deps) Tool {
	tool := confirming(newSiteTool(Def{
		Name: pagesFieldsName,
		Description: "Read a page's fields (current, base, hash, owner), or write one to the site as a run the person " +
			"confirms and runs_revert undoes.",
		Risk: RiskWrite,
	}, func(ctx context.Context, b Binding, in pagesFieldsArgs) (PagesFieldsResponse, error) {
		if in.Op != opGet && in.Op != opSet {
			return PagesFieldsResponse{}, errors.New(errors.Invalid, "pages_fields takes the op get or set").WithDetail("field", "op")
		}
		read, err := deps.SiteModel.Fields(ctx, sitemodel.FieldsRequest{
			SiteID: b.SiteID, PageID: in.PageID, Path: in.Path, Keys: in.Keys, Cursor: in.Cursor, Limit: in.Limit,
		})
		if err != nil || in.Op != opSet {
			return PagesFieldsResponse{Fields: &read}, err
		}
		started, err := deps.Runs.StartEdit(ctx, runs.EditRequest{SiteID: b.SiteID, PageID: read.PageID, Edit: in.edit()})
		if err != nil {
			return PagesFieldsResponse{}, err
		}
		return PagesFieldsResponse{Edit: &started}, nil
	}), func(ctx context.Context, b Binding, in pagesFieldsArgs) (string, error) {
		if in.Op != opSet {
			return "", nil
		}
		return deps.fieldSummary(ctx, b, in)
	})
	return editableOn(readsOn(tool, opGet), opSet, agent.EditField{Name: "value", Path: "value"})
}

func (in *pagesFieldsArgs) edit() run.Edit {
	return run.Edit{Op: run.EditSetField, Field: in.Key, Value: in.Value, Hash: in.Hash}
}

func (deps Deps) fieldSummary(ctx context.Context, b Binding, in pagesFieldsArgs) (string, error) {
	if err := in.edit().Validate(); err != nil {
		return "", err
	}
	read, err := deps.SiteModel.Fields(ctx, sitemodel.FieldsRequest{SiteID: b.SiteID, PageID: in.PageID, Path: in.Path, Keys: []string{in.Key}})
	if err != nil {
		return "", err
	}
	if !read.OnSite {
		return "", errors.New(errors.Invalid, "the page "+read.Path+" is not on the site, so there is no field to write there").
			WithDetail("field", "pageId")
	}
	current, hash := "", ""
	if len(read.Fields.Items) > 0 {
		current, hash = read.Fields.Items[0].Current, read.Fields.Items[0].Hash
	}
	if hash != in.Hash {
		return "", errors.New(errors.Conflict, "the "+in.Key+" of "+read.Path+" changed since you read it; get it again").
			WithDetail("field", "hash")
	}
	return "Set the " + in.Key + " of " + read.Path + " on the site from " + quoted(current) + " to " + quoted(in.Value) +
		"; nothing else on the page changes", nil
}

func quoted(text string) string {
	text = strings.Join(strings.Fields(text), " ")
	if text == "" {
		return "nothing"
	}
	if utf8.RuneCountInString(text) > quotedRunes {
		text = strings.TrimSpace(string([]rune(text)[:quotedRunes])) + "…"
	}
	return "“" + text + "”"
}
