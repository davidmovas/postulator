package sitemodel

import (
	"cmp"
	"context"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const (
	OwnerOurs   = "ours"
	OwnerClient = "client"
	OwnerStore  = "store"

	blockExcerptRunes = 120
	blockTextRunes    = 3000
)

type FieldsRequest struct {
	SiteID string   `json:"siteId"`
	PageID string   `json:"pageId,omitempty"`
	Path   string   `json:"path,omitempty"`
	Keys   []string `json:"keys,omitempty"`
	Cursor string   `json:"cursor,omitempty"`
	Limit  int      `json:"limit,omitempty"`
}

type FieldRow struct {
	Key      string `json:"key"`
	Current  string `json:"current"`
	Base     string `json:"base"`
	Hash     string `json:"hash"`
	Owner    string `json:"owner"`
	Writable bool   `json:"writable"`
	Enabled  bool   `json:"enabled"`
	AIWrites bool   `json:"aiWrites"`
}

type FieldsView struct {
	PageID string                `json:"pageId"`
	Path   string                `json:"path"`
	OnSite bool                  `json:"onSite"`
	Fields paging.List[FieldRow] `json:"fields"`
}

type BlocksRequest struct {
	SiteID string `json:"siteId"`
	PageID string `json:"pageId,omitempty"`
	Path   string `json:"path,omitempty"`
	Cursor string `json:"cursor,omitempty"`
	Limit  int    `json:"limit,omitempty"`
	Text   bool   `json:"text,omitempty"`
}

type BlockLink struct {
	Anchor string `json:"anchor"`
	Href   string `json:"href"`
	Path   string `json:"path,omitempty"`
}

type BlockRow struct {
	Index   int         `json:"index"`
	Kind    string      `json:"kind"`
	Name    string      `json:"name,omitempty"`
	Owner   string      `json:"owner"`
	Hash    string      `json:"hash"`
	Dynamic bool        `json:"dynamic,omitempty"`
	Excerpt string      `json:"excerpt,omitempty"`
	Text    string      `json:"text,omitempty"`
	From    int         `json:"from,omitempty"`
	Runes   int         `json:"runes,omitempty"`
	Links   []BlockLink `json:"links,omitempty"`
}

type BlocksView struct {
	PageID   string                `json:"pageId"`
	Path     string                `json:"path"`
	OnSite   bool                  `json:"onSite"`
	BodyKind string                `json:"bodyKind"`
	BodyHash string                `json:"bodyHash"`
	Rewrites bool                  `json:"rewrites"`
	Blocks   paging.List[BlockRow] `json:"blocks"`
}

func (s *Service) Fields(ctx context.Context, req FieldsRequest) (FieldsView, error) {
	var view FieldsView
	err := s.deps.Reader.Read(ctx, func(c context.Context) error {
		page, err := s.named(c, pageName{SiteID: req.SiteID, PageID: req.PageID, Path: req.Path})
		if err != nil {
			return err
		}
		values, err := s.deps.Fields.ListBySite(c, req.SiteID)
		if err != nil {
			return err
		}
		set, err := s.deps.Types.Set(c, req.SiteID)
		if err != nil {
			return err
		}
		kind := set.TypeOf(page)
		held := slices.Clone(values[page.ID])
		slices.SortFunc(held, func(a, b field.Value) int { return cmp.Compare(a.Key, b.Key) })
		rows := make([]FieldRow, 0, len(held))
		for i := range held {
			value := held[i]
			if len(req.Keys) > 0 && !slices.Contains(req.Keys, string(value.Key)) {
				continue
			}
			spec, _ := field.Lookup(value.Key)
			scope, _ := set.Scope(kind.ID, value.Key)
			rows = append(rows, FieldRow{
				Key: string(value.Key), Current: value.Current, Base: value.Base, Hash: value.CurrentHash,
				Owner: fieldOwner(value, spec), Writable: !spec.Denied && !spec.Person && spec.Write != field.None,
				Enabled: scope.Enabled, AIWrites: scope.AIWrites,
			})
		}
		keyset := paging.Keyset[FieldRow]{
			IDColumn: "field_key", ID: func(row FieldRow) string { return row.Key },
			Keys: []paging.SortKey[FieldRow]{paging.TextKey("key", "field_key", func(row FieldRow) any { return row.Key })},
		}
		listed, err := keyset.Page(rows, paging.Request{After: paging.Cursor(req.Cursor), Limit: dto.PageSize(req.Limit)})
		if err != nil {
			return err
		}
		view = FieldsView{PageID: page.ID, Path: page.Path, OnSite: page.WPID != nil, Fields: listed}
		return nil
	})
	return view, err
}

func fieldOwner(value field.Value, spec field.Spec) string {
	switch {
	case spec.Denied:
		return OwnerStore
	case value.WrittenRunID != nil && value.CurrentHash == value.BaseHash:
		return OwnerOurs
	default:
		return OwnerClient
	}
}

func (s *Service) Blocks(ctx context.Context, req BlocksRequest) (BlocksView, error) {
	var view BlocksView
	err := s.deps.Reader.Read(ctx, func(c context.Context) error {
		page, err := s.named(c, pageName{SiteID: req.SiteID, PageID: req.PageID, Path: req.Path})
		if err != nil {
			return err
		}
		bodies, err := s.deps.Bodies.ListCurrent(c, []string{page.ID})
		if err != nil {
			return err
		}
		written, err := s.deps.Bodies.ListWritten(c, page.ID)
		if err != nil {
			return err
		}
		set, err := s.deps.Types.Set(c, req.SiteID)
		if err != nil {
			return err
		}
		scope, _ := set.Scope(set.TypeOf(page).ID, bodyField(page))
		raw := ""
		if len(bodies) > 0 {
			raw = bodies[0].Raw
		}
		parsed := block.Parse(raw)
		rows := blockRows(parsed, written)
		if req.Text {
			owner, ownerErr := s.deps.Sites.Get(c, req.SiteID)
			if ownerErr != nil {
				return ownerErr
			}
			rows = textRows(parsed, rows, pagemap.NewSite(owner.BaseURL), page.Path)
		}
		keyset := paging.Keyset[BlockRow]{
			IDColumn: "block_part", ID: func(row BlockRow) string { return strconv.Itoa(row.Index) + ":" + strconv.Itoa(row.From) },
			Keys: []paging.SortKey[BlockRow]{
				paging.IntKey("index", "block_index", func(row BlockRow) any { return row.Index }),
				paging.IntKey("from", "block_from", func(row BlockRow) any { return row.From }),
			},
		}
		listed, err := keyset.Page(rows, paging.Request{After: paging.Cursor(req.Cursor), Limit: dto.PageSize(req.Limit)})
		if err != nil {
			return err
		}
		view = BlocksView{
			PageID: page.ID, Path: page.Path, OnSite: page.WPID != nil, BodyKind: string(parsed.Kind),
			BodyHash: block.BodyHash(raw), Rewrites: scope.Enabled && scope.Rewrites, Blocks: listed,
		}
		return nil
	})
	return view, err
}

func bodyField(page pagemap.Page) field.Key {
	if page.WPType == pagemap.WPProduct {
		return field.ProductDescription
	}
	return field.Body
}

func blockRows(body block.Body, written []block.Written) []BlockRow {
	ours := make(map[string]bool, len(written))
	for i := range written {
		ours[written[i].BlockHash] = true
	}
	rows := make([]BlockRow, 0, len(body.Blocks))
	for _, held := range body.Blocks {
		owner := OwnerClient
		if ours[held.Hash] {
			owner = OwnerOurs
		}
		rows = append(rows, BlockRow{
			Index: held.Index, Kind: string(held.Kind), Name: held.Name, Owner: owner, Hash: held.Hash,
			Dynamic: held.Dynamic, Excerpt: blockExcerpt(held.Text()),
		})
	}
	return rows
}

func textRows(body block.Body, rows []BlockRow, site pagemap.Site, document string) []BlockRow {
	out := make([]BlockRow, 0, len(rows))
	for i := range rows {
		row, held := rows[i], body.Blocks[i]
		row.Excerpt = ""
		row.Links = blockLinks(held, site, document)
		runes := []rune(strings.Join(strings.Fields(held.Text()), " "))
		row.Runes = len(runes)
		if len(runes) == 0 {
			out = append(out, row)
			continue
		}
		for from := 0; from < len(runes); from += blockTextRunes {
			part := row
			part.From, part.Text = from, string(runes[from:min(from+blockTextRunes, len(runes))])
			if from > 0 {
				part.Links = nil
			}
			out = append(out, part)
		}
	}
	return out
}

func blockLinks(held block.Block, site pagemap.Site, document string) []BlockLink {
	if held.Dynamic || held.Kind == block.Shortcode {
		return nil
	}
	doc, err := content.Parse(held.Raw)
	if err != nil {
		return nil
	}
	found := doc.Links()
	out := make([]BlockLink, 0, len(found))
	for _, link := range found {
		row := BlockLink{Anchor: strings.Join(strings.Fields(link.Anchor), " "), Href: link.Href}
		if path, kind := site.ResolveAt(document, link.Href); kind == pagemap.LinkPath {
			row.Path = path
		}
		out = append(out, row)
	}
	return out
}

func blockExcerpt(text string) string {
	text = strings.Join(strings.Fields(text), " ")
	if utf8.RuneCountInString(text) <= blockExcerptRunes {
		return text
	}
	return strings.TrimSpace(string([]rune(text)[:blockExcerptRunes])) + "…"
}
