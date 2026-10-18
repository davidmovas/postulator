package field_test

import (
	stderrors "errors"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func refusal(t *testing.T, err error) (name, reason string) {
	t.Helper()
	var kerr *errors.Error
	if !stderrors.As(err, &kerr) || kerr.Code != errors.Invalid {
		t.Fatalf("err = %v, want INVALID", err)
	}
	if held, ok := kerr.Details["field"].(string); ok {
		name = held
	}
	if held, ok := kerr.Details["reason"].(string); ok {
		reason = held
	}
	return name, reason
}

func TestCheckScopeRefusesWhatNoScopeMayName(t *testing.T) {
	t.Parallel()

	refused := map[string]struct {
		key    field.Key
		reason string
	}{
		"the slug":                       {field.Slug, field.ReasonDenied},
		"the status":                     {field.Status, field.ReasonDenied},
		"the parent":                     {field.Parent, field.ReasonDenied},
		"the price":                      {field.Price, field.ReasonDenied},
		"the regular price":              {field.RegularPrice, field.ReasonDenied},
		"the sale price":                 {field.SalePrice, field.ReasonDenied},
		"the SKU":                        {field.SKU, field.ReasonDenied},
		"the stock status":               {field.StockStatus, field.ReasonDenied},
		"the stock quantity":             {field.StockQuantity, field.ReasonDenied},
		"stock management":               {field.ManageStock, field.ReasonDenied},
		"the attributes":                 {field.Attributes, field.ReasonDenied},
		"a variation's facts":            {field.Instance(field.Variation, "42"), field.ReasonDenied},
		"the terms":                      {field.Terms, field.ReasonDenied},
		"a protected meta key":           {field.Instance(field.Meta, "_price"), field.ReasonDenied},
		"a store's sales counter":        {field.Instance(field.Meta, "total_sales"), field.ReasonDenied},
		"an SEO plugin's own key":        {field.Instance(field.Meta, "rank_math_focus_keyword"), field.ReasonDenied},
		"the plugin's own key":           {field.Instance(field.Meta, "_postulator_seo_title"), field.ReasonDenied},
		"a meta field without a key":     {field.Meta, field.ReasonUnknown},
		"a meta key with a comma":        {field.Instance(field.Meta, "spec,sheet"), field.ReasonUnknown},
		"a meta key with a space":        {field.Instance(field.Meta, "spec sheet"), field.ReasonUnknown},
		"a field the catalog lacks":      {field.Key("rating"), field.ReasonUnknown},
		"one variation's description":    {field.Instance(field.VariationDescription, "42"), field.ReasonUnknown},
		"the canonical, no written text": {field.SEOCanonical, field.ReasonUnknown},
	}
	for name, tc := range refused {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, reason := refusal(t, field.CheckScope(tc.key))
			if got != "field" || reason != tc.reason {
				t.Errorf("CheckScope(%s) = field %q reason %q, want field and %s", tc.key, got, reason, tc.reason)
			}
		})
	}

	allowed := []field.Key{
		field.Title, field.Excerpt, field.Body, field.SEOTitle, field.SEODescription, field.ProductDescription,
		field.ProductShortDescription, field.VariationDescription, field.ImageAlt, field.ImageTitle, field.Upsells,
		field.CrossSells, field.Instance(field.Meta, "spec_sheet"), field.Instance(field.Meta, "wpcf-dosage"),
	}
	for _, key := range allowed {
		if err := field.CheckScope(key); err != nil {
			t.Errorf("CheckScope(%s) = %v, want allowed", key, err)
		}
	}
}

