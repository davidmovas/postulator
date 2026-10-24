//go:build e2e

package e2e_test

import (
	"bytes"
	"encoding/json"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap/zaptest"

	"github.com/davidmovas/postulator/internal/adapters/llm/fake"
	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/app"
	"github.com/davidmovas/postulator/internal/application/imports"
	llmport "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/application/pagetypes"
	"github.com/davidmovas/postulator/internal/application/runs"
	"github.com/davidmovas/postulator/internal/application/sites"
	appsync "github.com/davidmovas/postulator/internal/application/sync"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

type wireRecord struct {
	Method string
	Path   string
	Body   []byte
}

type recordingProxy struct {
	server  *httptest.Server
	records []wireRecord
	mu      sync.Mutex
}

func startRecordingProxy(t *testing.T) *recordingProxy {
	t.Helper()

	proxy := &recordingProxy{}
	direct := &http.Transport{}
	proxy.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		proxy.mu.Lock()
		proxy.records = append(proxy.records, wireRecord{Method: r.Method, Path: r.URL.Path, Body: body})
		proxy.mu.Unlock()
		forwarded, err := http.NewRequestWithContext(r.Context(), r.Method, r.URL.String(), bytes.NewReader(body))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		forwarded.Header = r.Header.Clone()
		response, err := direct.RoundTrip(forwarded)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer func() {
			if closeErr := response.Body.Close(); closeErr != nil {
				t.Errorf("close the forwarded answer: %v", closeErr)
			}
		}()
		maps.Copy(w.Header(), response.Header)
		w.WriteHeader(response.StatusCode)
		if _, copyErr := io.Copy(w, response.Body); copyErr != nil {
			t.Errorf("copy the forwarded answer: %v", copyErr)
		}
	}))
	t.Cleanup(proxy.server.Close)
	return proxy
}

func (p *recordingProxy) writes() []wireRecord {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]wireRecord, 0, len(p.records))
	for _, record := range p.records {
		if record.Method != http.MethodGet && record.Method != http.MethodHead {
			out = append(out, record)
		}
	}
	return out
}

func openCoreThrough(t *testing.T, provider llmport.Client, proxy *recordingProxy) *app.Core {
	t.Helper()

	home := t.TempDir()
	core, err := app.Open(t.Context(), app.Config{
		DatabasePath: filepath.Join(home, "postulator.db"),
		KeyDir:       home,
		Provider:     provider,
		WordPress:    []wp.Option{wp.WithProxy(proxy.server.URL)},
	}, zaptest.NewLogger(t))
	if err != nil {
		t.Fatalf("open the application: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := core.Close(); closeErr != nil {
			t.Errorf("close the application: %v", closeErr)
		}
	})
	keyTheProviders(t, core)
	return core
}

type storeFacts struct {
	Name          string           `json:"name"`
	Slug          string           `json:"slug"`
	Status        string           `json:"status"`
	SKU           string           `json:"sku"`
	RegularPrice  string           `json:"regular_price"`
	SalePrice     string           `json:"sale_price"`
	StockStatus   string           `json:"stock_status"`
	ManageStock   any              `json:"manage_stock"`
	StockQuantity any              `json:"stock_quantity"`
	Attributes    []storeAttribute `json:"attributes"`
	Categories    []struct {
		ID int `json:"id"`
	} `json:"categories"`
}

type storeVariation struct {
	Description  string `json:"description"`
	SKU          string `json:"sku"`
	RegularPrice string `json:"regular_price"`
	StockStatus  string `json:"stock_status"`
	Attributes   []struct {
		Name   string `json:"name"`
		Option string `json:"option"`
	} `json:"attributes"`
	ID int `json:"id"`
}

type editedProduct struct {
	Description      string `json:"description"`
	ShortDescription string `json:"short_description"`
	MetaData         []struct {
		Key   string `json:"key"`
		Value any    `json:"value"`
	} `json:"meta_data"`
}

