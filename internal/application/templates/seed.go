package templates

import (
	"context"

	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const DefaultPolicyName = "Default"

func defaultPolicy() template.LinkPolicy {
	return template.LinkPolicy{
		Scope: template.ScopeGlobal,
		Name:  DefaultPolicyName,
		Rules: template.LinkRules{
			UpDepth:                    2,
			DownLinks:                  true,
			SiblingMinWeight:           0.5,
			MaxLinks:                   12,
			MaxPerTarget:               1,
			ParentLinkWithinParagraphs: 2,
			ChildrenSection:            true,
		},
		ForbidExternal: true,
		ForbidSelf:     true,
		AnchorStrategy: template.AnchorPreferUser,
	}
}

func (s *Service) EnsureSeeded(ctx context.Context) error {
	global := template.ScopeGlobal
	return s.uow.Do(ctx, func(c context.Context) error {
		seeds := template.Seed()
		for i := range seeds {
			seed := seeds[i]
			existing, err := s.templates.List(c, template.Query{Scope: &global, Name: seed.Name, Sort: template.SortCreatedAt}, paging.Request{Limit: 1})
			if err != nil {
				return err
			}
			if len(existing.Items) > 0 {
				if refreshErr := s.refresh(c, seed, existing.Items[0]); refreshErr != nil {
					return refreshErr
				}
				continue
			}
			now := s.now()
			seed.ID = id.New()
			seed.CreatedAt = now
			seed.UpdatedAt = now
			if validErr := seed.Validate(); validErr != nil {
				return validErr
			}
			if insertErr := s.templates.Insert(c, seed); insertErr != nil {
				return insertErr
			}
		}

		policies, err := s.policies.List(c, template.PolicyQuery{Scope: &global, Name: DefaultPolicyName, Sort: template.SortCreatedAt}, paging.Request{Limit: 1})
		if err != nil {
			return err
		}
		if len(policies.Items) > 0 {
			return nil
		}
		policy := defaultPolicy()
		now := s.now()
		policy.ID = id.New()
		policy.CreatedAt = now
		policy.UpdatedAt = now
		if validErr := policy.Validate(); validErr != nil {
			return validErr
		}
		return s.policies.Insert(c, policy)
	})
}

func (s *Service) refresh(ctx context.Context, seed, stored template.Template) error {
	if !template.Supersedes(seed, stored) {
		return nil
	}
	next := stored
	next.Spec = seed.Spec
	fits, err := s.overridesFit(ctx, &next)
	if err != nil || !fits {
		return err
	}
	next.Version = stored.Version + 1
	next.UpdatedAt = s.now()
	if validErr := next.Validate(); validErr != nil {
		return validErr
	}
	return s.templates.Update(ctx, next)
}

func (s *Service) overridesFit(ctx context.Context, next *template.Template) (bool, error) {
	overrides, err := s.templates.ListOverrides(ctx, next.ID)
	if err != nil {
		return false, err
	}
	for i := range overrides {
		siteOverride, pageOverride, chainErr := s.chain(ctx, next, &overrides[i])
		if errors.IsCode(chainErr, errors.NotFound) {
			continue
		}
		if chainErr != nil {
			return false, chainErr
		}
		if _, resolveErr := template.Resolve(next.Spec, siteOverride, pageOverride); resolveErr != nil {
			return false, nil
		}
	}
	return true, nil
}
