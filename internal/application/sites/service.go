package sites

import (
	"context"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/application"
	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

type siteStore interface {
	Insert(ctx context.Context, s site.Site) error
	Update(ctx context.Context, s site.Site) error
	Delete(ctx context.Context, id string) error
	Get(ctx context.Context, id string) (site.Site, error)
	List(ctx context.Context, q site.Query, page paging.Request) (paging.List[site.Site], error)
}

type secretStore interface {
	Put(ctx context.Context, ref, value string) error
	Delete(ctx context.Context, ref string) error
}

type unitOfWork interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}

type Service struct {
	store     siteStore
	secrets   secretStore
	uow       unitOfWork
	publisher application.Publisher
	clock     clock.Clock
}

func New(store siteStore, secrets secretStore, uow unitOfWork, publisher application.Publisher, clk clock.Clock) *Service {
	return &Service{store: store, secrets: secrets, uow: uow, publisher: publisher, clock: clk}
}

func (s *Service) now() time.Time {
	return s.clock.Now().UTC().Truncate(time.Second)
}

func (s *Service) changed(siteID string) error {
	return s.publisher.Publish(events.SitesChanged, events.SitesChangedPayload{SiteID: siteID})
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (CreateResponse, error) {
	baseURL, err := site.NormalizeBaseURL(req.BaseURL, req.AllowInsecure)
	if err != nil {
		return CreateResponse{}, err
	}

	now := s.now()
	record := site.Site{
		ID:            id.New(),
		Name:          strings.TrimSpace(req.Name),
		BaseURL:       baseURL,
		Username:      strings.TrimSpace(req.Username),
		Status:        site.StatusActive,
		AllowInsecure: req.AllowInsecure,
		Plugin:        site.PluginState{Capabilities: []string{}},
		Defaults:      site.Defaults{ModelProfiles: map[llm.Role]llm.ModelRef{}},
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	record.SecretRef = site.SecretRef(record.ID)
	if validateErr := record.Validate(); validateErr != nil {
		return CreateResponse{}, validateErr
	}

	err = s.uow.Do(ctx, func(c context.Context) error {
		if insertErr := s.store.Insert(c, record); insertErr != nil {
			return insertErr
		}
		if req.Password == "" {
			return nil
		}
		return s.secrets.Put(c, record.SecretRef, req.Password)
	})
	if err != nil {
		return CreateResponse{}, err
	}
	if publishErr := s.changed(record.ID); publishErr != nil {
		return CreateResponse{}, publishErr
	}
	return CreateResponse{Site: view(record)}, nil
}

func (s *Service) Update(ctx context.Context, req UpdateRequest) (UpdateResponse, error) {
	var updated site.Site
	err := s.uow.Do(ctx, func(c context.Context) error {
		current, getErr := s.store.Get(c, req.ID)
		if getErr != nil {
			return getErr
		}
		next, applyErr := apply(&current, &req)
		if applyErr != nil {
			return applyErr
		}
		next.UpdatedAt = s.now()
		if validateErr := next.Validate(); validateErr != nil {
			return validateErr
		}
		if updateErr := s.store.Update(c, next); updateErr != nil {
			return updateErr
		}
		if req.Password != nil {
			if rotateErr := s.rotate(c, next.SecretRef, *req.Password); rotateErr != nil {
				return rotateErr
			}
		}
		updated = next
		return nil
	})
	if err != nil {
		return UpdateResponse{}, err
	}
	if publishErr := s.changed(updated.ID); publishErr != nil {
		return UpdateResponse{}, publishErr
	}
	return UpdateResponse{Site: view(updated)}, nil
}

func apply(current *site.Site, req *UpdateRequest) (site.Site, error) {
	next := *current
	if req.Name != nil {
		next.Name = strings.TrimSpace(*req.Name)
	}
	if req.Username != nil {
		next.Username = strings.TrimSpace(*req.Username)
	}
	if req.AllowInsecure != nil {
		next.AllowInsecure = *req.AllowInsecure
	}
	if req.BaseURL != nil {
		baseURL, err := site.NormalizeBaseURL(*req.BaseURL, next.AllowInsecure)
		if err != nil {
			return site.Site{}, err
		}
		next.BaseURL = baseURL
	}
	if req.Status != nil {
		next.Status = site.Status(*req.Status)
	}
	if req.Defaults != nil {
		next.Defaults = site.Defaults{
			TemplateID:    req.Defaults.TemplateID,
			LinkPolicyID:  req.Defaults.LinkPolicyID,
			ModelProfiles: profilesOf(req.Defaults.ModelProfiles),
		}
	}
	return next, nil
}

func (s *Service) rotate(ctx context.Context, ref, password string) error {
	if password == "" {
		return ignoreNotFound(s.secrets.Delete(ctx, ref))
	}
	return s.secrets.Put(ctx, ref, password)
}

func ignoreNotFound(err error) error {
	if errors.IsCode(err, errors.NotFound) {
		return nil
	}
	return err
}

func (s *Service) Delete(ctx context.Context, req DeleteRequest) (DeleteResponse, error) {
	err := s.uow.Do(ctx, func(c context.Context) error {
		current, getErr := s.store.Get(c, req.ID)
		if getErr != nil {
			return getErr
		}
		if deleteErr := s.store.Delete(c, req.ID); deleteErr != nil {
			return deleteErr
		}
		return ignoreNotFound(s.secrets.Delete(c, current.SecretRef))
	})
	if err != nil {
		return DeleteResponse{}, err
	}
	if publishErr := s.changed(req.ID); publishErr != nil {
		return DeleteResponse{}, publishErr
	}
	return DeleteResponse{}, nil
}

func (s *Service) Get(ctx context.Context, req GetRequest) (GetResponse, error) {
	record, err := s.store.Get(ctx, req.ID)
	if err != nil {
		return GetResponse{}, err
	}
	return GetResponse{Site: view(record)}, nil
}

func (s *Service) List(ctx context.Context, req ListRequest) (paging.List[Site], error) {
	q := site.Query{Sort: site.SortCreatedAt}
	if req.Status != "" {
		status := site.Status(req.Status)
		if !status.Valid() {
			return paging.List[Site]{}, errors.New(errors.Invalid, "site status is not recognized").WithDetail("field", "status")
		}
		q.Status = &status
	}
	if req.Sort != nil {
		q.Sort = site.Sort(req.Sort.Field)
		if !q.Sort.Valid() {
			return paging.List[Site]{}, errors.New(errors.Invalid, "sites cannot be sorted by this field").WithDetail("field", "sort.field")
		}
		q.Desc = req.Sort.Desc
	}

	list, err := s.store.List(ctx, q, application.PageRequest(req.ListRequest))
	if err != nil {
		return paging.List[Site]{}, err
	}
	return application.MapList(list, view), nil
}
