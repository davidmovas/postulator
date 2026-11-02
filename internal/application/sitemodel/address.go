package sitemodel

import (
	"context"
	"slices"

	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

type AddressRequest struct {
	SiteID string        `json:"siteId"`
	Folder string        `json:"folder,omitempty"`
	Within string        `json:"within,omitempty"`
	Cursor paging.Cursor `json:"cursor,omitempty"`
	Limit  int           `json:"limit,omitempty"`
}

type AddressNode struct {
	Path     string               `json:"path"`
	Children int                  `json:"children"`
	Pages    int                  `json:"pages"`
	Page     *contextpack.MapNode `json:"page,omitempty"`
}

type AddressView struct {
	Folder string                   `json:"folder"`
	Nodes  paging.List[AddressNode] `json:"nodes"`
}

func (s *Service) AddressTree(ctx context.Context, req AddressRequest) (AddressView, error) {
	switch {
	case req.SiteID == "":
		return AddressView{}, errors.New(errors.Invalid, "name the site whose pages to read").WithDetail("field", "siteId")
	case req.Folder != "" && !contextpack.IsFolder(req.Folder):
		return AddressView{}, errors.New(errors.Invalid, "a folder is a path with both slashes, such as /products/").
			WithDetail("field", "folder").WithDetail("value", req.Folder)
	}
	view := AddressView{Folder: req.Folder}
	err := s.onMap(ctx, req.SiteID, func(c context.Context) error {
		var err error
		view.Nodes, err = s.addressed(c, req)
		return err
	})
	return view, err
}

func (s *Service) addressed(ctx context.Context, req AddressRequest) (paging.List[AddressNode], error) {
	rows, err := s.deps.Maps.ListBySite(ctx, req.SiteID)
	if err != nil {
		return paging.List[AddressNode]{}, err
	}
	if req.Within != "" {
		rows = slices.DeleteFunc(rows, func(row contextpack.MapRow) bool { return !row.Typed(req.Within) })
	}
	keyset := paging.Keyset[contextpack.Folder]{
		IDColumn: "path",
		ID:       func(folder contextpack.Folder) string { return folder.Path },
		Keys: []paging.SortKey[contextpack.Folder]{
			paging.TextKey("path", "path", func(folder contextpack.Folder) any { return folder.Path }),
		},
	}
	page, err := keyset.Page(contextpack.Folders(rows, req.Folder), paging.Request{After: req.Cursor, Limit: clamped(req.Limit, mapDefaultLimit, mapMaxLimit)})
	if err != nil {
		return paging.List[AddressNode]{}, err
	}

	byID := make(map[string]contextpack.MapRow, len(rows))
	for i := range rows {
		byID[rows[i].PageID] = rows[i]
	}
	held := make([]contextpack.MapRow, 0, len(page.Items))
	for _, folder := range page.Items {
		if folder.PageID != "" {
			held = append(held, byID[folder.PageID])
		}
	}
	read, err := s.mapNodes(ctx, req.SiteID, held, nil, false)
	if err != nil {
		return paging.List[AddressNode]{}, err
	}
	if whyErr := s.whys(ctx, read, read.nodes, nil); whyErr != nil {
		return paging.List[AddressNode]{}, whyErr
	}
	if bodyErr := s.bodiesOf(ctx, read.nodes); bodyErr != nil {
		return paging.List[AddressNode]{}, bodyErr
	}
	nodes := make(map[string]*contextpack.MapNode, len(read.nodes))
	for i := range read.nodes {
		nodes[read.nodes[i].Page.PageID] = &read.nodes[i]
	}

	out := paging.List[AddressNode]{Cursors: page.Cursors, HasMore: page.HasMore, Items: make([]AddressNode, 0, len(page.Items))}
	for _, folder := range page.Items {
		out.Items = append(out.Items, AddressNode{Path: folder.Path, Children: folder.Children, Pages: folder.Pages, Page: nodes[folder.PageID]})
	}
	return out, nil
}
