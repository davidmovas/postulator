package keyword_test

import (
	"encoding/json"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/keyword"
)

const unknown = -1

type want struct {
	text   string
	volume int
}

func assertList(t *testing.T, got keyword.List, wanted []want) {
	t.Helper()
	if len(got) != len(wanted) {
		t.Fatalf("list = %s, want %d keywords", describe(got), len(wanted))
	}
	for i, expected := range wanted {
		if got[i].Text != expected.text {
			t.Fatalf("keyword %d = %q, want %q in %s", i, got[i].Text, expected.text, describe(got))
		}
		switch {
		case expected.volume == unknown && got[i].Volume != nil:
			t.Fatalf("keyword %q carries volume %d, want none", got[i].Text, *got[i].Volume)
		case expected.volume != unknown && got[i].Volume == nil:
			t.Fatalf("keyword %q carries no volume, want %d", got[i].Text, expected.volume)
		case expected.volume != unknown && *got[i].Volume != expected.volume:
			t.Fatalf("keyword %q carries volume %d, want %d", got[i].Text, *got[i].Volume, expected.volume)
		}
	}
}

func describe(list keyword.List) string {
	encoded, err := json.Marshal(list)
	if err != nil {
		return err.Error()
	}
	return string(encoded)
}

func TestNewOrdersByVolumeAndKeepsTheRestAsGiven(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		items []keyword.Keyword
		want  []want
	}{
		{
			name: "the highest volume leads and the unmeasured follow in their order",
			items: []keyword.Keyword{
				{Text: "bpc-157"},
				{Text: "bpc 157 dosage", Volume: new(1900)},
				{Text: "what is bpc"},
				{Text: "bpc 157", Volume: new(12000)},
			},
			want: []want{{"bpc 157", 12000}, {"bpc 157 dosage", 1900}, {"bpc-157", unknown}, {"what is bpc", unknown}},
		},
		{
			name:  "no volume anywhere keeps the order of the file",
			items: []keyword.Keyword{{Text: "tb 500"}, {Text: "tb500 peptide"}, {Text: "thymosin beta 4"}},
			want:  []want{{"tb 500", unknown}, {"tb500 peptide", unknown}, {"thymosin beta 4", unknown}},
		},
		{
			name:  "equal volumes keep their order",
			items: []keyword.Keyword{{Text: "b", Volume: new(50)}, {Text: "a", Volume: new(50)}, {Text: "c", Volume: new(70)}},
			want:  []want{{"c", 70}, {"b", 50}, {"a", 50}},
		},
		{
			name:  "text is trimmed and a blank one is dropped",
			items: []keyword.Keyword{{Text: "  buy bpc 157 "}, {Text: "   "}, {Text: ""}},
			want:  []want{{"buy bpc 157", unknown}},
		},
		{
			name:  "a repeat keeps the first spelling",
			items: []keyword.Keyword{{Text: "BPC 157"}, {Text: "bpc 157"}, {Text: "Bpc 157 "}},
			want:  []want{{"BPC 157", unknown}},
		},
		{
			name:  "a repeat with a volume gives it to the first spelling",
			items: []keyword.Keyword{{Text: "BPC 157"}, {Text: "bpc 157", Volume: new(800)}},
			want:  []want{{"BPC 157", 800}},
		},
		{
			name:  "the first known volume of a repeat stands",
			items: []keyword.Keyword{{Text: "bpc 157", Volume: new(500)}, {Text: "BPC 157", Volume: new(900)}},
			want:  []want{{"bpc 157", 500}},
		},
		{
			name:  "a zero volume is known and leads the unmeasured",
			items: []keyword.Keyword{{Text: "unmeasured"}, {Text: "nobody searches", Volume: new(0)}},
			want:  []want{{"nobody searches", 0}, {"unmeasured", unknown}},
		},
		{
			name:  "a negative volume is not a volume",
			items: []keyword.Keyword{{Text: "broken", Volume: new(-5)}, {Text: "fine", Volume: new(1)}},
			want:  []want{{"fine", 1}, {"broken", unknown}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assertList(t, keyword.New(tc.items), tc.want)
		})
	}
}

func TestNewOfNothingIsAnEmptyList(t *testing.T) {
	t.Parallel()

	list := keyword.New(nil)
	if list == nil || len(list) != 0 {
		t.Fatalf("New(nil) = %#v, want an empty list that is not nil", list)
	}
	if got := describe(list); got != "[]" {
		t.Fatalf("an empty list encodes as %s, want []", got)
	}
}

func TestOfIsAListOfPhrasesInTheOrderGiven(t *testing.T) {
	t.Parallel()

	assertList(t, keyword.Of("running shoes", " trail shoes ", "", "Trail Shoes", "road shoes"),
		[]want{{"running shoes", unknown}, {"trail shoes", unknown}, {"road shoes", unknown}})

	if empty := keyword.Of(); empty == nil || len(empty) != 0 {
		t.Fatalf("Of() = %#v, want an empty list that is not nil", empty)
	}
}

