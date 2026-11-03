package tools

import (
	"context"
	"strconv"
	"strings"

	"github.com/davidmovas/postulator/internal/application/graph"
	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type selectArgs struct {
	PageIDs []string `json:"pageIds,omitempty" description:"These pages"`
	Under   []string `json:"under,omitempty" description:"Each of these pages and every page below it"`
	OfType  string   `json:"ofType,omitempty" description:"With under, only pages of this type id below; alone, every page of it"`
	Gap     string   `json:"gap,omitempty" enum:"planned,empty,unmapped,orphan,missing_links,findings,no_hub" description:"Every page with this gap; not with under"`
	Except  []string `json:"except,omitempty" description:"Leave these pages out"`
}

func (in *selectArgs) selection(siteID string) pages.Selection {
	sel := pages.Selection{SiteID: siteID, PageIDs: in.PageIDs, Except: in.Except}
	for _, pageID := range in.Under {
		sel.Subtrees = append(sel.Subtrees, pages.Subtree{PageID: pageID, TypeID: in.OfType})
	}
	if len(in.Under) == 0 && (in.OfType != "" || in.Gap != "") {
		sel.Filter = &pages.PageFilter{PageTypeID: in.OfType, Gap: contextpack.Gap(in.Gap)}
	}
	return sel
}

func (in *selectArgs) checked() error {
	switch {
	case in == nil || (len(in.PageIDs) == 0 && len(in.Under) == 0 && in.OfType == "" && in.Gap == ""):
		return errors.New(errors.Invalid, "name the pages: pageIds, under, ofType or gap").WithDetail("field", "select")
	case in.Gap != "" && len(in.Under) > 0:
		return errors.New(errors.Invalid, "a gap selects across the whole site; name it without under").WithDetail("field", "select.gap")
	default:
		return nil
	}
}

type selected struct {
	ids   []string
	paths []string
}

func (deps Deps) selectPages(ctx context.Context, b Binding, in *selectArgs) (selected, error) {
	if err := in.checked(); err != nil {
		return selected{}, err
	}
	resolved, err := deps.Pages.Resolve(ctx, in.selection(b.SiteID))
	if err != nil {
		return selected{}, err
	}
	if len(resolved.PageIDs) == 0 {
		return selected{}, errors.New(errors.Invalid, "the selection holds no page").WithDetail("field", "select")
	}
	out := selected{ids: resolved.PageIDs, paths: make([]string, 0, len(resolved.PageIDs))}
	for _, pageID := range resolved.PageIDs {
		out.paths = append(out.paths, resolved.Paths[pageID])
	}
	return out, nil
}

func (s selected) named() string {
	shown := s.paths[:min(len(s.paths), maxPagesNamed)]
	said := plural(len(s.ids), "page") + " (" + strings.Join(shown, ", ")
	if more := len(s.paths) - len(shown); more > 0 {
		said += " and " + strconv.Itoa(more) + " more"
	}
	return said + ")"
}

func (deps Deps) ownPage(ctx context.Context, b Binding, pageID string) (pages.Page, error) {
	held, err := deps.Pages.Get(ctx, pages.GetRequest{ID: strings.TrimSpace(pageID)})
	if err != nil {
		return pages.Page{}, err
	}
	if held.Page.SiteID != b.SiteID {
		return pages.Page{}, errors.New(errors.NotFound, "page not found").WithDetail("pageId", pageID)
	}
	return held.Page, nil
}

func (deps Deps) entityName(ctx context.Context, entityID string) (string, error) {
	held, err := deps.Graph.GetEntity(ctx, graph.GetEntityRequest{ID: strings.TrimSpace(entityID)})
	if err != nil {
		return "", err
	}
	return held.Entity.Name, nil
}
