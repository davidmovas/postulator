package wptest_test

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
)

func TestTheMetaRouteAnswersOnlyTheKeysAsked(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	page := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Holder", Meta: map[string]string{"color": "red", "size": "10", "_secret": "x"}})[0]
	path := func(keys string) string {
		return "/wp-json/postulator/v1/meta/" + itoa(page.ID) + "?keys=" + url.QueryEscape(keys)
	}

	response, payload := call(t, server, http.MethodGet, path("color,missing"), nil, true)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d: %s", response.StatusCode, payload)
	}
	var got struct {
		Meta map[string]*string `json:"meta"`
		ID   int64              `json:"id"`
	}
	decode(t, payload, &got)
	if got.ID != page.ID || len(got.Meta) != 2 || got.Meta["color"] == nil || *got.Meta["color"] != "red" || got.Meta["missing"] != nil {
		t.Errorf("meta = %s", payload)
	}

	many := make([]string, 51)
	for index := range many {
		many[index] = "key" + strconv.Itoa(index)
	}
	refusals := []struct {
		keys string
		code string
	}{
		{keys: "color,_secret", code: "meta_key_denied"},
		{keys: strings.Join(many, ","), code: "too_many_keys"},
		{keys: " , ", code: "invalid_param"},
	}
	for _, tc := range refusals {
		response, payload = call(t, server, http.MethodGet, path(tc.keys), nil, true)
		var refusal struct {
			Code string `json:"code"`
		}
		decode(t, payload, &refusal)
		if response.StatusCode != http.StatusBadRequest || refusal.Code != tc.code {
			t.Errorf("keys=%.20s answers %d %s, want 400 %s", tc.keys, response.StatusCode, payload, tc.code)
		}
	}
}