func TestNewDoesNotShareTheCallersVolume(t *testing.T) {
	t.Parallel()

	volume := 300
	list := keyword.New([]keyword.Keyword{{Text: "bpc 157", Volume: &volume}})
	volume = 1

	assertList(t, list, []want{{"bpc 157", 300}})
}

func TestAKeywordEncodesItsVolumeOnlyWhenItHasOne(t *testing.T) {
	t.Parallel()

	list := keyword.New([]keyword.Keyword{{Text: "bpc 157", Volume: new(12000)}, {Text: "bpc-157"}})
	if got, expected := describe(list), `[{"text":"bpc 157","volume":12000},{"text":"bpc-157"}]`; got != expected {
		t.Fatalf("encoded = %s, want %s", got, expected)
	}

	var back []keyword.Keyword
	if err := json.Unmarshal([]byte(describe(list)), &back); err != nil {
		t.Fatalf("decode: %v", err)
	}
	assertList(t, keyword.New(back), []want{{"bpc 157", 12000}, {"bpc-157", unknown}})
}

func TestMainIsTheFirstKeyword(t *testing.T) {
	t.Parallel()

	list := keyword.New([]keyword.Keyword{{Text: "bpc-157"}, {Text: "bpc 157", Volume: new(12000)}})
	if got := list.Main(); got != "bpc 157" {
		t.Fatalf("Main = %q, want the keyword with the highest volume", got)
	}
	if got := keyword.New(nil).Main(); got != "" {
		t.Fatalf("Main of an empty list = %q, want nothing", got)
	}
}

func TestTextsAreTheKeywordsInOrder(t *testing.T) {
	t.Parallel()

	list := keyword.New([]keyword.Keyword{{Text: "bpc-157"}, {Text: "bpc 157", Volume: new(12000)}})
	texts := list.Texts()
	if len(texts) != 2 || texts[0] != "bpc 157" || texts[1] != "bpc-157" {
		t.Fatalf("Texts = %v", texts)
	}
	if empty := keyword.New(nil).Texts(); empty == nil || len(empty) != 0 {
		t.Fatalf("Texts of an empty list = %#v, want an empty slice that is not nil", empty)
	}
}

func TestMergeTakesWhatTheFileSaysAndKeepsTheRest(t *testing.T) {
	t.Parallel()

	stored := []keyword.Keyword{
		{Text: "BPC 157", Volume: new(9000)},
		{Text: "bpc 157 dosage", Volume: new(1900)},
		{Text: "curated by hand"},
	}

	cases := []struct {
		name string
		file []keyword.Keyword
		want []want
	}{
		{
			name: "an empty cell changes nothing",
			file: nil,
			want: []want{{"BPC 157", 9000}, {"bpc 157 dosage", 1900}, {"curated by hand", unknown}},
		},
		{
			name: "the volume of the file replaces the stored one and the spelling stays",
			file: []keyword.Keyword{{Text: "bpc 157", Volume: new(12000)}},
			want: []want{{"BPC 157", 12000}, {"bpc 157 dosage", 1900}, {"curated by hand", unknown}},
		},
		{
			name: "a keyword the file gives without a volume keeps the stored volume",
			file: []keyword.Keyword{{Text: "bpc 157 dosage"}},
			want: []want{{"BPC 157", 9000}, {"bpc 157 dosage", 1900}, {"curated by hand", unknown}},
		},
		{
			name: "a new keyword is added and the list is ordered again",
			file: []keyword.Keyword{{Text: "buy bpc 157", Volume: new(5400)}, {Text: "bpc-157"}},
			want: []want{
				{"BPC 157", 9000}, {"buy bpc 157", 5400}, {"bpc 157 dosage", 1900},
				{"curated by hand", unknown}, {"bpc-157", unknown},
			},
		},
		{
			name: "a volume the file lowers reorders the list",
			file: []keyword.Keyword{{Text: "BPC 157", Volume: new(100)}},
			want: []want{{"bpc 157 dosage", 1900}, {"BPC 157", 100}, {"curated by hand", unknown}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			merged := keyword.New(stored).Merge(keyword.New(tc.file))
			assertList(t, merged, tc.want)
		})
	}
}

func TestMergeLeavesBothListsAsTheyWere(t *testing.T) {
	t.Parallel()

	stored := keyword.New([]keyword.Keyword{{Text: "bpc 157", Volume: new(9000)}})
	file := keyword.New([]keyword.Keyword{{Text: "bpc 157", Volume: new(12000)}, {Text: "bpc-157"}})

	stored.Merge(file)

	assertList(t, stored, []want{{"bpc 157", 9000}})
	assertList(t, file, []want{{"bpc 157", 12000}, {"bpc-157", unknown}})
}