func (p editedProduct) meta(key string) string {
	for _, entry := range p.MetaData {
		if text, ok := entry.Value.(string); entry.Key == key && ok {
			return text
		}
	}
	return ""
}

func (s *site) variationsOf(t *testing.T, productID int) []storeVariation {
	t.Helper()

	var listed []storeVariation
	s.call(t, http.MethodGet, "/wp-json/wc/v3/products/"+strconv.Itoa(productID)+"/variations?context=edit&per_page=100", nil, http.StatusOK, &listed)
	slices.SortFunc(listed, func(a, b storeVariation) int { return a.ID - b.ID })
	return listed
}

func (s *site) factsOf(t *testing.T, productID int) (storeFacts, []storeVariation) {
	t.Helper()

	var facts storeFacts
	s.call(t, http.MethodGet, "/wp-json/wc/v3/products/"+strconv.Itoa(productID)+"?context=edit", nil, http.StatusOK, &facts)
	variations := s.variationsOf(t, productID)
	for i := range variations {
		variations[i].Description = ""
	}
	return facts, variations
}

func (s *site) edited(t *testing.T, productID int) editedProduct {
	t.Helper()

	var held editedProduct
	s.call(t, http.MethodGet, "/wp-json/wc/v3/products/"+strconv.Itoa(productID)+"?context=edit", nil, http.StatusOK, &held)
	return held
}

var deniedOnTheWire = []string{
	"name", "slug", "status", "sku", "price", "regular_price", "sale_price", "stock_status", "stock_quantity", "manage_stock",
	"categories", "tags", "attributes", "type", "parent_id", "menu_order",
}

