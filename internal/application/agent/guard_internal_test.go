package agent

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func page(at int) map[string]any {
	return map[string]any{
		"id":    "page-" + strings.Repeat("0", 8) + string(rune('a'+at%26)),
		"path":  "/coffee/espresso/" + strings.Repeat("x", 24) + "/",
		"title": strings.Repeat("Espresso ", 6),
	}
}

func listing(rows int) map[string]any {
	items := make([]any, 0, rows)
	for at := range rows {
		items = append(items, page(at))
	}
	return map[string]any{"items": items, "hasMore": true, "nextCursor": "c-0001"}
}

func TestCapKeepsTheAnswerReadableAndSaysWhatItDropped(t *testing.T) {
	t.Parallel()

	capped, cut := Cap(listing(200), 2048)
	if !cut {
		t.Fatal("a listing well over the cap was not cut")
	}

	encoded, err := json.Marshal(capped)
	if err != nil {
		t.Fatalf("the capped result cannot be encoded: %v", err)
	}
	if len(encoded) > 2048 {
		t.Fatalf("the capped result is %d bytes, over the 2048 it was given", len(encoded))
	}

	var decoded struct {
		Truncated  bool           `json:"truncated"`
		TotalBytes int            `json:"totalBytes"`
		Dropped    map[string]int `json:"droppedItems"`
		Result     struct {
			Items      []map[string]any `json:"items"`
			HasMore    bool             `json:"hasMore"`
			NextCursor string           `json:"nextCursor"`
		} `json:"result"`
	}
	if unmarshalErr := json.Unmarshal(encoded, &decoded); unmarshalErr != nil {
		t.Fatalf("the capped result no longer decodes: %v", unmarshalErr)
	}

	switch {
	case !decoded.Truncated || decoded.TotalBytes == 0:
		t.Fatalf("the capped result is %s", encoded)
	case decoded.Result.NextCursor != "c-0001" || !decoded.Result.HasMore:
		t.Fatalf("the cap lost the paging of the listing: %s", encoded)
	case len(decoded.Result.Items) == 0 || len(decoded.Result.Items) >= 200:
		t.Fatalf("the cap kept %d of 200 rows", len(decoded.Result.Items))
	case decoded.Dropped["items"] != 200-len(decoded.Result.Items):
		t.Fatalf("the cap dropped %d rows and reported %v", 200-len(decoded.Result.Items), decoded.Dropped)
	}
	for _, item := range decoded.Result.Items {
		if item["id"] == nil || item["path"] == nil {
			t.Fatalf("a kept row lost its fields: %v", item)
		}
	}
}

func TestCapShortensTheLongestTextWhenThereIsNoListToThin(t *testing.T) {
	t.Parallel()

	capped, cut := Cap(map[string]any{
		"body":  strings.Repeat("espresso ", 400),
		"title": "Espresso",
	}, 1024)
	if !cut {
		t.Fatal("a long body was not cut")
	}

	var decoded struct {
		Shortened int `json:"shortenedText"`
		Result    struct {
			Body  string `json:"body"`
			Title string `json:"title"`
		} `json:"result"`
	}
	encoded, err := json.Marshal(capped)
	if err != nil {
		t.Fatalf("the capped result cannot be encoded: %v", err)
	}
	if unmarshalErr := json.Unmarshal(encoded, &decoded); unmarshalErr != nil {
		t.Fatalf("the capped result no longer decodes: %v", unmarshalErr)
	}

	switch {
	case decoded.Shortened == 0:
		t.Fatalf("the cap shortened nothing and still reports %s", encoded)
	case decoded.Result.Title != "Espresso":
		t.Fatalf("the cap touched a short field: %q", decoded.Result.Title)
	case !strings.HasSuffix(decoded.Result.Body, cutMarker):
		t.Fatalf("the shortened body does not say it was cut: %q", decoded.Result.Body)
	}
}

func TestCapLeavesWhatFits(t *testing.T) {
	t.Parallel()

	held := map[string]any{"ok": true}
	if _, cut := Cap(held, 1024); cut {
		t.Error("a small result must pass through")
	}
	if _, cut := Cap(held, 0); cut {
		t.Error("a result with no ceiling must pass through")
	}
}

func TestCutAtRuneNeverSplitsACharacter(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		text  string
		limit int
		want  string
	}{
		{name: "a multibyte character is kept whole", text: "héllo", limit: 2, want: "h"},
		{name: "a short text passes through", text: "short", limit: 50, want: "short"},
		{name: "an exact cut stands", text: "abc", limit: 2, want: "ab"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := CutAtRune(tc.text, tc.limit); got != tc.want {
				t.Fatalf("CutAtRune = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestPermitAnswersTheAllowList(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		allowed []string
		tool    string
		refused bool
	}{
		{name: "an empty list opens every tool", tool: "pages_tree"},
		{name: "a listed tool is open", allowed: []string{"pages_tree"}, tool: "pages_tree"},
		{name: "an unlisted tool is refused", allowed: []string{"sites_list"}, tool: "pages_tree", refused: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := Permit(tc.allowed, tc.tool)
			if refused := errors.IsCode(err, errors.Unauthorized); refused != tc.refused {
				t.Fatalf("Permit = %v, want refused %v", err, tc.refused)
			}
		})
	}
}
