package pages_test

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type issued struct {
	siteID string
	wpType string
	wpID   int64
}

type recordingIssuer struct {
	answer   pages.IssuedPreview
	err      error
	trashErr error
	calls    []issued
	trashed  []issued
}

func (r *recordingIssuer) IssuePreview(_ context.Context, siteID string, wpID int64) (pages.IssuedPreview, error) {
	r.calls = append(r.calls, issued{siteID: siteID, wpID: wpID})
	return r.answer, r.err
}

func (r *recordingIssuer) TrashItem(_ context.Context, siteID string, wpID int64, wpType string) error {
	r.trashed = append(r.trashed, issued{siteID: siteID, wpID: wpID, wpType: wpType})
	return r.trashErr
}

func (h harness) placed(t *testing.T, path string, status pagemap.Status, wpID int64) pages.Page {
	t.Helper()

	created := h.page(t, path, nil)
	repo := sqlite.NewPageRepo(h.store)
	stored, err := repo.Get(t.Context(), created.ID)
	if err != nil {
		t.Fatalf("Get %s: %v", path, err)
	}
	stored.Status = status
	if wpID != 0 {
		stored.WPID = &wpID
	}
	if err = repo.Update(t.Context(), stored); err != nil {
		t.Fatalf("Update %s: %v", path, err)
	}
	return created
}

func fieldOf(err error) string {
	var kernel *errors.Error
	if !stderrors.As(err, &kernel) {
		return ""
	}
	field, ok := kernel.Details["field"].(string)
	if !ok {
		return ""
	}
	return field
}

func TestPreviewLinkRefusesWhatCannotBeSeen(t *testing.T) {
	t.Parallel()

	issuer := &recordingIssuer{}
	h := newPreviewHarness(t, issuer)
	planned := h.placed(t, "/planned/", pagemap.StatusPlanned, 0)
	archived := h.placed(t, "/gone/", pagemap.StatusArchived, 41)
	h.recorder.Reset()

	cases := []struct {
		name  string
		id    string
		code  errors.Code
		field string
	}{
		{name: "a page is needed", id: "  ", code: errors.Invalid, field: "pageId"},
		{name: "an unknown page is missing", id: "6fb0b1d6-1ad5-4a26-8f26-2f2b6d9e4b23", code: errors.NotFound},
		{name: "a planned page is not on the site", id: planned.ID, code: errors.Invalid, field: "wpId"},
		{name: "an archived page is gone from the site", id: archived.ID, code: errors.Invalid, field: "status"},
	}
	for _, tc := range cases {
		_, err := h.service.PreviewLink(t.Context(), pages.PreviewLinkRequest{PageID: tc.id})
		if !errors.IsCode(err, tc.code) {
			t.Errorf("%s: PreviewLink = %v, want %s", tc.name, err, tc.code)
		}
		if tc.field != "" && fieldOf(err) != tc.field {
			t.Errorf("%s: field = %q, want %q", tc.name, fieldOf(err), tc.field)
		}
	}
	if len(issuer.calls) != 0 {
		t.Errorf("the issuer was asked %d times for pages it cannot show", len(issuer.calls))
	}
	h.wantEvents(t)
}

func TestPreviewLinkOfAPublishedPageIsItsAddress(t *testing.T) {
	t.Parallel()

	issuer := &recordingIssuer{}
	h := newPreviewHarness(t, issuer)
	published := h.placed(t, "/coffee/", pagemap.StatusPublished, 42)
	h.recorder.Reset()

	answered, err := h.service.PreviewLink(t.Context(), pages.PreviewLinkRequest{PageID: published.ID})
	if err != nil {
		t.Fatalf("PreviewLink: %v", err)
	}
	if answered.URL != "https://shop.example.com/coffee/" || answered.Kind != string(pages.PreviewPublic) {
		t.Errorf("answer = %+v", answered)
	}
	encoded, err := json.Marshal(answered)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if string(encoded) != `{"url":"https://shop.example.com/coffee/","expiresAt":null,"kind":"public"}` {
		t.Errorf("json = %s", encoded)
	}
	if len(issuer.calls) != 0 {
		t.Errorf("a published page asked the site for a link")
	}
	h.wantEvents(t)
}

func TestPreviewLinkOfADraftIsIssuedByTheSite(t *testing.T) {
	t.Parallel()

	expires := time.Date(2026, time.September, 18, 10, 0, 0, 0, time.UTC)
	issuer := &recordingIssuer{answer: pages.IssuedPreview{URL: "https://shop.example.com/?page_id=43&postulator_preview=abc", ExpiresAt: expires}}
	h := newPreviewHarness(t, issuer)
	draft := h.placed(t, "/draft/", pagemap.StatusExists, 43)
	h.recorder.Reset()

	answered, err := h.service.PreviewLink(t.Context(), pages.PreviewLinkRequest{PageID: draft.ID})
	if err != nil {
		t.Fatalf("PreviewLink: %v", err)
	}
	if answered.URL != issuer.answer.URL || answered.Kind != string(pages.PreviewIssued) || !answered.ExpiresAt.Std().Equal(expires) {
		t.Errorf("answer = %+v", answered)
	}
	if len(issuer.calls) != 1 || issuer.calls[0] != (issued{siteID: h.siteID, wpID: 43}) {
		t.Errorf("issuer calls = %+v", issuer.calls)
	}
	h.wantEvents(t)
}

func TestPreviewLinkPassesTheSitesRefusalThrough(t *testing.T) {
	t.Parallel()

	refusal := errors.New(errors.Invalid, "the Postulator companion plugin is not installed on this site").
		WithDetail("code", "plugin_missing")
	issuer := &recordingIssuer{err: refusal}
	h := newPreviewHarness(t, issuer)
	draft := h.placed(t, "/draft/", pagemap.StatusExists, 43)

	_, err := h.service.PreviewLink(t.Context(), pages.PreviewLinkRequest{PageID: draft.ID})
	var kernel *errors.Error
	if !stderrors.As(err, &kernel) || kernel.Code != errors.Invalid || kernel.Details["code"] != "plugin_missing" {
		t.Fatalf("PreviewLink = %v, want the plugin_missing refusal unchanged", err)
	}
}
