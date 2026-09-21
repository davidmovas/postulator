package agent

import (
	"testing"

	"github.com/gollem-dev/gollem"
)

func TestJoinedKeepsTheWordsTheModelStreamed(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		answered *gollem.ExecuteResponse
		want     string
	}{
		{name: "nothing answered", answered: nil, want: ""},
		{name: "no texts", answered: &gollem.ExecuteResponse{}, want: ""},
		{
			name:     "one whole answer",
			answered: gollem.NewExecuteResponse("Ingredient guide"),
			want:     "Ingredient guide",
		},
		{
			name:     "the pieces a stream arrives in",
			answered: gollem.NewExecuteResponse("От", "лич", "ная", " идея", "."),
			want:     "Отличная идея.",
		},
		{
			name:     "pieces that already carry their own spacing",
			answered: gollem.NewExecuteResponse("##", " Supplement", " Ingredient", " Guide", "\n\n", "Text."),
			want:     "## Supplement Ingredient Guide\n\nText.",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := joined(tc.answered); got != tc.want {
				t.Errorf("joined = %q, want %q", got, tc.want)
			}
		})
	}
}
