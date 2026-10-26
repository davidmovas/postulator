package steps

import (
	"testing"

	"github.com/davidmovas/postulator/internal/domain/template"
)

func TestTheWritersCeilingMakesRoomForTheLongestADirectiveAsks(t *testing.T) {
	t.Parallel()

	spec := template.TemplateSpec{Sections: []template.Section{{TargetWords: 2000}}}
	cases := []struct {
		name     string
		maxWords int
		want     int
	}{
		{name: "no directive keeps the template's", want: 2000*tokensPerWord + tokenHeadroom},
		{name: "a shorter directive keeps the template's", maxWords: 600, want: 2000*tokensPerWord + tokenHeadroom},
		{name: "a longer directive widens it", maxWords: 3000, want: 3000*tokensPerWord + tokenHeadroom},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := writerCeiling(spec, tc.maxWords, 0); got != tc.want {
				t.Fatalf("writerCeiling = %d, want %d", got, tc.want)
			}
		})
	}
}
