package field_test

import (
	"slices"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/domain/field"
)

func TestTheCatalogNamesEveryDeniedField(t *testing.T) {
	t.Parallel()

	want := []field.Key{
		field.Attributes, field.ManageStock, field.Parent, field.Price, field.RegularPrice, field.SalePrice, field.SKU,
		field.Slug, field.Status, field.StockQuantity, field.StockStatus, field.Terms, field.Variation,
	}
	got := field.Denied()
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Fatalf("Denied() = %v, want %v", got, want)
	}
	for _, key := range got {
		spec, found := field.Lookup(key)
		if !found || !spec.Denied || spec.Write != field.None {
			t.Errorf("%s: spec %+v found %t, want denied with no writer", key, spec, found)
		}
	}
	for _, spec := range field.Catalog() {
		if !spec.Denied && spec.Text && spec.Write == field.None {
			t.Errorf("%s is a writable text field without a writer", spec.Key)
		}
		if spec.Read == field.None {
			t.Errorf("%s has no reader", spec.Key)
		}
	}
}

func TestAnInstancedKeyResolvesByItsKind(t *testing.T) {
	t.Parallel()

	key := field.Instance(field.VariationDescription, "123")
	if key != "variation_description:123" {
		t.Fatalf("Instance = %q", key)
	}
	if key.Kind() != field.VariationDescription || key.Instance() != "123" {
		t.Errorf("Kind = %q, Instance = %q", key.Kind(), key.Instance())
	}
	spec, found := field.Lookup(key)
	if !found || !spec.Instanced || spec.Key != field.VariationDescription {
		t.Errorf("Lookup(%s) = %+v %t", key, spec, found)
	}
	if field.Title.Instance() != "" || field.Title.Kind() != field.Title {
		t.Error("a plain key has no instance")
	}
	if _, found := field.Lookup("made_up"); found {
		t.Error("an unknown key is found")
	}
	if _, found := field.Lookup(field.Instance(field.Meta, "_price")); !found {
		t.Error("a meta key by name is a field")
	}
}

func TestAValueStartsAgreed(t *testing.T) {
	t.Parallel()

	at := time.Date(2026, time.October, 17, 12, 0, 0, 0, time.UTC)
	value := field.NewValue("page", field.Title, "Hello", at)
	if value.Current != "Hello" || value.Base != "Hello" {
		t.Fatalf("value = %+v", value)
	}
	if value.CurrentHash == "" || value.CurrentHash != field.Hash("Hello") || value.BaseHash != value.CurrentHash {
		t.Errorf("hashes = %q %q", value.CurrentHash, value.BaseHash)
	}
	if value.Changed() || !value.ObservedAt.Equal(at) {
		t.Errorf("a new value is agreed and stamped: %+v", value)
	}
	value.Current = "Bye"
	value.CurrentHash = field.Hash("Bye")
	if !value.Changed() {
		t.Error("a moved current is a change")
	}
	if field.Hash("") != "" {
		t.Error("an empty value has an empty hash")
	}
}
