package note_test

import (
	"reflect"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/note"
)

func TestCleanKeepsOneNoteALabel(t *testing.T) {
	t.Parallel()

	got := note.Clean([]note.Note{
		{Label: " Notes ", Text: " Sold as a vial "}, {Label: "notes", Text: "a second"}, {Label: "Reason", Text: " "}, {Label: "", Text: "lost"},
	})
	want := []note.Note{{Label: "Notes", Text: "Sold as a vial"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Clean = %+v, want %+v", got, want)
	}
}

func TestMergeTakesTheFileAndDropsNothing(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		stored []note.Note
		file   []note.Note
		want   []note.Note
	}{
		{
			name:   "the file's text replaces the note of the same label",
			stored: []note.Note{{Label: "Notes", Text: "old"}, {Label: "Reason", Text: "kept"}},
			file:   []note.Note{{Label: "notes", Text: "new"}},
			want:   []note.Note{{Label: "Notes", Text: "new"}, {Label: "Reason", Text: "kept"}},
		},
		{
			name:   "a label the page lacks is added",
			stored: []note.Note{{Label: "Notes", Text: "kept"}},
			file:   []note.Note{{Label: "Intent Owner", Text: "Commercial"}},
			want:   []note.Note{{Label: "Notes", Text: "kept"}, {Label: "Intent Owner", Text: "Commercial"}},
		},
		{
			name:   "an empty cell erases nothing",
			stored: []note.Note{{Label: "Notes", Text: "kept"}},
			file:   []note.Note{{Label: "Notes", Text: ""}},
			want:   []note.Note{{Label: "Notes", Text: "kept"}},
		},
		{name: "nothing on either side", want: []note.Note{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := note.Merge(tc.stored, tc.file); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("Merge = %+v, want %+v", got, tc.want)
			}
		})
	}
}
