package application_test

import (
	"encoding/json"
	stderrors "errors"
	"testing"

	"github.com/davidmovas/postulator/internal/application"
	"github.com/davidmovas/postulator/internal/domain/keyword"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestKeywordViewsCarryTheListAsItIsOrdered(t *testing.T) {
	t.Parallel()

	views := application.KeywordViews(keyword.New([]keyword.Keyword{
		{Text: "bpc-157"}, {Text: "bpc 157", Volume: new(12000)}, {Text: "rare", Volume: new(0)},
	}))
	encoded, err := json.Marshal(views)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if want := `[{"text":"bpc 157","volume":12000},{"text":"rare","volume":0},{"text":"bpc-157"}]`; string(encoded) != want {
		t.Fatalf("views = %s, want %s", encoded, want)
	}

	empty := application.KeywordViews(nil)
	if empty == nil || len(empty) != 0 {
		t.Fatalf("the views of no keywords = %#v, want an empty list that is not nil", empty)
	}
}

func TestKeywordViewsDoNotShareTheVolumesOfTheList(t *testing.T) {
	t.Parallel()

	list := keyword.New([]keyword.Keyword{{Text: "bpc 157", Volume: new(12000)}})
	views := application.KeywordViews(list)
	*views[0].Volume = 1

	if *list[0].Volume != 12000 {
		t.Fatalf("writing to a view changed the list: %d", *list[0].Volume)
	}
}

func TestKeywordListNormalisesWhatARequestCarries(t *testing.T) {
	t.Parallel()

	list, err := application.KeywordList([]dto.Keyword{
		{Text: " bpc-157 "}, {Text: "bpc 157", Volume: new(12000)}, {Text: "BPC-157"},
	}, "keywords")
	if err != nil {
		t.Fatalf("KeywordList: %v", err)
	}
	want := keyword.New([]keyword.Keyword{{Text: "bpc 157", Volume: new(12000)}, {Text: "bpc-157"}})
	if !list.Equal(want) {
		t.Fatalf("list = %+v, want %+v", list, want)
	}

	empty, err := application.KeywordList(nil, "keywords")
	if err != nil || empty == nil || len(empty) != 0 {
		t.Fatalf("the list of no keywords = %#v, %v; want an empty list that is not nil", empty, err)
	}
}

func TestKeywordListRefusesAKeywordItCannotKeep(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		items []dto.Keyword
		field string
	}{
		{name: "no phrase", items: []dto.Keyword{{Text: "kept"}, {Text: "  ", Volume: new(40)}}, field: "entities[2].keywords[1].text"},
		{name: "a negative volume", items: []dto.Keyword{{Text: "bpc 157", Volume: new(-1)}}, field: "entities[2].keywords[0].volume"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := application.KeywordList(tc.items, "entities[2].keywords")
			if !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("KeywordList = %v, want an invalid error", err)
			}
			var kernel *errors.Error
			if !stderrors.As(err, &kernel) || kernel.Details["field"] != tc.field {
				t.Fatalf("error %v names the wrong field, want %s", err, tc.field)
			}
		})
	}
}
