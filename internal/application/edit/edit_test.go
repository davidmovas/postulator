package edit_test

import (
	"context"
	stderrors "errors"
	"maps"
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/application/edit"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type site struct {
	core     map[field.Key]string
	store    map[field.Key]string
	body     string
	seo      map[string]string
	meta     map[string]string
	calls    []string
	outdated bool
	noPlugin bool
}

func newSite() *site {
	return &site{
		core:  map[field.Key]string{field.Title: "Old title", field.Excerpt: "", field.FeaturedImage: ""},
		store: map[field.Key]string{field.ProductShortDescription: "Old short"},
		body:  "<p>Old body</p>",
		seo:   map[string]string{"title": "Old SEO"},
		meta:  map[string]string{"subtitle": "Old"},
	}
}

func (s *site) ReadCore(context.Context, edit.Target) (map[field.Key]string, error) {
	s.calls = append(s.calls, "read core")
	held := maps.Clone(s.core)
	held[field.Body] = s.body
	return held, nil
}

func (s *site) WriteCore(_ context.Context, _ edit.Target, values map[field.Key]string) (map[field.Key]string, error) {
	s.calls = append(s.calls, "write core")
	for key, value := range values {
		if key == field.Body {
			s.body = value
			continue
		}
		s.core[key] = value
	}
	held := maps.Clone(s.core)
	held[field.Body] = s.body
	return held, nil
}

func (s *site) ReadStore(context.Context, edit.Target) (map[field.Key]string, error) {
	s.calls = append(s.calls, "read store")
	return maps.Clone(s.store), nil
}

func (s *site) WriteStore(_ context.Context, _ edit.Target, values map[field.Key]string) (map[field.Key]string, error) {
	s.calls = append(s.calls, "write store")
	maps.Copy(s.store, values)
	return maps.Clone(s.store), nil
}

func (s *site) Plugin(context.Context) (bool, error) {
	if s.outdated {
		return true, errors.New(errors.Invalid, "the plugin is too old").WithDetail("code", "plugin_outdated")
	}
	return !s.noPlugin, nil
}

func (s *site) WriteBody(_ context.Context, _ edit.Target, content, expected string) (edit.Stored, *string, error) {
	s.calls = append(s.calls, "write body")
	if current := block.BodyHash(s.body); current != expected {
		return edit.Stored{}, &current, nil
	}
	s.body = content
	return edit.Stored{Value: &s.body, Hash: new(block.BodyHash(s.body))}, nil, nil
}

func (s *site) WriteSEO(_ context.Context, _ edit.Target, values, expected map[string]string) (hashes, moved map[string]string, err error) {
	s.calls = append(s.calls, "write seo")
	moved = make(map[string]string)
	for name, want := range expected {
		if current := block.BodyHash(s.seo[name]); current != want {
			moved[name] = current
		}
	}
	if len(moved) > 0 {
		return nil, moved, nil
	}
	hashes = make(map[string]string, len(values))
	for name, value := range values {
		s.seo[name] = value
		hashes[name] = block.BodyHash(value)
	}
	return hashes, nil, nil
}

func (s *site) WriteMeta(_ context.Context, _ edit.Target, writes map[string]edit.MetaWrite) (stored map[string]edit.Stored, moved map[string]*string, err error) {
	s.calls = append(s.calls, "write meta")
	moved = make(map[string]*string)
	for key, write := range writes {
		var current *string
		if value, held := s.meta[key]; held {
			current = new(block.BodyHash(value))
		}
		if (current == nil) != (write.Expected == nil) || (current != nil && *current != *write.Expected) {
			moved[key] = current
		}
	}
	if len(moved) > 0 {
		return nil, moved, nil
	}
	stored = make(map[string]edit.Stored, len(writes))
	for key, write := range writes {
		if write.Value == nil {
			delete(s.meta, key)
			stored[key] = edit.Stored{}
			continue
		}
		s.meta[key] = *write.Value
		stored[key] = edit.Stored{Value: write.Value, Hash: new(block.BodyHash(*write.Value))}
	}
	return stored, nil, nil
}

