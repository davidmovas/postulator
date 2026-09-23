package templates

import (
	"context"

	"github.com/davidmovas/postulator/internal/domain/template"
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
