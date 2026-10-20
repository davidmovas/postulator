package wptest_test

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
)

func TestTheFakeDeniesTheSharedMetaTable(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "wp-plugin", "testdata", "meta_deny.json"))
	if err != nil {
		t.Fatalf("read the meta deny table: %v", err)
	}
	var table struct {
		Cases []struct {
			Key    string `json:"key"`
			Denied bool   `json:"denied"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &table); err != nil {
		t.Fatalf("decode the meta deny table: %v", err)
	}

	server := wptest.New(t)
	page := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Holder"})[0]
	for _, tc := range table.Cases {
		if tc.Key == "" {
			continue
		}
		t.Run(tc.Key, func(t *testing.T) {
			t.Parallel()

			if !strings.Contains(tc.Key, ",") {
				response, payload := call(t, server, http.MethodGet, "/wp-json/postulator/v1/meta/"+itoa(page.ID)+"?keys="+url.QueryEscape(tc.Key), nil, true)
				if refused := response.StatusCode == http.StatusBadRequest; refused != tc.Denied {
					t.Errorf("a read of %q answers %d %s, want denied %t", tc.Key, response.StatusCode, payload, tc.Denied)
				}
			}
			body, err := json.Marshal(map[string]any{"meta": map[string]any{tc.Key: "x"}, "expected": map[string]any{tc.Key: nil}})
			if err != nil {
				t.Fatalf("encode the write: %v", err)
			}
			response, payload := call(t, server, http.MethodPut, "/wp-json/postulator/v1/meta/"+itoa(page.ID), body, true)
			if refused := response.StatusCode == http.StatusBadRequest; refused != tc.Denied {
				t.Errorf("a write of %q answers %d %s, want denied %t", tc.Key, response.StatusCode, payload, tc.Denied)
			}
		})
	}
}

func TestTheFakesMetaWriteRefusesWhatThePluginRefuses(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	page := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Holder", Meta: map[string]string{"subtitle": "Old"}})[0]
	path := "/wp-json/postulator/v1/meta/" + itoa(page.ID)
	cases := []struct {
		name   string
		body   map[string]any
		status int
		code   string
	}{
		{name: "an unguarded key", body: map[string]any{"meta": map[string]any{"subtitle": "New"}, "expected": map[string]any{}}, status: http.StatusBadRequest, code: "missing_expected"},
		{name: "a value that is no text", body: map[string]any{"meta": map[string]any{"subtitle": []string{"a"}}, "expected": map[string]any{"subtitle": nil}}, status: http.StatusBadRequest, code: "meta_key_denied"},
		{name: "no expectations at all", body: map[string]any{"meta": map[string]any{"subtitle": "New"}}, status: http.StatusBadRequest, code: "invalid_body"},
		{name: "a value read as absent", body: map[string]any{"meta": map[string]any{"subtitle": "New"}, "expected": map[string]any{"subtitle": nil}}, status: http.StatusConflict, code: "hash_mismatch"},
	}
	for _, tc := range cases {
		body, err := json.Marshal(tc.body)
		if err != nil {
			t.Fatalf("encode %s: %v", tc.name, err)
		}
		response, payload := call(t, server, http.MethodPut, path, body, true)
		var refusal struct {
			Code  string             `json:"code"`
			Moved map[string]*string `json:"moved"`
		}
		decode(t, payload, &refusal)
		if response.StatusCode != tc.status || refusal.Code != tc.code {
			t.Errorf("%s answers %d %s, want %d %s", tc.name, response.StatusCode, payload, tc.status, tc.code)
		}
		if tc.status == http.StatusConflict && refusal.Moved["subtitle"] == nil {
			t.Errorf("%s names no moved hash: %s", tc.name, payload)
		}
	}
	if held, _ := server.Lookup(page.ID); held.Meta["subtitle"] != "Old" {
		t.Errorf("a refused write changed the site: %v", held.Meta)
	}
}
