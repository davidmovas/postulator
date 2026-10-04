package pagemap_test

import (
	"reflect"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

func TestNewNotesKeepsOneNoteALabel(t *testing.T) {
	t.Parallel()

	got := pagemap.NewNotes([]pagemap.Note{
		{Label: " Notes ", Text: " Sold as a vial "}, {Label: "notes", Text: "a second"}, {Label: "Reason", Text: " "}, {Label: "", Text: "lost"},
	})
	want := []pagemap.Note{{Label: "Notes", Text: "Sold as a vial"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("NewNotes = %+v, want %+v", got, want)
	}
}

func TestMergeNotesTakesTheFileAndDropsNothing(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		stored []pagemap.Note
		file   []pagemap.Note
		want   []pagemap.Note
	}{
		{
			name:   "the file's text replaces the note of the same label",
			stored: []pagemap.Note{{Label: "Notes", Text: "old"}, {Label: "Reason", Text: "kept"}},
			file:   []pagemap.Note{{Label: "notes", Text: "new"}},
			want:   []pagemap.Note{{Label: "Notes", Text: "new"}, {Label: "Reason", Text: "kept"}},
		},
		{
			name:   "a label the page lacks is added",
			stored: []pagemap.Note{{Label: "Notes", Text: "kept"}},
			file:   []pagemap.Note{{Label: "Intent Owner", Text: "Commercial"}},
			want:   []pagemap.Note{{Label: "Notes", Text: "kept"}, {Label: "Intent Owner", Text: "Commercial"}},
		},
		{
			name:   "an empty cell erases nothing",
			stored: []pagemap.Note{{Label: "Notes", Text: "kept"}},
			file:   []pagemap.Note{{Label: "Notes", Text: ""}},
			want:   []pagemap.Note{{Label: "Notes", Text: "kept"}},
		},
		{name: "nothing on either side", want: []pagemap.Note{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := pagemap.MergeNotes(tc.stored, tc.file); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("MergeNotes = %+v, want %+v", got, tc.want)
			}
		})
	}
}
