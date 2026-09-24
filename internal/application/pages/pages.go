package pages

import (
	"context"
	"strings"

	"github.com/davidmovas/postulator/internal/application"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

func (s *Service) Create(ctx context.Context, req CreateRequest) (CreateResponse, error) {
	if err := requireSite(req.SiteID); err != nil {
		return CreateResponse{}, err
	}
	if _, err := s.sites.Get(ctx, req.SiteID); err != nil {
		return CreateResponse{}, err
	}

	status := pagemap.Status(req.Status)
	if req.Status == "" {
		status = pagemap.StatusPlanned
	}
	wpType := pagemap.WPType(req.WPType)
	if req.WPType == "" {
		wpType = pagemap.WPPage
	}
	now := s.now()
	page, err := pagemap.NewPage(pagemap.Page{
		ID:              id.New(),
		SiteID:          req.SiteID,
		Path:            req.Path,
		WPType:          wpType,
		Title:           req.Title,
		H1:              req.H1,
		MetaTitle:       req.MetaTitle,
		MetaDescription: req.MetaDescription,
		Canonical:       req.Canonical,
		PrimaryKeyword:  req.PrimaryKeyword,
		Keywords:        req.Keywords,
		Status:          status,
		EntityID:        req.EntityID,
		TemplateID:      req.TemplateID,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	if err != nil {
		return CreateResponse{}, err
	}

	doErr := s.uow.Do(ctx, func(c context.Context) error {
		entity, entityErr := s.entityFor(c, &page)
		if entityErr != nil {
			return entityErr
		}
		siblings, listErr := s.pages.ListBySite(c, page.SiteID)
		if listErr != nil {
			return listErr
		}
		index := pagemap.NewIndex(siblings)
		if verdictErr := s.verdict(c, page, index, entity); verdictErr != nil {
			return verdictErr
		}
		if parent, found := index.ByPath(pagemap.ParentPath(page.Path)); found {
			page.ParentPageID = &parent.ID
		}
		if insertErr := s.pages.Insert(c, page); insertErr != nil {
			return insertErr
		}
		return s.adopt(c, &page, siblings)
	})
	if doErr != nil {
		return CreateResponse{}, doErr
	}
	if publishErr := s.changed(page.SiteID); publishErr != nil {
		return CreateResponse{}, publishErr
	}
	return CreateResponse{Page: view(page)}, nil
}

func (s *Service) adopt(ctx context.Context, parent *pagemap.Page, siblings []pagemap.Page) error {
	for i := range siblings {
		child := siblings[i]
		if child.ParentPageID != nil || pagemap.ParentPath(child.Path) != parent.Path {
			continue
		}
		child.ParentPageID = &parent.ID
		child.UpdatedAt = parent.UpdatedAt
		if err := s.pages.Update(ctx, child); err != nil {
			return err
		}
	}
	return nil
}

func applyUpdate(current *pagemap.Page, req *UpdateRequest) (pagemap.Page, error) {
	next := *current
	if req.Path != nil {
		next.Path = *req.Path
	}
	if req.WPType != nil {
		next.WPType = pagemap.WPType(*req.WPType)
	}
	if req.Title != nil {
		next.Title = *req.Title
	}
	if req.H1 != nil {
		next.H1 = *req.H1
	}
	if req.MetaTitle != nil {
		next.MetaTitle = *req.MetaTitle
	}
	if req.MetaDescription != nil {
		next.MetaDescription = *req.MetaDescription
	}
	if req.Canonical != nil {
		next.Canonical = *req.Canonical
	}
	if req.PrimaryKeyword != nil {
		next.PrimaryKeyword = *req.PrimaryKeyword
	}
	if req.Keywords != nil {
		next.Keywords = req.Keywords
	}
	if req.Status != nil {
		next.Status = pagemap.Status(*req.Status)
	}
	if req.TemplateID != nil {
		next.TemplateID = nil
		if *req.TemplateID != "" {
			next.TemplateID = req.TemplateID
		}
	}
	return pagemap.NewPage(next)
}

func countDescendants(path string, pages []pagemap.Page) int {
	count := 0
	for i := range pages {
		if pages[i].Path != path && strings.HasPrefix(pages[i].Path, path) {
			count++
		}
	}
	return count
}

func (s *Service) Update(ctx context.Context, req UpdateRequest) (UpdateResponse, error) {
	var updated pagemap.Page
	err := s.uow.Do(ctx, func(c context.Context) error {
		current, getErr := s.pages.Get(c, req.ID)
		if getErr != nil {
			return getErr
		}
		next, applyErr := applyUpdate(&current, &req)
		if applyErr != nil {
			return applyErr
		}
		next.UpdatedAt = s.now()

		if next.Path != current.Path {
			siblings, listErr := s.pages.ListBySite(c, current.SiteID)
			if listErr != nil {
				return listErr
			}
			if descendants := countDescendants(current.Path, siblings); descendants > 0 {
				return errors.New(errors.Conflict, "a page with descendants cannot change its path").
					WithDetail("pageId", current.ID).WithDetail("descendants", descendants)
			}
			index := pagemap.NewIndex(siblings)
			if verdictErr := s.verdict(c, next, index, graph.Entity{}); verdictErr != nil {
				return verdictErr
			}
			next.ParentPageID = nil
			if parent, found := index.ByPath(pagemap.ParentPath(next.Path)); found && parent.ID != next.ID {
				next.ParentPageID = &parent.ID
			}
		}

		if updateErr := s.pages.Update(c, next); updateErr != nil {
			return updateErr
		}
		updated = next
		return nil
	})
	if err != nil {
		return UpdateResponse{}, err
	}
	if publishErr := s.changed(updated.SiteID); publishErr != nil {
		return UpdateResponse{}, publishErr
	}
	return UpdateResponse{Page: view(updated)}, nil
}

func (s *Service) Delete(ctx context.Context, req DeleteRequest) (DeleteResponse, error) {
	current, err := s.pages.Get(ctx, req.ID)
	if err != nil {
		return DeleteResponse{}, err
	}
	if req.OnSite {
		if current.WPID == nil {
			return DeleteResponse{}, errors.New(errors.Invalid,
				"the page is not on the site, so there is nothing to remove there").
				WithDetail("field", "wpId").WithDetail("pageId", current.ID)
		}
		if trashErr := s.preview.TrashItem(ctx, current.SiteID, *current.WPID, string(current.WPType)); trashErr != nil {
			return DeleteResponse{}, trashErr
		}
	}

	siteID := current.SiteID
	if dropErr := s.uow.Do(ctx, func(c context.Context) error {
		return s.pages.Delete(c, req.ID)
	}); dropErr != nil {
		return DeleteResponse{}, dropErr
	}
	if publishErr := s.changed(siteID); publishErr != nil {
		return DeleteResponse{}, publishErr
	}
	if publishErr := s.graphChanged(siteID); publishErr != nil {
		return DeleteResponse{}, publishErr
	}
	return DeleteResponse{}, nil
}

func (s *Service) Get(ctx context.Context, req GetRequest) (GetResponse, error) {
	page, err := s.pages.Get(ctx, req.ID)
	if err != nil {
		return GetResponse{}, err
	}
	links, err := s.links.ListForPage(ctx, page.ID)
	if err != nil {
		return GetResponse{}, err
	}
	return GetResponse{Page: view(page), Links: linkViews(links)}, nil
}

func pageSort(sort *dto.Sort) (key pagemap.Sort, desc bool, err error) {
	if sort == nil {
		return pagemap.SortCreatedAt, false, nil
	}
	key = pagemap.Sort(sort.Field)
	if !key.Valid() {
		return "", false, errors.New(errors.Invalid, "pages cannot be sorted by this field").WithDetail("field", "sort.field")
	}
	return key, sort.Desc, nil
}

func (s *Service) List(ctx context.Context, req ListRequest) (paging.List[Page], error) {
	if err := requireSite(req.SiteID); err != nil {
		return paging.List[Page]{}, err
	}
	q := pagemap.Query{SiteID: req.SiteID, Unmapped: req.Unmapped, PathPrefix: strings.ToLower(req.PathPrefix)}
	if req.Status != "" {
		status := pagemap.Status(req.Status)
		if !status.Valid() {
			return paging.List[Page]{}, errors.New(errors.Invalid, "page status is not recognized").WithDetail("field", "status")
		}
		q.Status = &status
	}
	if req.EntityID != "" {
		entityID := req.EntityID
		q.EntityID = &entityID
	}
	key, desc, err := pageSort(req.Sort)
	if err != nil {
		return paging.List[Page]{}, err
	}
	q.Sort, q.Desc = key, desc

	list, err := s.pages.List(ctx, q, application.PageRequest(req.ListRequest))
	if err != nil {
		return paging.List[Page]{}, err
	}
	return application.MapList(list, view), nil
}

func (s *Service) MapToEntity(ctx context.Context, req MapToEntityRequest) (MapToEntityResponse, error) {
	var updated pagemap.Page
	err := s.uow.Do(ctx, func(c context.Context) error {
		page, getErr := s.pages.Get(c, req.PageID)
		if getErr != nil {
			return getErr
		}
		page.EntityID = &req.EntityID
		entity, entityErr := s.entityFor(c, &page)
		if entityErr != nil {
			return entityErr
		}
		siblings, listErr := s.pages.ListBySite(c, page.SiteID)
		if listErr != nil {
			return listErr
		}
		if verdictErr := s.verdict(c, page, pagemap.NewIndex(siblings), entity); verdictErr != nil {
			return verdictErr
		}
		page.UpdatedAt = s.now()
		if updateErr := s.pages.Update(c, page); updateErr != nil {
			return updateErr
		}
		updated = page
		return nil
	})
	if err != nil {
		return MapToEntityResponse{}, err
	}
	if publishErr := s.changed(updated.SiteID); publishErr != nil {
		return MapToEntityResponse{}, publishErr
	}
	return MapToEntityResponse{Page: view(updated)}, nil
}

func (s *Service) Unmap(ctx context.Context, req UnmapRequest) (UnmapResponse, error) {
	var (
		updated          pagemap.Page
		touched          bool
		clearedCanonical bool
	)
	err := s.uow.Do(ctx, func(c context.Context) error {
		page, getErr := s.pages.Get(c, req.PageID)
		if getErr != nil {
			return getErr
		}
		updated = page
		if page.EntityID == nil {
			return nil
		}

		now := s.now()
		entity, entityErr := s.entities.Get(c, *page.EntityID)
		if entityErr != nil && !errors.IsCode(entityErr, errors.NotFound) {
			return entityErr
		}
		if entityErr == nil && entity.CanonicalPageID != nil && *entity.CanonicalPageID == page.ID {
			if clearErr := s.entities.SetCanonicalPage(c, entity.ID, nil, now); clearErr != nil {
				return clearErr
			}
			clearedCanonical = true
		}
		page.EntityID = nil
		page.UpdatedAt = now
		if updateErr := s.pages.Update(c, page); updateErr != nil {
			return updateErr
		}
		updated = page
		touched = true
		return nil
	})
	if err != nil {
		return UnmapResponse{}, err
	}
	if touched {
		if publishErr := s.changed(updated.SiteID); publishErr != nil {
			return UnmapResponse{}, publishErr
		}
	}
	if clearedCanonical {
		if publishErr := s.graphChanged(updated.SiteID); publishErr != nil {
			return UnmapResponse{}, publishErr
		}
	}
	return UnmapResponse{Page: view(updated)}, nil
}

func (s *Service) SetCanonical(ctx context.Context, req SetCanonicalRequest) (SetCanonicalResponse, error) {
	var updated pagemap.Page
	err := s.uow.Do(ctx, func(c context.Context) error {
		entity, entityErr := s.entities.Get(c, req.EntityID)
		if entityErr != nil {
			return entityErr
		}
		page, getErr := s.pages.Get(c, req.PageID)
		if getErr != nil {
			return getErr
		}
		if page.SiteID != entity.SiteID {
			return errors.New(errors.Invalid, "page and entity belong to different sites").WithDetail("pageId", page.ID).WithDetail("entityId", entity.ID)
		}

		now := s.now()
		switch {
		case page.EntityID == nil:
			page.EntityID = &entity.ID
			page.UpdatedAt = now
			if updateErr := s.pages.Update(c, page); updateErr != nil {
				return updateErr
			}
		case *page.EntityID != entity.ID:
			return errors.New(errors.Invalid, "page is mapped to another entity").WithDetail("pageId", page.ID).WithDetail("entityId", *page.EntityID)
		}
		if setErr := s.entities.SetCanonicalPage(c, entity.ID, &page.ID, now); setErr != nil {
			return setErr
		}
		updated = page
		return nil
	})
	if err != nil {
		return SetCanonicalResponse{}, err
	}
	if publishErr := s.graphChanged(updated.SiteID); publishErr != nil {
		return SetCanonicalResponse{}, publishErr
	}
	if publishErr := s.changed(updated.SiteID); publishErr != nil {
		return SetCanonicalResponse{}, publishErr
	}
	return SetCanonicalResponse{Page: view(updated)}, nil
}

func (s *Service) Tree(ctx context.Context, req TreeRequest) (TreeResponse, error) {
	if err := requireSite(req.SiteID); err != nil {
		return TreeResponse{}, err
	}
	if _, err := s.sites.Get(ctx, req.SiteID); err != nil {
		return TreeResponse{}, err
	}
	all, err := s.pages.ListBySite(ctx, req.SiteID)
	if err != nil {
		return TreeResponse{}, err
	}
	return TreeResponse{Roots: nodeViews(pagemap.BuildTree(all))}, nil
}
