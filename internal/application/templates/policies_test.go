package templates_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestPolicyLifecycleAndEffectivePolicy(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	if err := h.service.EnsureSeeded(t.Context()); err != nil {
		t.Fatalf("EnsureSeeded: %v", err)
	}
	h.recorder.Reset()

	seededPolicies, err := h.service.ListPolicies(t.Context(), templates.ListPoliciesRequest{Scope: "global"})
	if err != nil || len(seededPolicies.Items) != 1 || seededPolicies.Items[0].Name != templates.DefaultPolicyName {
		t.Fatalf("seeded policies = %+v, %v", seededPolicies.Items, err)
	}

	effective, err := h.service.GetEffectivePolicy(t.Context(), templates.GetEffectivePolicyRequest{SiteID: h.siteID})
	if err != nil || effective.Policy.Name != templates.DefaultPolicyName || effective.Policy.Scope != "global" {
		t.Fatalf("GetEffectivePolicy fallback = %+v, %v", effective.Policy, err)
	}

	rules := template.LinkRules{UpDepth: 1, DownLinks: true, SiblingMinWeight: 0.8, MaxLinks: 6, MaxPerTarget: 1, ParentLinkWithinParagraphs: 1}
	created, err := h.service.CreatePolicy(t.Context(), templates.CreatePolicyRequest{SiteID: &h.siteID, Name: "Strict", Rules: rules, ForbidExternal: true, ForbidSelf: true})
	if err != nil {
		t.Fatalf("CreatePolicy: %v", err)
	}
	if created.Policy.Scope != "site" || created.Policy.AnchorStrategy != "prefer_user" || created.Policy.Rules.MaxLinks != 6 {
		t.Errorf("CreatePolicy = %+v", created.Policy)
	}
	h.wantEvents(t, 1)

	owner, err := sqlite.NewSiteRepo(h.store).Get(t.Context(), h.siteID)
	if err != nil {
		t.Fatalf("Get site: %v", err)
	}
	owner.Defaults.LinkPolicyID = &created.Policy.ID
	if err = sqlite.NewSiteRepo(h.store).Update(t.Context(), owner); err != nil {
		t.Fatalf("set the site policy: %v", err)
	}
	effective, err = h.service.GetEffectivePolicy(t.Context(), templates.GetEffectivePolicyRequest{SiteID: h.siteID})
	if err != nil || effective.Policy.ID != created.Policy.ID {
		t.Errorf("GetEffectivePolicy via the site default = %+v, %v", effective.Policy, err)
	}

	updated, err := h.service.UpdatePolicy(t.Context(), templates.UpdatePolicyRequest{ID: created.Policy.ID, Name: ptr("Stricter"), ForbidExternal: ptr(false), AnchorStrategy: ptr("rotate")})
	if err != nil || updated.Policy.Name != "Stricter" || updated.Policy.ForbidExternal || updated.Policy.AnchorStrategy != "rotate" || updated.Policy.Rules.MaxLinks != 6 {
		t.Errorf("UpdatePolicy = %+v, %v", updated.Policy, err)
	}
	h.wantEvents(t, 1)

	if _, err = h.service.UpdatePolicy(t.Context(), templates.UpdatePolicyRequest{ID: created.Policy.ID, Rules: &template.LinkRules{MaxLinks: -1}}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("bad rules code = %q, want INVALID", errors.CodeOf(err))
	}
	if _, err = h.service.UpdatePolicy(t.Context(), templates.UpdatePolicyRequest{ID: "missing", Name: ptr("x")}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("UpdatePolicy missing code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if _, err = h.service.CreatePolicy(t.Context(), templates.CreatePolicyRequest{Name: "default", Rules: rules}); !errors.IsCode(err, errors.Conflict) {
		t.Errorf("duplicate global name code = %q, want CONFLICT", errors.CodeOf(err))
	}
	if _, err = h.service.CreatePolicy(t.Context(), templates.CreatePolicyRequest{Scope: "site", Name: "x", Rules: rules}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("site scope without a site code = %q, want INVALID", errors.CodeOf(err))
	}
	if _, err = h.service.CreatePolicy(t.Context(), templates.CreatePolicyRequest{Scope: "galaxy", Name: "x", Rules: rules}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("unknown scope code = %q, want INVALID", errors.CodeOf(err))
	}
	if _, err = h.service.CreatePolicy(t.Context(), templates.CreatePolicyRequest{Name: "x", Rules: rules, AnchorStrategy: "random"}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("bad strategy code = %q, want INVALID", errors.CodeOf(err))
	}

	got, err := h.service.GetPolicy(t.Context(), templates.GetPolicyRequest{ID: created.Policy.ID})
	if err != nil || got.Policy.Name != "Stricter" {
		t.Errorf("GetPolicy = %+v, %v", got.Policy, err)
	}
	if _, err = h.service.GetPolicy(t.Context(), templates.GetPolicyRequest{ID: "missing"}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("GetPolicy missing code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	bySite, err := h.service.ListPolicies(t.Context(), templates.ListPoliciesRequest{SiteID: h.siteID})
	if err != nil || len(bySite.Items) != 1 {
		t.Errorf("ListPolicies by site = %+v, %v", bySite, err)
	}
	if _, err = h.service.ListPolicies(t.Context(), templates.ListPoliciesRequest{Scope: "galaxy"}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("bad scope code = %q, want INVALID", errors.CodeOf(err))
	}
	if _, err = h.service.ListPolicies(t.Context(), templates.ListPoliciesRequest{ListRequest: dto.ListRequest{Sort: &dto.Sort{Field: "rules"}}}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("bad sort code = %q, want INVALID", errors.CodeOf(err))
	}

	if _, err = h.service.DeletePolicy(t.Context(), templates.DeletePolicyRequest{ID: created.Policy.ID}); err != nil {
		t.Fatalf("DeletePolicy: %v", err)
	}
	h.wantEvents(t, 1)
	effective, err = h.service.GetEffectivePolicy(t.Context(), templates.GetEffectivePolicyRequest{SiteID: h.siteID})
	if err != nil || effective.Policy.Name != templates.DefaultPolicyName {
		t.Errorf("after deleting the site policy the default applies again: %+v, %v", effective.Policy, err)
	}
	if _, err = h.service.DeletePolicy(t.Context(), templates.DeletePolicyRequest{ID: created.Policy.ID}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("DeletePolicy twice code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if _, err = h.service.GetEffectivePolicy(t.Context(), templates.GetEffectivePolicyRequest{SiteID: "missing"}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("unknown site code = %q, want NOT_FOUND", errors.CodeOf(err))
	}

	other := sqlitetest.Site(t, h.store, "blog")
	if _, err = h.service.GetEffectivePolicy(t.Context(), templates.GetEffectivePolicyRequest{SiteID: other.ID}); err != nil {
		t.Errorf("every site falls back to the default policy: %v", err)
	}
}

func TestEffectiveRulesTakeTheTemplateOverThePolicy(t *testing.T) {
	t.Parallel()

	stored := template.LinkRules{
		UpDepth: 2, DownLinks: true, SiblingMinWeight: 0.5, MaxLinks: 12, MaxPerTarget: 1,
		ParentLinkWithinParagraphs: 2, ChildrenSection: true,
	}
	opinionated := template.LinkRules{
		UpDepth: 1, DownLinks: false, SiblingMinWeight: 0.7, MaxLinks: 8, MaxPerTarget: 2,
		ParentLinkWithinParagraphs: 1,
	}

	cases := []struct {
		name   string
		policy template.LinkRules
		spec   template.LinkRules
		want   template.LinkRules
	}{
		{name: "a template that says nothing inherits the policy", policy: stored, spec: template.LinkRules{}, want: stored},
		{name: "a template that carries rules replaces them whole", policy: stored, spec: opinionated, want: opinionated},
		{
			name:   "a template that only turns down links off still replaces them whole",
			policy: stored, spec: template.LinkRules{UpDepth: 1}, want: template.LinkRules{UpDepth: 1},
		},
		{name: "neither says anything", policy: template.LinkRules{}, spec: template.LinkRules{}, want: template.LinkRules{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := templates.EffectiveRules(tc.policy, tc.spec); got != tc.want {
				t.Errorf("EffectiveRules = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestGetEffectivePolicyWithoutASeededDefault(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	if _, err := h.service.GetEffectivePolicy(t.Context(), templates.GetEffectivePolicyRequest{SiteID: h.siteID}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("no policy at all code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
}