func TestAScopeValidatesItsCustomFieldAndItsLimits(t *testing.T) {
	t.Parallel()

	policy := field.LinkPolicy{Links: true, Rules: field.LinkRules{UpDepth: 1, MaxLinks: 3, MaxPerTarget: 1}, AnchorStrategy: field.AnchorPreferUser}
	valid := field.Scope{PageTypeID: "type-1", Key: field.Body, Enabled: true, AIWrites: true, Links: policy}
	custom := field.Scope{
		PageTypeID: "type-1", Key: field.Instance(field.Meta, "spec_sheet"), Enabled: true, Label: "Spec sheet",
		Provider: field.ProviderPlugin, Links: field.NoLinks(),
	}

	cases := map[string]struct {
		scope field.Scope
		field string
	}{
		"a body scope":                         {scope: valid},
		"a custom field with its label":        {scope: custom},
		"a scope with no type":                 {scope: func() field.Scope { s := valid; s.PageTypeID = ""; return s }(), field: "pageTypeId"},
		"a custom field with no label":         {scope: func() field.Scope { s := custom; s.Label = " "; return s }(), field: "label"},
		"a custom field with no provider":      {scope: func() field.Scope { s := custom; s.Provider = ""; return s }(), field: "provider"},
		"a provider on a catalog field":        {scope: func() field.Scope { s := valid; s.Provider = field.ProviderStore; return s }(), field: "provider"},
		"a negative minimum":                   {scope: func() field.Scope { s := valid; s.MinLength = -1; return s }(), field: "minLength"},
		"a maximum below the minimum":          {scope: func() field.Scope { s := valid; s.MinLength, s.MaxLength = 50, 10; return s }(), field: "maxLength"},
		"a policy with an unknown anchor":      {scope: func() field.Scope { s := valid; s.Links.AnchorStrategy = "loudest"; return s }(), field: "links.anchorStrategy"},
		"a policy weighing siblings above one": {scope: func() field.Scope { s := valid; s.Links.Rules.SiblingMinWeight = 1.5; return s }(), field: "links.rules.siblingMinWeight"},
		"a denied key":                         {scope: func() field.Scope { s := valid; s.Key = field.SKU; return s }(), field: "field"},
		"a maximum of zero means no maximum":   {scope: func() field.Scope { s := valid; s.MinLength, s.MaxLength = 50, 0; return s }()},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := tc.scope.Validate()
			if tc.field == "" {
				if err != nil {
					t.Fatalf("Validate = %v, want valid", err)
				}
				return
			}
			if got, _ := refusal(t, err); got != tc.field {
				t.Errorf("Validate refused %q, want %q", got, tc.field)
			}
		})
	}
}

func TestAStoredPolicyMustSayWhetherItCarriesLinks(t *testing.T) {
	t.Parallel()

	if _, err := field.ParsePolicy([]byte(`{}`)); err == nil {
		t.Error("a policy without links was read as no links")
	}
	if _, err := field.ParsePolicy([]byte(`{"rules":{"maxLinks":3}}`)); err == nil {
		t.Error("a policy without links was read as no links")
	}
	got, err := field.ParsePolicy([]byte(`{"links":true,"rules":{"upDepth":1,"downLinks":true,"siblingMinWeight":0.5,"maxLinks":20,"maxPerTarget":1,"parentLinkWithinParagraphs":2,"childrenSection":true},"forbidExternal":true,"forbidSelf":true,"anchorStrategy":"prefer_user"}`))
	if err != nil {
		t.Fatalf("ParsePolicy: %v", err)
	}
	want := field.LinkPolicy{
		Links:          true,
		Rules:          field.LinkRules{UpDepth: 1, DownLinks: true, SiblingMinWeight: 0.5, MaxLinks: 20, MaxPerTarget: 1, ParentLinkWithinParagraphs: 2, ChildrenSection: true},
		ForbidExternal: true, ForbidSelf: true, AnchorStrategy: field.AnchorPreferUser,
	}
	if got != want {
		t.Errorf("ParsePolicy = %+v, want %+v", got, want)
	}
	if none, err := field.ParsePolicy([]byte(`{"links":false}`)); err != nil || none.Links || none != field.NoLinks() {
		t.Errorf("ParsePolicy(no links) = %+v %v, want NoLinks", none, err)
	}
}
