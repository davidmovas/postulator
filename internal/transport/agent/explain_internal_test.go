package agent

import (
	stderrors "errors"
	"testing"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestExplainTellsTheModelWhichFieldItGotWrong(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "an error the kernel does not own is left alone",
			err:  stderrors.New("the provider hung up"),
			want: "the provider hung up",
		},
		{
			name: "an error without details is left alone",
			err:  errors.New(errors.Invalid, "the anchor source is not recognized"),
			want: "the anchor source is not recognized",
		},
		{
			name: "the field the domain named reaches the model",
			err: errors.New(errors.Invalid, "the anchor source is not recognized").
				WithDetail("field", "anchors[0].source"),
			want: "the anchor source is not recognized (field: anchors[0].source)",
		},
		{
			name: "several details are named in a settled order",
			err: errors.New(errors.Invalid, "the site refused the write").
				WithDetail("field", "path").WithDetail("code", "plugin_missing"),
			want: "the site refused the write (code: plugin_missing, field: path)",
		},
		{
			name: "a detail that carries a credential is masked",
			err: errors.New(errors.Invalid, "the provider refused the key").
				WithDetail("apiKey", "sk-live-1234").WithDetail("provider", "openai"),
			want: "the provider refused the key (apiKey: ***, provider: openai)",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := explain(tc.err).Error(); got != tc.want {
				t.Errorf("explain = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestExplainKeepsTheCodeTheGuardReadsBack(t *testing.T) {
	t.Parallel()

	denied := errors.New(errors.Unauthorized, "this tool works inside one site").WithDetail("tool", "pages_create")
	if !errors.IsCode(explain(denied), errors.Unauthorized) {
		t.Fatalf("explain dropped the code the audit reads to tell a denial from a failure")
	}
}
