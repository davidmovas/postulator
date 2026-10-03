package category_test

import (
	stderrors "errors"
	"slices"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/domain/category"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	siteA = "0b6c2a4e-1f3d-4c8b-9a2e-5d7f8e9a0b1c"
	catA  = "1a1a1a1a-1a1a-4a1a-8a1a-1a1a1a1a1a1a"
	catB  = "2b2b2b2b-2b2b-4b2b-8b2b-2b2b2b2b2b2b"
	catC  = "3c3c3c3c-3c3c-4c3c-8c3c-3c3c3c3c3c3c"
	catD  = "4d4d4d4d-4d4d-4d4d-8d4d-4d4d4d4d4d4d"
	catE  = "5e5e5e5e-5e5e-4e5e-8e5e-5e5e5e5e5e5e"
)

var stamp = time.Date(2026, time.October, 4, 9, 0, 0, 0, time.UTC)

func fieldOf(t *testing.T, err error) string {
	t.Helper()
	var kernel *errors.Error
	if !stderrors.As(err, &kernel) {
		t.Fatalf("error %v is not a kernel error", err)
	}
	field, ok := kernel.Details["field"].(string)
	if !ok {
		t.Fatalf("error %v carries no field detail", err)
	}
	return field
}

func node(id, name, parentID string) category.Category {
	return category.Category{ID: id, SiteID: siteA, Name: name, Key: category.Key(name), ParentID: parentID, CreatedAt: stamp, UpdatedAt: stamp}
}

func ids(chain []category.Category) []string {
	out := make([]string, 0, len(chain))
	for i := range chain {
		out = append(out, chain[i].ID)
	}
	return out
}

func TestNewTrimsTheNameAndComputesTheKey(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		in       category.Category
		wantName string
		wantKey  string
	}{
		{
			name:     "a top-level category",
			in:       category.Category{ID: catA, SiteID: siteA, Name: "  Healing   Peptides ", CreatedAt: stamp, UpdatedAt: stamp},
			wantName: "Healing   Peptides", wantKey: "healing peptides",
		},
		{
			name:     "a category under another",
			in:       category.Category{ID: catB, SiteID: siteA, Name: "Tools &amp; Kits", ParentID: catA, CreatedAt: stamp, UpdatedAt: stamp},
			wantName: "Tools &amp; Kits", wantKey: "tools & kits",
		},
		{
			name:     "a key the caller set is replaced",
			in:       category.Category{ID: catC, SiteID: siteA, Name: "Liquid", Key: "stale", ParentID: catB},
			wantName: "Liquid", wantKey: "liquid",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := category.New(tc.in)
			if err != nil {
				t.Fatalf("New: %v", err)
			}
			want := tc.in
			want.Name = tc.wantName
			want.Key = tc.wantKey
			if got != want {
				t.Fatalf("New = %+v, want %+v", got, want)
			}
		})
	}
}

func TestNewRejects(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		mutate func(*category.Category)
		field  string
	}{
		{name: "no id", mutate: func(c *category.Category) { c.ID = "" }, field: "id"},
		{name: "no site", mutate: func(c *category.Category) { c.SiteID = "" }, field: "siteId"},
		{name: "an empty name", mutate: func(c *category.Category) { c.Name = "" }, field: "name"},
		{name: "a blank name", mutate: func(c *category.Category) { c.Name = " \t " }, field: "name"},
		{name: "a name that decodes to a space", mutate: func(c *category.Category) { c.Name = "&nbsp;" }, field: "name"},
		{name: "a category under itself", mutate: func(c *category.Category) { c.ParentID = c.ID }, field: "parentId"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			c := node(catB, "Healing", catA)
			tc.mutate(&c)
			_, err := category.New(c)
			if !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("code = %q, want INVALID", errors.CodeOf(err))
			}
			if got := fieldOf(t, err); got != tc.field {
				t.Errorf("field = %q, want %q", got, tc.field)
			}
		})
	}
}

func TestChainWalksFromTheRootToTheLeaf(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		all  []category.Category
		leaf string
		want []string
	}{
		{
			name: "a top-level leaf is its own chain",
			all:  []category.Category{node(catA, "Healing", "")},
			leaf: catA, want: []string{catA},
		},
		{
			name: "three levels come root first",
			all: []category.Category{
				node(catC, "Liquid", catB), node(catA, "Healing", ""), node(catB, "BPC-157", catA),
			},
			leaf: catC, want: []string{catA, catB, catC},
		},
		{
			name: "a middle leaf stops at itself",
			all: []category.Category{
				node(catA, "Healing", ""), node(catB, "BPC-157", catA), node(catC, "Liquid", catB),
			},
			leaf: catB, want: []string{catA, catB},
		},
		{
			name: "a sibling branch is not walked",
			all: []category.Category{
				node(catA, "Healing", ""), node(catB, "BPC-157", catA), node(catD, "TB-500", catA), node(catC, "Liquid", catD),
			},
			leaf: catC, want: []string{catA, catD, catC},
		},
		{
			name: "a parent that is not among them ends the walk",
			all: []category.Category{
				node(catB, "BPC-157", catE), node(catC, "Liquid", catB),
			},
			leaf: catC, want: []string{catB, catC},
		},
		{
			name: "a loop stops at the first category seen twice",
			all: []category.Category{
				node(catA, "Healing", catB), node(catB, "BPC-157", catA), node(catC, "Liquid", catA),
			},
			leaf: catC, want: []string{catB, catA, catC},
		},
		{
			name: "a leaf that is not among them has no chain",
			all:  []category.Category{node(catA, "Healing", "")},
			leaf: catE, want: []string{},
		},
		{
			name: "no leaf has no chain",
			all:  []category.Category{node(catA, "Healing", "")},
			leaf: "", want: []string{},
		},
		{
			name: "no categories have no chain",
			all:  nil,
			leaf: catA, want: []string{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			chain := category.Chain(tc.all, tc.leaf)
			if chain == nil {
				t.Fatal("Chain = nil, want a list")
			}
			if got := ids(chain); !slices.Equal(got, tc.want) {
				t.Fatalf("Chain(%s) = %v, want %v", tc.leaf, got, tc.want)
			}
		})
	}
}

func TestChainCarriesCopiesOfTheCategories(t *testing.T) {
	t.Parallel()

	all := []category.Category{node(catA, "Healing", ""), node(catB, "BPC-157", catA)}
	chain := category.Chain(all, catB)
	if len(chain) != 2 || chain[0] != all[0] || chain[1] != all[1] {
		t.Fatalf("Chain = %+v, want the categories root first", chain)
	}

	chain[0].Name = "Renamed"
	if all[0].Name != "Healing" {
		t.Fatal("writing to a chain changed the categories it was read from")
	}
}
