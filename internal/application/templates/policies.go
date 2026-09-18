package templates

import (
	"context"
	"strings"

	"github.com/davidmovas/postulator/internal/application"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

func policyView(p template.LinkPolicy) LinkPolicy {
	return LinkPolicy{
		ID:             p.ID,
		Scope:          string(p.Scope),
		SiteID:         p.SiteID,
		Name:           p.Name,
		Rules:          p.Rules,
		ForbidExternal: p.ForbidExternal,
		ForbidSelf:     p.ForbidSelf,
		AnchorStrategy: string(p.AnchorStrategy),
		CreatedAt:      dto.NewTime(p.CreatedAt),
		UpdatedAt:      dto.NewTime(p.UpdatedAt),
	}
}

func (s *Service) CreatePolicy(ctx context.Context, req CreatePolicyRequest) (CreatePolicyResponse, error) {
	scope, err := scopeOf(req.Scope, req.SiteID)
	if err != nil {
		return CreatePolicyResponse{}, err
	}
	if targetErr := s.requireTarget(ctx, scope, req.SiteID); targetErr != nil {
		return CreatePolicyResponse{}, targetErr
	}

	strategy := template.AnchorStrategy(req.AnchorStrategy)
	if req.AnchorStrategy == "" {
		strategy = template.AnchorPreferUser
	}
	now := s.now()
	record := template.LinkPolicy{
		ID:             id.New(),
		Scope:          scope,
		SiteID:         req.SiteID,
		Name:           strings.TrimSpace(req.Name),
		Rules:          req.Rules,
		ForbidExternal: req.ForbidExternal,
		ForbidSelf:     req.ForbidSelf,
		AnchorStrategy: strategy,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if scope == template.ScopeGlobal {
		record.SiteID = nil
	}
	if validErr := record.Validate(); validErr != nil {
		return CreatePolicyResponse{}, validErr
	}
	if doErr := s.uow.Do(ctx, func(c context.Context) error { return s.policies.Insert(c, record) }); doErr != nil {
		return CreatePolicyResponse{}, doErr
	}
	if publishErr := s.changed(); publishErr != nil {
		return CreatePolicyResponse{}, publishErr
	}
	return CreatePolicyResponse{Policy: policyView(record)}, nil
}

func (s *Service) UpdatePolicy(ctx context.Context, req UpdatePolicyRequest) (UpdatePolicyResponse, error) {
	var updated template.LinkPolicy
	err := s.uow.Do(ctx, func(c context.Context) error {
		current, getErr := s.policies.Get(c, req.ID)
		if getErr != nil {
			return getErr
		}
		next := current
		if req.Name != nil {
			next.Name = strings.TrimSpace(*req.Name)
		}
		if req.Rules != nil {
			next.Rules = *req.Rules
		}
		if req.ForbidExternal != nil {
			next.ForbidExternal = *req.ForbidExternal
		}
		if req.ForbidSelf != nil {
			next.ForbidSelf = *req.ForbidSelf
		}
		if req.AnchorStrategy != nil {
			next.AnchorStrategy = template.AnchorStrategy(*req.AnchorStrategy)
		}
		next.UpdatedAt = s.now()
		if validErr := next.Validate(); validErr != nil {
			return validErr
		}
		if updateErr := s.policies.Update(c, next); updateErr != nil {
			return updateErr
		}
		updated = next
		return nil
	})
	if err != nil {
		return UpdatePolicyResponse{}, err
	}
	if publishErr := s.changed(); publishErr != nil {
		return UpdatePolicyResponse{}, publishErr
	}
	return UpdatePolicyResponse{Policy: policyView(updated)}, nil
}

func (s *Service) DeletePolicy(ctx context.Context, req DeletePolicyRequest) (DeletePolicyResponse, error) {
	if err := s.uow.Do(ctx, func(c context.Context) error { return s.policies.Delete(c, req.ID) }); err != nil {
		return DeletePolicyResponse{}, err
	}
	if err := s.changed(); err != nil {
		return DeletePolicyResponse{}, err
	}
	return DeletePolicyResponse{}, nil
}

func (s *Service) GetPolicy(ctx context.Context, req GetPolicyRequest) (GetPolicyResponse, error) {
	record, err := s.policies.Get(ctx, req.ID)
	if err != nil {
		return GetPolicyResponse{}, err
	}
	return GetPolicyResponse{Policy: policyView(record)}, nil
}

func (s *Service) ListPolicies(ctx context.Context, req ListPoliciesRequest) (paging.List[LinkPolicy], error) {
	scope, err := scopeFilter(req.Scope)
	if err != nil {
		return paging.List[LinkPolicy]{}, err
	}
	key, desc, err := sortOf(req.Sort)
	if err != nil {
		return paging.List[LinkPolicy]{}, err
	}
	q := template.PolicyQuery{Scope: scope, SiteID: siteFilter(req.SiteID), Sort: key, Desc: desc}

	list, err := s.policies.List(ctx, q, application.PageRequest(req.ListRequest))
	if err != nil {
		return paging.List[LinkPolicy]{}, err
	}
	return application.MapList(list, policyView), nil
}

func (s *Service) GetEffectivePolicy(ctx context.Context, req GetEffectivePolicyRequest) (GetEffectivePolicyResponse, error) {
	owner, err := s.sites.Get(ctx, req.SiteID)
	if err != nil {
		return GetEffectivePolicyResponse{}, err
	}
	if owner.Defaults.LinkPolicyID != nil {
		record, getErr := s.policies.Get(ctx, *owner.Defaults.LinkPolicyID)
		if getErr != nil {
			return GetEffectivePolicyResponse{}, getErr
		}
		return GetEffectivePolicyResponse{Policy: policyView(record)}, nil
	}

	global := template.ScopeGlobal
	list, err := s.policies.List(ctx, template.PolicyQuery{Scope: &global, Name: DefaultPolicyName, Sort: template.SortCreatedAt}, paging.Request{Limit: 1})
	if err != nil {
		return GetEffectivePolicyResponse{}, err
	}
	if len(list.Items) == 0 {
		return GetEffectivePolicyResponse{}, errors.New(errors.NotFound, "no link policy applies to the site").WithDetail("siteId", req.SiteID)
	}
	return GetEffectivePolicyResponse{Policy: policyView(list.Items[0])}, nil
}
