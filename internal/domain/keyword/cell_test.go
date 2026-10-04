package keyword_test

import (
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/keyword"
)

func TestParseReadsACellOfKeywords(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		cell string
		want []want
		bad  []string
	}{
		{
			name: "the documented form",
			cell: "bpc 157 (12000), buy bpc 157 (5400), bpc 157 dosage (1,900), bpc-157",
			want: []want{{"bpc 157", 12000}, {"buy bpc 157", 5400}, {"bpc 157 dosage", 1900}, {"bpc-157", unknown}},
		},
		{
			name: "keywords without a volume keep the order of the cell",
			cell: "tb 500, tb500 peptide, thymosin beta 4",
			want: []want{{"tb 500", unknown}, {"tb500 peptide", unknown}, {"thymosin beta 4", unknown}},
		},
		{name: "an empty cell holds no keywords", cell: "", want: nil},
		{name: "a cell of blanks and separators holds no keywords", cell: "  , ;\n | ", want: nil},
		{
			name: "a semicolon, a bar and a line break separate as a comma does",
			cell: "one; two | three\nfour\r\nfive",
			want: []want{{"one", unknown}, {"two", unknown}, {"three", unknown}, {"four", unknown}, {"five", unknown}},
		},
		{
			name: "square brackets hold a volume as round ones do",
			cell: "bpc 157 [1200], tb 500 [ 300 ]",
			want: []want{{"bpc 157", 1200}, {"tb 500", 300}},
		},
		{
			name: "a volume is read however its thousands are written",
			cell: "plain (1200), comma (1,300), space (1 400), narrow (1 500), dot (1.600), million (2,500,000)",
			want: []want{
				{"million", 2500000}, {"dot", 1600}, {"narrow", 1500}, {"space", 1400}, {"comma", 1300}, {"plain", 1200},
			},
		},
		{
			name: "a volume is read with a thousand or a million suffix",
			cell: "small (1.2k), upper (3K), comma (4,5k), whole (12k), big (1.5m), bigger (2M)",
			want: []want{
				{"bigger", 2000000}, {"big", 1500000}, {"whole", 12000}, {"comma", 4500}, {"upper", 3000}, {"small", 1200},
			},
		},
		{
			name: "a separator inside brackets does not split the cell",
			cell: "bpc 157 (1,900), tb 500",
			want: []want{{"bpc 157", 1900}, {"tb 500", unknown}},
		},
		{
			name: "brackets that hold words belong to the keyword",
			cell: "iphone (pro), vitamin (b12), peptides (injectable, oral) (700)",
			want: []want{{"peptides (injectable, oral)", 700}, {"iphone (pro)", unknown}, {"vitamin (b12)", unknown}},
		},
		{
			name: "a number that is not in brackets belongs to the keyword",
			cell: "omega 3, bpc 157 5mg (90)",
			want: []want{{"bpc 157 5mg", 90}, {"omega 3", unknown}},
		},
		{
			name: "a zero volume is a volume",
			cell: "nobody searches (0), unmeasured",
			want: []want{{"nobody searches", 0}, {"unmeasured", unknown}},
		},
		{
			name: "a volume that cannot be read leaves the keyword without one and is reported",
			cell: "typo (12o0), fraction (1.5), short group (1,20), two suffixes (1kk), fine (40)",
			want: []want{
				{"fine", 40}, {"typo", unknown}, {"fraction", unknown}, {"short group", unknown}, {"two suffixes", unknown},
			},
			bad: []string{"typo (12o0)", "fraction (1.5)", "short group (1,20)", "two suffixes (1kk)"},
		},
		{
			name: "a bracket that never closes is reported and the cell still splits",
			cell: "bpc 157 (1200, tb 500",
			want: []want{{"bpc 157", unknown}, {"tb 500", unknown}},
			bad:  []string{"bpc 157 (1200"},
		},
		{
			name: "a volume with no keyword is reported and dropped",
			cell: "(1200), tb 500",
			want: []want{{"tb 500", unknown}},
			bad:  []string{"(1200)"},
		},
		{
			name: "a keyword written twice is kept once",
			cell: "BPC 157 (500), bpc 157, tb 500",
			want: []want{{"BPC 157", 500}, {"tb 500", unknown}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			list, bad := keyword.Parse(tc.cell)
			assertList(t, list, tc.want)
			if !slices.Equal(bad, tc.bad) {
				t.Fatalf("unreadable volumes = %q, want %q", bad, tc.bad)
			}
		})
	}
}

func TestParseOfAnEmptyCellIsAnEmptyList(t *testing.T) {
	t.Parallel()

	list, bad := keyword.Parse("   ")
	if list == nil || len(list) != 0 {
		t.Fatalf("Parse of a blank cell = %#v, want an empty list that is not nil", list)
	}
	if bad != nil {
		t.Fatalf("a blank cell reported %q", bad)
	}
}

func TestCellWritesTheDocumentedForm(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		items []keyword.Keyword
		want  string
	}{
		{name: "nothing", items: nil, want: ""},
		{name: "one without a volume", items: []keyword.Keyword{{Text: "bpc-157"}}, want: "bpc-157"},
		{
			name: "volumes are plain digits",
			items: []keyword.Keyword{
				{Text: "bpc 157", Volume: new(12000)}, {Text: "bpc 157 dosage", Volume: new(1900)}, {Text: "bpc-157"},
			},
			want: "bpc 157 (12000), bpc 157 dosage (1900), bpc-157",
		},
		{name: "a zero volume is written", items: []keyword.Keyword{{Text: "rare", Volume: new(0)}}, want: "rare (0)"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := keyword.New(tc.items).Cell(); got != tc.want {
				t.Fatalf("Cell = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestACellReadsBackAsItWasWritten(t *testing.T) {
	t.Parallel()

	list := keyword.New([]keyword.Keyword{
		{Text: "bpc 157", Volume: new(12000)},
		{Text: "peptides (injectable, oral)", Volume: new(700)},
		{Text: "rare", Volume: new(0)},
		{Text: "iphone (pro)"},
		{Text: "bpc-157"},
	})

	back, bad := keyword.Parse(list.Cell())
	if bad != nil {
		t.Fatalf("the written cell reported %q", bad)
	}
	if describe(back) != describe(list) {
		t.Fatalf("read back %s, want %s", describe(back), describe(list))
	}
}
