//go:build e2e

package e2e_test

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

func metaPath(id int, keys string) string {
	return fmt.Sprintf("/wp-json/postulator/v1/meta/%d?keys=%s", id, url.QueryEscape(keys))
}

type metaValues struct {
	Meta map[string]any `json:"meta"`
	ID   int            `json:"id"`
}

type metaRefusal struct {
	Code string `json:"code"`
	Data struct {
		Keys []string `json:"keys"`
	} `json:"data"`
}

func TestMetaReadsOnlyTheKeysAsked(t *testing.T) {
	c, env := newClient(t)
	stand := standOf(t, env)

	page := createPage(t, c, pageSpec{title: "Meta holder", slug: uniqueSlug("meta"), content: "<p>x</p>"})
	wpCLI(t, env, "eval", fmt.Sprintf("update_post_meta(%d, 'color', 'red'); update_post_meta(%d, 'size', '10'); update_post_meta(%d, '_secret', 'x');", page, page, page))

	var one metaValues
	c.expect(t, http.MethodGet, metaPath(page, "color"), nil, http.StatusOK, &one)
	if one.ID != page || len(one.Meta) != 1 || one.Meta["color"] != "red" {
		t.Errorf("keys=color answers %+v, want color alone", one)
	}

	var two metaValues
	c.expect(t, http.MethodGet, metaPath(page, "color, size,missing"), nil, http.StatusOK, &two)
	missing, present := two.Meta["missing"]
	if len(two.Meta) != 3 || two.Meta["size"] != "10" || !present || missing != nil {
		t.Errorf("keys=color,size,missing answers %+v, want the two values and a null", two)
	}

	var denied metaRefusal
	c.expect(t, http.MethodGet, metaPath(page, "color,_secret"), nil, http.StatusBadRequest, &denied)
	if denied.Code != "meta_key_denied" || len(denied.Data.Keys) != 1 || denied.Data.Keys[0] != "_secret" {
		t.Errorf("a protected key answers %+v, want meta_key_denied naming _secret", denied)
	}

	if env.woo {
		product := findBySlug(t, c, "product", stand.simpleProduct)
		var store metaRefusal
		c.expect(t, http.MethodGet, metaPath(product.ID, "_price"), nil, http.StatusBadRequest, &store)
		if store.Code != "meta_key_denied" {
			t.Errorf("a product's _price answers %+v, want meta_key_denied", store)
		}
	}

	keys := make([]string, 51)
	for index := range keys {
		keys[index] = "key" + strconv.Itoa(index)
	}
	var crowded metaRefusal
	c.expect(t, http.MethodGet, metaPath(page, strings.Join(keys, ",")), nil, http.StatusBadRequest, &crowded)
	if crowded.Code != "too_many_keys" {
		t.Errorf("51 keys answer %+v, want too_many_keys", crowded)
	}

	var empty metaRefusal
	c.expect(t, http.MethodGet, metaPath(page, " , "), nil, http.StatusBadRequest, &empty)
	if empty.Code != "invalid_param" {
		t.Errorf("no key answers %+v, want invalid_param", empty)
	}

	if status, _, _ := c.anonymous(t, env.baseURL+metaPath(page, "color")); status != http.StatusUnauthorized {
		t.Errorf("an anonymous read = %d, want 401", status)
	}
}