func serviceOver(s *site) edit.Service {
	return edit.New(edit.Writers{Core: s, Store: s, Plugin: s})
}

func hashOf(value string) *string {
	return new(block.BodyHash(value))
}

var page = edit.Target{Type: "page", ID: 7}

var product = edit.Target{Type: "product", ID: 9, Product: true}

func reasonOf(t *testing.T, err error) string {
	t.Helper()

	var kerr *errors.Error
	if !stderrors.As(err, &kerr) || kerr.Code != errors.Invalid {
		t.Fatalf("err = %v, want INVALID", err)
	}
	if reason, ok := kerr.Details["reason"].(string); ok {
		return reason
	}
	return ""
}

func TestADeniedFieldIsRefusedByTheUseCase(t *testing.T) {
	t.Parallel()

	for _, key := range append(field.Denied(), field.Instance(field.Meta, "_price"), field.Instance(field.Meta, "TOTAL_SALES"), field.H1, field.Key("rating")) {
		s := newSite()
		_, err := serviceOver(s).Write(t.Context(), page, []edit.Write{
			{Key: field.Title, Value: new("New title"), Expected: hashOf("Old title")},
			{Key: key, Value: new("9.99")},
		})
		if reason := reasonOf(t, err); reason != field.ReasonDenied && reason != edit.ReasonNoWriter {
			t.Errorf("%s: reason %q, want a refusal", key, reason)
		}
		if len(s.calls) != 0 || s.core[field.Title] != "Old title" {
			t.Errorf("%s: a refused write still reached the site: %v", key, s.calls)
		}
	}
}

func TestAProductsNameIsNeverWritten(t *testing.T) {
	t.Parallel()

	for _, key := range []field.Key{field.Title, field.Excerpt, field.FeaturedImage} {
		s := newSite()
		_, err := serviceOver(s).Write(t.Context(), product, []edit.Write{{Key: key, Value: new("x"), Expected: hashOf("Old title")}})
		if reason := reasonOf(t, err); reason != edit.ReasonStoreOwns {
			t.Errorf("%s on a product: reason %q, want %s", key, reason, edit.ReasonStoreOwns)
		}
		if len(s.calls) != 0 {
			t.Errorf("%s on a product reached the site: %v", key, s.calls)
		}
	}
	s := newSite()
	if _, err := serviceOver(s).Write(t.Context(), page, []edit.Write{{Key: field.ProductShortDescription, Value: new("x")}}); reasonOf(t, err) != edit.ReasonNoWriter {
		t.Errorf("a short description on a page = %v, want refused", err)
	}
}