func TestAProductIsEditedThroughTheRegistry(t *testing.T) {
	live := newSite(t)
	requirePlugin(t, live.env, true)
	requireStore(t, live)

	tag := strconv.FormatInt(time.Now().Unix()%1_000_000, 10)
	slug := "registry-vial-" + tag
	var planted struct {
		ID int `json:"id"`
	}
	live.call(t, http.MethodPost, "/wp-json/wc/v3/products", map[string]any{
		"name": "Registry Vial " + tag, "slug": slug, "type": "variable", "status": "publish", "sku": slug,
		"description": clientDescription, "short_description": clientShort,
		"attributes": []map[string]any{{"name": "Size", "options": []string{"5 mg", "10 mg"}, "visible": true, "variation": true}},
		"meta_data":  []map[string]any{{"key": "dosage_note", "value": "The client's own note."}},
	}, http.StatusCreated, &planted)
	live.removeLater(t, "/wp-json/wc/v3/products/"+strconv.Itoa(planted.ID)+"?force=true")
	variationIDs := make([]int, 0, 2)
	for _, size := range []string{"5 mg", "10 mg"} {
		var variation struct {
			ID int `json:"id"`
		}
		live.call(t, http.MethodPost, "/wp-json/wc/v3/products/"+strconv.Itoa(planted.ID)+"/variations", map[string]any{
			"regular_price": "19.90", "sku": slug + "-" + strings.Fields(size)[0], "description": "<p>The client's " + size + ".</p>",
			"attributes": []map[string]any{{"name": "Size", "option": size}},
		}, http.StatusCreated, &variation)
		variationIDs = append(variationIDs, variation.ID)
	}
	productPath := "/product/" + slug + "/"
	beforeFacts, beforeVariations := live.factsOf(t, planted.ID)
	clientVariations := live.variationsOf(t, planted.ID)

	fieldsAnswer := func(llmport.Request) string {
		values := make([]map[string]string, 0, 2+len(variationIDs))
		values = append(values,
			map[string]string{"key": string(field.ProductShortDescription), "value": "A registry vial of peptide, measured by the dropper."},
			map[string]string{"key": "meta:dosage_note", "value": "Once a day, as the label says."},
		)
		for i, id := range variationIDs {
			values = append(values, map[string]string{
				"key": string(field.Instance(field.VariationDescription, strconv.Itoa(id))), "value": "The " + []string{"five", "ten"}[i] + " milligram vial.",
			})
		}
		encoded, err := json.Marshal(map[string]any{"fields": values})
		if err != nil {
			t.Errorf("encode the fields answer: %v", err)
		}
		return string(encoded)
	}
	proxy := startRecordingProxy(t)
	core := openCoreThrough(t, fake.NewScripted(append(replies(),
		fake.Reply{Step: steps.NameGenerateBody, Match: "Path: " + productPath + "\n", Text: productAnswer("registry vial")},
		fake.Reply{Step: steps.NameGenerateMeta, Match: "Path: " + productPath + "\n", Text: metaOf("registry vial")},
		fake.Reply{Step: steps.NameGenerateFields, Match: "Path: " + productPath + "\n", Make: fieldsAnswer},
	)...), proxy)
	owner, err := core.Sites.Create(t.Context(), sites.CreateRequest{
		Name: "Registry Store", BaseURL: live.env.baseURL, Username: live.env.user, Password: live.env.pass, AllowInsecure: true,
	})
	if err != nil {
		t.Fatalf("create the site: %v", err)
	}
	siteID := owner.Site.ID
	if checked, checkErr := core.Sync.CheckPlugin(t.Context(), appsync.CheckPluginRequest{SiteID: siteID}); checkErr != nil || checked.Commerce != "ready" {
		t.Fatalf("the check reports %+v (%v), want a store that can be edited", checked, checkErr)
	}

	listed, err := core.PageTypes.List(t.Context(), pagetypes.ListRequest{SiteID: siteID})
	if err != nil {
		t.Fatalf("list the page types: %v", err)
	}
	productType := ""
	for i := range listed.Types {
		if pagetype.Preset(listed.Types[i].Preset) == pagetype.Product {
			productType = listed.Types[i].ID
		}
	}
	for _, scope := range []pagetypes.SetScopeRequest{
		{PageTypeID: productType, Field: string(field.VariationDescription), Enabled: true, AIWrites: true, Links: field.NoLinks()},
		{
			PageTypeID: productType, Field: "meta:dosage_note", Enabled: true, AIWrites: true, Label: "Dosage note",
			Provider: string(field.ProviderStore), Links: field.NoLinks(),
		},
	} {
		if _, err = core.PageTypes.SetScope(t.Context(), scope); err != nil {
			t.Fatalf("set the scope of %s: %v", scope.Field, err)
		}
	}

	synced, err := core.Sync.SyncSite(t.Context(), appsync.SyncSiteRequest{SiteID: siteID})
	if err != nil {
		t.Fatalf("sync the site: %v", err)
	}
	awaitRun(t, core.Runs, synced.RunID)
	sheet := filepath.Join(t.TempDir(), "registry.csv")
	if err = os.WriteFile(sheet, []byte("path,h1,keywords\n"+productPath+",Registry Vial "+tag+",registry vial\n"), 0o600); err != nil {
		t.Fatalf("write the sheet: %v", err)
	}
	if applied, applyErr := core.Imports.Apply(t.Context(), imports.ApplyRequest{SiteID: siteID, Path: sheet, Mapping: imports.Mapping{
		SiteID: siteID, Name: "registry", Columns: map[string]string{
			string(importmap.FieldPath): "path", string(importmap.FieldH1): "h1", string(importmap.FieldKeywords): "keywords",
		}, Options: imports.Options{RowType: importmap.RowProducts},
	}}); applyErr != nil || len(applied.Report.Errors) != 0 {
		t.Fatalf("apply the sheet: %v %+v", applyErr, applied.Report.Errors)
	}
	page, ok := pagesByPath(t, core.Pages, siteID)[productPath]
	if !ok || page.WPID == nil {
		t.Fatalf("the product at %s is not in the model: %+v", productPath, page)
	}

	started, err := core.Runs.Start(t.Context(), runs.StartRequest{
		SiteID: siteID, PageIDs: []string{page.ID}, Rewrite: []string{page.ID},
		TemplateID: productTemplate(t, core), PublishMode: string(run.PublishLive), Recipe: recipe(),
	})
	if err != nil {
		t.Fatalf("start the run: %v", err)
	}
	awaitRun(t, core.Runs, started.RunID)

	written := live.edited(t, planted.ID)
	if !strings.Contains(written.ShortDescription, "measured by the dropper") || written.meta("dosage_note") != "Once a day, as the label says." ||
		strings.Contains(written.Description, clientDescription) {
		t.Errorf("the product holds %q / %q / %q, want the run's short description, custom field and description",
			written.ShortDescription, written.meta("dosage_note"), written.Description)
	}
	for i, variation := range live.variationsOf(t, planted.ID) {
		if !strings.Contains(variation.Description, "milligram vial") {
			t.Errorf("variation %d holds %q, want the run's description", i, variation.Description)
		}
	}

	wireKeys := make([]string, 0)
	for _, record := range proxy.writes() {
		if !strings.Contains(record.Path, "/wc/v3/products") {
			continue
		}
		var body map[string]any
		if err = json.Unmarshal(record.Body, &body); err != nil {
			t.Errorf("a store write sent a body that is not JSON: %s", record.Body)
			continue
		}
		keys := slices.Sorted(maps.Keys(body))
		wireKeys = append(wireKeys, record.Method+" "+record.Path+" "+strings.Join(keys, ","))
		for _, denied := range deniedOnTheWire {
			if slices.Contains(keys, denied) {
				t.Errorf("%s %s sent %s, which the store owns: %s", record.Method, record.Path, denied, record.Body)
			}
		}
		if entries, sent := body["meta_data"].([]any); sent {
			for _, entry := range entries {
				if held, isObject := entry.(map[string]any); !isObject || held["key"] != "dosage_note" {
					t.Errorf("a meta_data write named %v, want the custom field alone", entry)
				}
			}
		}
	}
	afterFacts, afterVariations := live.factsOf(t, planted.ID)
	if !reflect.DeepEqual(beforeFacts, afterFacts) || !reflect.DeepEqual(beforeVariations, afterVariations) {
		t.Errorf("the store's own facts moved:\nbefore %+v %+v\nafter  %+v %+v", beforeFacts, beforeVariations, afterFacts, afterVariations)
	}
	t.Logf("written: short description, meta:dosage_note, %d variation descriptions, the description; the wire's store writes, "+
		"each request's keys: %s; never sent: %s", len(variationIDs), strings.Join(wireKeys, " | "), strings.Join(deniedOnTheWire, ", "))

	reverted, err := core.Runs.Revert(t.Context(), runs.RevertRequest{RunID: started.RunID})
	if err != nil {
		t.Fatalf("revert the run: %v", err)
	}
	awaitRun(t, core.Runs, reverted.RunID)
	back := live.edited(t, planted.ID)
	if back.Description != clientDescription || back.ShortDescription != clientShort || back.meta("dosage_note") != "The client's own note." {
		t.Errorf("after the revert the product holds %q / %q / %q, want the client's", back.Description, back.ShortDescription, back.meta("dosage_note"))
	}
	for i, variation := range live.variationsOf(t, planted.ID) {
		want := clientVariations[i].Description
		if !strings.Contains(want, []string{"5 mg", "10 mg"}[i]) || variation.Description != want {
			t.Errorf("after the revert variation %d holds %q, want %q, what the store answered before the run", i, variation.Description, want)
		}
	}
	if finalFacts, finalVariations := live.factsOf(t, planted.ID); !reflect.DeepEqual(beforeFacts, finalFacts) || !reflect.DeepEqual(beforeVariations, finalVariations) {
		t.Errorf("the revert moved the store's own facts:\nbefore %+v %+v\nafter  %+v %+v", beforeFacts, beforeVariations, finalFacts, finalVariations)
	}
}
