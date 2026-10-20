package edit_test

import (
	"context"
	stderrors "errors"
	"maps"
	"slices"
	"strconv"
	"testing"

	"github.com/davidmovas/postulator/internal/application/edit"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type site struct {
	core       map[field.Key]string
	store      map[field.Key]string
	variations map[int64]string
	media      map[int64]edit.Media
	body       string
	seo        map[string]string
	meta       map[string]string
	calls      []string
	outdated   bool
	noPlugin   bool
}

func newSite() *site {
	return &site{
		core:       map[field.Key]string{field.Title: "Old title", field.Excerpt: "", field.FeaturedImage: ""},
		store:      map[field.Key]string{field.ProductShortDescription: "Old short"},
		variations: map[int64]string{5: "<p>Old five</p>", 6: "<p>Old six</p>"},
		media:      map[int64]edit.Media{41: {Alt: "", Title: "vial", Parent: 9}, 42: {Alt: "Shared", Title: "shared", Parent: 0}},
		body:       "<p>Old body</p>",
		seo:        map[string]string{"title": "Old SEO"},
		meta:       map[string]string{"subtitle": "Old"},
	}
}

func (s *site) ReadMedia(_ context.Context, id int64) (edit.Media, error) {
	s.calls = append(s.calls, "read media "+strconv.FormatInt(id, 10))
	held, found := s.media[id]
	if !found {
		return edit.Media{}, errors.New(errors.NotFound, "no such media")
	}
	return held, nil
}

func (s *site) WriteMedia(_ context.Context, id int64, alt, title *string) (edit.Media, error) {
	s.calls = append(s.calls, "write media "+strconv.FormatInt(id, 10))
	held := s.media[id]
	if alt != nil {
		held.Alt = *alt
	}
	if title != nil {
		held.Title = *title
	}
	s.media[id] = held
	return held, nil
}

func (s *site) ReadVariation(_ context.Context, _ edit.Target, id int64) (string, error) {
	s.calls = append(s.calls, "read variation "+strconv.FormatInt(id, 10))
	held, found := s.variations[id]
	if !found {
		return "", errors.New(errors.NotFound, "no such variation")
	}
	return held, nil
}

func (s *site) WriteVariation(_ context.Context, _ edit.Target, id int64, description string) (string, error) {
	s.calls = append(s.calls, "write variation "+strconv.FormatInt(id, 10))
	s.variations[id] = description
	return description, nil
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

	for _, key := range []field.Key{field.Title, field.Excerpt} {
		s := newSite()
		_, err := serviceOver(s).Write(t.Context(), product, []edit.Write{{Key: key, Value: new("x"), Expected: hashOf("Old title")}})
		if reason := reasonOf(t, err); reason != edit.ReasonStoreOwns {
			t.Errorf("%s on a product: reason %q, want %s", key, reason, edit.ReasonStoreOwns)
		}
		if len(s.calls) != 0 {
			t.Errorf("%s on a product reached the site: %v", key, s.calls)
		}
	}
	image := newSite()
	if _, err := serviceOver(image).Write(t.Context(), product, []edit.Write{{Key: field.FeaturedImage, Value: new("12")}}); err != nil ||
		!slices.Equal(image.calls, []string{"read store", "write store"}) {
		t.Errorf("a product's featured image = %v with calls %v, want it written through the store", err, image.calls)
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

func TestAVariationsDescriptionIsWrittenUnderTheValueRead(t *testing.T) {
	t.Parallel()

	five, six := field.Instance(field.VariationDescription, "5"), field.Instance(field.VariationDescription, "6")

	s := newSite()
	results, err := serviceOver(s).Write(t.Context(), product, []edit.Write{
		{Key: field.SEOTitle, Value: new("New SEO"), Expected: hashOf("Old SEO")},
		{Key: six, Value: new("<p>New six</p>"), Expected: hashOf("<p>Old six</p>")},
		{Key: five, Value: new("<p>New five</p>"), Expected: hashOf("<p>Old five</p>")},
		{Key: field.ProductShortDescription, Value: new("New short"), Expected: hashOf("Old short")},
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	want := []string{"read store", "write store", "read variation 5", "read variation 6", "write variation 5", "write variation 6", "write seo"}
	if !slices.Equal(s.calls, want) {
		t.Errorf("calls = %v, want %v: the variations after the store's own fields, each read before any is written", s.calls, want)
	}
	for _, result := range results {
		if result.Outcome != edit.OutcomeWritten || result.Hash == nil {
			t.Errorf("%s: %+v, want written with its hash", result.Key, result)
		}
	}
	if s.variations[5] != "<p>New five</p>" || s.variations[6] != "<p>New six</p>" {
		t.Errorf("the variations hold %v", s.variations)
	}

	moved := newSite()
	moved.variations[6] = "<p>His six</p>"
	results, err = serviceOver(moved).Write(t.Context(), product, []edit.Write{
		{Key: five, Value: new("<p>New five</p>"), Expected: hashOf("<p>Old five</p>")},
		{Key: six, Value: new("<p>New six</p>"), Expected: hashOf("<p>Old six</p>")},
	})
	if err != nil {
		t.Fatalf("Write over a moved variation: %v", err)
	}
	outcomes := map[field.Key]edit.Outcome{}
	for _, result := range results {
		outcomes[result.Key] = result.Outcome
	}
	if outcomes[six] != edit.OutcomeMoved || outcomes[five] == edit.OutcomeWritten || moved.variations[5] != "<p>Old five</p>" || moved.variations[6] != "<p>His six</p>" {
		t.Errorf("outcomes %v over %v, want the moved one named and nothing written", outcomes, moved.variations)
	}

	already := newSite()
	already.variations[5] = "<p>New five</p>"
	results, err = serviceOver(already).Write(t.Context(), product, []edit.Write{
		{Key: five, Value: new("<p>New five</p>"), Expected: hashOf("<p>Old five</p>")},
	})
	if err != nil || len(results) != 1 || results[0].Outcome != edit.OutcomeWritten || slices.Contains(already.calls, "write variation 5") {
		t.Errorf("a variation already holding our value = %+v, %v with calls %v, want it taken as written and not sent again", results, err, already.calls)
	}
}

func TestAnImagesAltAndTitleAreWrittenOnlyWhereItIsAttached(t *testing.T) {
	t.Parallel()

	alt, title := field.Instance(field.ImageAlt, "41"), field.Instance(field.ImageTitle, "41")
	s := newSite()
	results, err := serviceOver(s).Write(t.Context(), product, []edit.Write{
		{Key: title, Value: new("Vial of powder"), Expected: hashOf("vial")},
		{Key: alt, Value: new("A vial of powder")},
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if !slices.Equal(s.calls, []string{"read media 41", "write media 41"}) {
		t.Errorf("calls = %v, want the image read once and written once with both", s.calls)
	}
	for _, result := range results {
		if result.Outcome != edit.OutcomeWritten || result.Hash == nil {
			t.Errorf("%s: %+v, want written", result.Key, result)
		}
	}
	if held := s.media[41]; held.Alt != "A vial of powder" || held.Title != "Vial of powder" {
		t.Errorf("the image holds %+v", held)
	}

	shared := newSite()
	_, err = serviceOver(shared).Write(t.Context(), product, []edit.Write{{Key: field.Instance(field.ImageAlt, "42"), Value: new("Ours"), Expected: hashOf("Shared")}})
	if reason := reasonOf(t, err); reason != edit.ReasonNotAttached {
		t.Errorf("an image attached elsewhere: reason %q, want %s", reason, edit.ReasonNotAttached)
	}
	if slices.Contains(shared.calls, "write media 42") || shared.media[42].Alt != "Shared" {
		t.Errorf("an image another page may show was renamed for this one: %v %+v", shared.calls, shared.media[42])
	}

	moved := newSite()
	moved.media[41] = edit.Media{Alt: "His alt", Title: "vial", Parent: 9}
	results, err = serviceOver(moved).Write(t.Context(), product, []edit.Write{
		{Key: alt, Value: new("Ours")}, {Key: title, Value: new("Ours"), Expected: hashOf("vial")},
	})
	if err != nil || slices.Contains(moved.calls, "write media 41") || results[0].Outcome == edit.OutcomeWritten && results[1].Outcome == edit.OutcomeWritten {
		t.Errorf("an alt the client wrote = %+v, %v with calls %v, want nothing written", results, err, moved.calls)
	}

	if _, err = serviceOver(newSite()).Write(t.Context(), product, []edit.Write{{Key: field.Instance(field.ImageAlt, "image"), Value: new("x")}}); reasonOf(t, err) != field.ReasonUnknown {
		t.Errorf("an image named by no id = %v, want refused", err)
	}
}

func TestACustomFieldIsWrittenByTheProviderItsScopeNames(t *testing.T) {
	t.Parallel()

	dosage := field.Instance(field.Meta, "dosage")
	store := newSite()
	results, err := serviceOver(store).Write(t.Context(), product, []edit.Write{{Key: dosage, Value: new("Once a day"), Provider: field.ProviderStore}})
	if err != nil || len(results) != 1 || results[0].Outcome != edit.OutcomeWritten {
		t.Fatalf("a store custom field = %+v, %v", results, err)
	}
	if !slices.Equal(store.calls, []string{"read store", "write store"}) || store.store[dosage] != "Once a day" {
		t.Errorf("calls %v, store %v, want it written through the store", store.calls, store.store)
	}

	plugin := newSite()
	if _, err = serviceOver(plugin).Write(t.Context(), product, []edit.Write{{Key: dosage, Value: new("Once a day"), Provider: field.ProviderPlugin}}); err != nil ||
		!slices.Equal(plugin.calls, []string{"write meta"}) {
		t.Errorf("a plugin custom field = %v with calls %v, want it written through the plugin", err, plugin.calls)
	}

	page2 := newSite()
	if _, err = serviceOver(page2).Write(t.Context(), page, []edit.Write{{Key: dosage, Value: new("x"), Provider: field.ProviderStore}}); reasonOf(t, err) != edit.ReasonNoWriter {
		t.Errorf("a store custom field on a page = %v, want refused", err)
	}
	denied := newSite()
	if _, err = serviceOver(denied).Write(t.Context(), product, []edit.Write{{Key: field.Instance(field.Meta, "_price"), Value: new("1"), Provider: field.ProviderStore}}); reasonOf(t, err) != field.ReasonDenied ||
		len(denied.calls) != 0 {
		t.Errorf("the store's own key through the store = %v with calls %v, want refused before anything leaves", err, denied.calls)
	}
}

func TestAProductsUpsellsAreWrittenThroughTheStoreAndAPagesAreRefused(t *testing.T) {
	t.Parallel()

	for _, key := range []field.Key{field.Upsells, field.CrossSells} {
		s := newSite()
		results, err := serviceOver(s).Write(t.Context(), product, []edit.Write{{Key: key, Value: new("21,30")}})
		if err != nil || len(results) != 1 || results[0].Outcome != edit.OutcomeWritten {
			t.Errorf("%s on a product = %+v, %v", key, results, err)
		}
		if !slices.Equal(s.calls, []string{"read store", "write store"}) || s.store[key] != "21,30" {
			t.Errorf("%s: calls %v, store %v, want it written through the store", key, s.calls, s.store)
		}
		if _, err = serviceOver(newSite()).Write(t.Context(), page, []edit.Write{{Key: key, Value: new("21")}}); reasonOf(t, err) != edit.ReasonNoWriter {
			t.Errorf("%s on a page = %v, want refused", key, err)
		}
	}
}

func TestAVariationIsWrittenOnAProductOnly(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		target edit.Target
		key    field.Key
		reason string
	}{
		{name: "a page", target: page, key: field.Instance(field.VariationDescription, "5"), reason: edit.ReasonNoWriter},
		{name: "no variation named", target: product, key: field.VariationDescription, reason: field.ReasonUnknown},
		{name: "a variation that is no id", target: product, key: field.Instance(field.VariationDescription, "five"), reason: field.ReasonUnknown},
		{name: "a variation's own facts", target: product, key: field.Instance(field.Variation, "5"), reason: field.ReasonDenied},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s := newSite()
			_, err := serviceOver(s).Write(t.Context(), tc.target, []edit.Write{{Key: tc.key, Value: new("<p>x</p>")}})
			if reason := reasonOf(t, err); reason != tc.reason {
				t.Errorf("reason %q, want %q", reason, tc.reason)
			}
			if len(s.calls) != 0 {
				t.Errorf("a refused write reached the site: %v", s.calls)
			}
		})
	}
}

func TestAValueAlreadyStoredIsTakenAsWritten(t *testing.T) {
	t.Parallel()

	s := newSite()
	s.core[field.Title], s.body, s.seo["title"], s.meta["subtitle"] = "New title", "<p>New</p>", "New SEO", "New"
	results, err := serviceOver(s).Write(t.Context(), page, []edit.Write{
		{Key: field.Title, Value: new("New title"), Expected: hashOf("Old title")},
		{Key: field.Excerpt, Value: new("Fresh")},
		{Key: field.Instance(field.Meta, "subtitle"), Value: new("New"), Expected: hashOf("Old")},
		{Key: field.SEOTitle, Value: new("New SEO"), Expected: hashOf("Old SEO")},
		{Key: field.Body, Value: new("<p>New</p>"), Expected: hashOf("<p>Old body</p>")},
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	for _, result := range results {
		if result.Outcome != edit.OutcomeWritten {
			t.Errorf("%s is %s, want a value the site already holds taken as written", result.Key, result.Outcome)
		}
	}
	if s.core[field.Excerpt] != "Fresh" {
		t.Errorf("the excerpt beside an already stored title was not written: %q", s.core[field.Excerpt])
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