func TestEveryFieldIsWrittenInItsOrder(t *testing.T) {
	t.Parallel()

	s := newSite()
	results, err := serviceOver(s).Write(t.Context(), page, []edit.Write{
		{Key: field.Body, Value: new("<p>New body</p>"), Expected: hashOf("<p>Old body</p>")},
		{Key: field.SEOTitle, Value: new("New SEO"), Expected: hashOf("Old SEO"), Social: &edit.Social{Value: "New OG", Expected: *hashOf("")}},
		{Key: field.Instance(field.Meta, "subtitle"), Value: new("New"), Expected: hashOf("Old")},
		{Key: field.Title, Value: new("New title"), Expected: hashOf("Old title")},
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if want := []string{"read core", "write core", "write meta", "write seo", "write body"}; !slices.Equal(s.calls, want) {
		t.Errorf("calls = %v, want %v", s.calls, want)
	}
	for _, result := range results {
		if result.Outcome != edit.OutcomeWritten || result.Hash == nil {
			t.Errorf("%s: %+v, want written with its hash", result.Key, result)
		}
	}
	if s.core[field.Title] != "New title" || s.body != "<p>New body</p>" || s.seo["title"] != "New SEO" || s.seo["ogTitle"] != "New OG" || s.meta["subtitle"] != "New" {
		t.Errorf("the site holds %+v %q %v %v", s.core, s.body, s.seo, s.meta)
	}
}

func TestAMovedValueIsNotWrittenOver(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		writes []edit.Write
		moved  field.Key
	}{
		{
			name:   "a title the client changed",
			writes: []edit.Write{{Key: field.Title, Value: new("New"), Expected: hashOf("Another title")}, {Key: field.Excerpt, Value: new("x")}},
			moved:  field.Title,
		},
		{
			name:   "a body the client changed",
			writes: []edit.Write{{Key: field.Body, Value: new("<p>New</p>"), Expected: hashOf("<p>His</p>")}},
			moved:  field.Body,
		},
		{
			name:   "an SEO title the client changed",
			writes: []edit.Write{{Key: field.SEOTitle, Value: new("New"), Expected: hashOf("His")}},
			moved:  field.SEOTitle,
		},
		{
			name:   "a custom field the client added",
			writes: []edit.Write{{Key: field.Instance(field.Meta, "note"), Value: new("Ours")}},
			moved:  field.Instance(field.Meta, "note"),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s := newSite()
			s.meta["note"] = "His"
			before := *s
			results, err := serviceOver(s).Write(t.Context(), page, tc.writes)
			if err != nil {
				t.Fatalf("Write: %v", err)
			}
			if s.core[field.Title] != before.core[field.Title] || s.body != before.body || s.seo["title"] != "Old SEO" || s.meta["note"] != "His" {
				t.Errorf("a moved value was written over: %+v %q %v %v", s.core, s.body, s.seo, s.meta)
			}
			found := false
			for _, result := range results {
				if result.Key == tc.moved {
					found = result.Outcome == edit.OutcomeMoved
				} else if result.Outcome == edit.OutcomeWritten {
					t.Errorf("%s was written beside a moved value", result.Key)
				}
			}
			if !found {
				t.Errorf("results = %+v, want %s moved", results, tc.moved)
			}
		})
	}
}

func TestAnOutdatedPluginWritesNothing(t *testing.T) {
	t.Parallel()

	s := newSite()
	s.outdated = true
	_, err := serviceOver(s).Write(t.Context(), page, []edit.Write{
		{Key: field.Title, Value: new("New title"), Expected: hashOf("Old title")},
		{Key: field.SEOTitle, Value: new("New SEO"), Expected: hashOf("Old SEO")},
	})
	if err == nil || len(s.calls) != 0 {
		t.Fatalf("err = %v with calls %v, want a refusal before any write", err, s.calls)
	}
}

func TestWithoutThePluginTheBodyIsWrittenThroughCoreUnderTheValueRead(t *testing.T) {
	t.Parallel()

	s := newSite()
	s.noPlugin = true
	results, err := serviceOver(s).Write(t.Context(), page, []edit.Write{{Key: field.Body, Value: new("<p>New</p>"), Expected: hashOf("<p>Old body</p>")}})
	if err != nil || len(results) != 1 || results[0].Outcome != edit.OutcomeWritten || s.body != "<p>New</p>" {
		t.Fatalf("results = %+v, err = %v, body %q", results, err, s.body)
	}
	if want := []string{"read core", "write core"}; !slices.Equal(s.calls, want) {
		t.Errorf("calls = %v, want %v", s.calls, want)
	}

	s = newSite()
	s.noPlugin = true
	if _, err = serviceOver(s).Write(t.Context(), page, []edit.Write{{Key: field.SEOTitle, Value: new("x")}}); reasonOf(t, err) != edit.ReasonNoPlugin {
		t.Errorf("an SEO field without the plugin = %v", err)
	}
	if _, err = serviceOver(s).Write(t.Context(), product, []edit.Write{{Key: field.ProductDescription, Value: new("x")}}); reasonOf(t, err) != edit.ReasonNoPlugin {
		t.Errorf("a product description without the plugin = %v", err)
	}
}
