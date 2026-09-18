package site_test

import (
	stderrors "errors"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestSecretRef(t *testing.T) {
	t.Parallel()

	if got := site.SecretRef("9f0d0d22-6f4f-4c1a-9c07-5b6c1f6bd9a1"); got != "site:9f0d0d22-6f4f-4c1a-9c07-5b6c1f6bd9a1:wp_password" {
		t.Fatalf("SecretRef = %q", got)
	}
}

func TestNormalizeBaseURL(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		raw      string
		insecure bool
		want     string
		code     errors.Code
	}{
		{name: "https is kept", raw: "https://Example.com", want: "https://example.com"},
		{name: "trailing slash and query are dropped", raw: " https://example.com/blog/?utm=1#x ", want: "https://example.com/blog"},
		{name: "path is kept", raw: "https://example.com/wp", want: "https://example.com/wp"},
		{name: "http refused by default", raw: "http://example.com", code: errors.Invalid},
		{name: "http allowed when insecure", raw: "http://example.com", insecure: true, want: "http://example.com"},
		{name: "empty", raw: "  ", code: errors.Invalid},
		{name: "no scheme", raw: "example.com", code: errors.Invalid},
		{name: "ftp", raw: "ftp://example.com", code: errors.Invalid},
		{name: "no host", raw: "https://", code: errors.Invalid},
		{name: "credentials in url", raw: "https://user:pw@example.com", code: errors.Invalid},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := site.NormalizeBaseURL(tc.raw, tc.insecure)
			if tc.code != "" {
				if !errors.IsCode(err, tc.code) {
					t.Fatalf("code = %q, want %q (err %v)", errors.CodeOf(err), tc.code, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("NormalizeBaseURL: %v", err)
			}
			if got != tc.want {
				t.Errorf("NormalizeBaseURL = %q, want %q", got, tc.want)
			}
		})
	}
}

func validSite() site.Site {
	at := time.Date(2026, time.September, 18, 9, 0, 0, 0, time.UTC)
	return site.Site{
		ID:        "9f0d0d22-6f4f-4c1a-9c07-5b6c1f6bd9a1",
		Name:      "Shop",
		BaseURL:   "https://shop.example.com",
		Username:  "editor",
		SecretRef: site.SecretRef("9f0d0d22-6f4f-4c1a-9c07-5b6c1f6bd9a1"),
		Status:    site.StatusActive,
		Defaults:  site.Defaults{ModelProfiles: map[llm.Role]llm.ModelRef{llm.RoleWriter: {Provider: "openai", Model: "gpt"}}},
		CreatedAt: at,
		UpdatedAt: at,
	}
}

func TestSiteValidate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		mutate func(*site.Site)
		field  string
	}{
		{name: "valid", mutate: func(*site.Site) {}},
		{name: "no id", mutate: func(s *site.Site) { s.ID = "" }, field: "id"},
		{name: "blank name", mutate: func(s *site.Site) { s.Name = "  " }, field: "name"},
		{name: "http without allow", mutate: func(s *site.Site) { s.BaseURL = "http://shop.example.com" }, field: "baseUrl"},
		{name: "unknown status", mutate: func(s *site.Site) { s.Status = "sleeping" }, field: "status"},
		{name: "foreign secret ref", mutate: func(s *site.Site) { s.SecretRef = "site:other:wp_password" }, field: "secretRef"},
		{name: "unknown role", mutate: func(s *site.Site) {
			s.Defaults.ModelProfiles = map[llm.Role]llm.ModelRef{"painter": {Provider: "a", Model: "b"}}
		}, field: "defaults.modelProfiles.painter"},
		{name: "incomplete ref", mutate: func(s *site.Site) {
			s.Defaults.ModelProfiles = map[llm.Role]llm.ModelRef{llm.RoleChat: {Provider: "a"}}
		}, field: "defaults.modelProfiles.chat"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s := validSite()
			tc.mutate(&s)
			err := s.Validate()
			if tc.field == "" {
				if err != nil {
					t.Fatalf("Validate: %v", err)
				}
				return
			}
			if !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("code = %q, want INVALID (err %v)", errors.CodeOf(err), err)
			}
			var kernel *errors.Error
			if !stderrors.As(err, &kernel) || kernel.Details["field"] != tc.field {
				t.Errorf("details = %v, want field %q", err, tc.field)
			}
		})
	}
}

func TestStatusAndSort(t *testing.T) {
	t.Parallel()

	for _, status := range []site.Status{site.StatusActive, site.StatusPaused, site.StatusError} {
		if !status.Valid() {
			t.Errorf("%q must be valid", status)
		}
	}
	if site.Status("x").Valid() {
		t.Error("unknown status must be invalid")
	}
	if !site.SortCreatedAt.Valid() || !site.SortName.Valid() || site.Sort("age").Valid() {
		t.Error("sort validity is wrong")
	}
}
