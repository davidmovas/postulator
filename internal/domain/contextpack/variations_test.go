package contextpack_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/field"
)

func TestThePagesVariationsAreReadFromItsFields(t *testing.T) {
	t.Parallel()

	at := time.Unix(1, 0).UTC()
	run := "run-1"
	ours := field.NewValue("p", field.Instance(field.VariationDescription, "12"), "<p>Ours.</p>", at)
	ours.WrittenRunID = &run
	edited := field.NewValue("p", field.Instance(field.VariationDescription, "15"), "<p>Edited since.</p>", at)
	edited.WrittenRunID, edited.BaseHash = &run, field.Hash("<p>Ours once.</p>")
	values := []field.Value{
		field.NewValue("p", field.Instance(field.Variation, "12"), `{"sku":"A","attributes":[{"name":"Size","option":"10 mg"},{"name":"Form","option":"Liquid"}]}`, at),
		ours,
		field.NewValue("p", field.Instance(field.Variation, "9"), `{"attributes":[{"name":"Size","option":"5 mg"},{"name":"","option":"x"}]}`, at),
		field.NewValue("p", field.Instance(field.VariationDescription, "9"), "<p>His.</p>", at),
		field.NewValue("p", field.Instance(field.Variation, "15"), `not the facts`, at),
		edited,
		field.NewValue("p", field.Instance(field.VariationDescription, "30"), "<p>A variation the store no longer lists.</p>", at),
		field.NewValue("p", field.ProductShortDescription, "<p>Short.</p>", at),
	}

	want := []contextpack.Variation{
		{Key: field.Instance(field.VariationDescription, "9"), Options: []string{"Size: 5 mg"}, Description: "<p>His.</p>"},
		{Key: field.Instance(field.VariationDescription, "12"), Options: []string{"Size: 10 mg", "Form: Liquid"}, Description: "<p>Ours.</p>", Ours: true},
		{Key: field.Instance(field.VariationDescription, "15"), Options: []string{}, Description: "<p>Edited since.</p>"},
	}
	if got := contextpack.Variations(values); !reflect.DeepEqual(got, want) {
		t.Errorf("Variations =\n%+v\nwant\n%+v", got, want)
	}
	if got := contextpack.Variations(nil); len(got) != 0 {
		t.Errorf("a page with no fields has the variations %+v", got)
	}
}
