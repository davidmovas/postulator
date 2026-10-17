package wptest

import (
	"net/http"
	"strconv"
)

type MetaEntry struct {
	Key   string
	Value any
	ID    int64
}

type Choice struct {
	Name   string
	Option string
	ID     int64
}

type StorePages struct {
	Shop     int64
	Cart     int64
	Checkout int64
	Account  int64
	Terms    int64
}

func WithoutStoreSettingsAccess() Option {
	return func(s *Server) { s.noStoreSettings = true }
}

func (s *Server) SetStorePages(pages StorePages) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.storePages = pages
}

func (s *Server) routeStore(mux *http.ServeMux) {
	mux.HandleFunc("GET "+wooNamespace+"/products/{id}/{tail...}", s.withCommerce(s.handleProductTail))
	mux.HandleFunc("GET "+wooNamespace+"/settings/products/woocommerce_shop_page_id", s.withCommerce(s.handleShopPageSetting))
	mux.HandleFunc("GET "+wooNamespace+"/settings/advanced", s.withCommerce(s.handleAdvancedSettings))
}

func (s *Server) handleProductTail(w http.ResponseWriter, r *http.Request) {
	if r.PathValue("tail") != "variations" {
		s.fail(w, http.StatusNotFound, "rest_no_route", "No route was found matching the URL and request method.")
		return
	}
	id, ok := pathID(r)
	if !ok {
		s.fail(w, http.StatusNotFound, "woocommerce_rest_product_invalid_id", "Invalid ID.")
		return
	}
	query := r.URL.Query()
	page, perPage, bad := listWindow(query)
	if bad != "" {
		s.fail(w, http.StatusBadRequest, "woocommerce_rest_invalid_param", "Invalid parameter(s): "+bad)
		return
	}
	edit := query.Get("context") == "edit"

	s.mu.Lock()
	parent, found := s.items[id]
	if !found || parent.Type != TypeProduct {
		s.mu.Unlock()
		s.fail(w, http.StatusNotFound, "woocommerce_rest_product_invalid_id", "Invalid ID.")
		return
	}
	matched := s.variationsOf(parent.ID)
	total := len(matched)
	totalPages := (total + perPage - 1) / perPage
	start := min((page-1)*perPage, total)
	end := min(start+perPage, total)
	payload := make([]map[string]any, 0, end-start)
	for _, stored := range matched[start:end] {
		payload = append(payload, narrowFields(s.variationPayload(stored, parent, edit), query.Get("_fields")))
	}
	s.mu.Unlock()

	w.Header().Set("X-WP-Total", strconv.Itoa(total))
	w.Header().Set("X-WP-TotalPages", strconv.Itoa(totalPages))
	s.respond(w, http.StatusOK, payload)
}

func (s *Server) variationsOf(productID int64) []*Item {
	matched := make([]*Item, 0)
	for _, id := range s.order {
		stored := s.items[id]
		if stored.Type == TypeProductVariation && stored.Parent == productID && stored.Status != "trash" {
			matched = append(matched, stored)
		}
	}
	return matched
}

func (s *Server) variationIDs(productID int64) []int64 {
	variations := s.variationsOf(productID)
	ids := make([]int64, 0, len(variations))
	for _, stored := range variations {
		ids = append(ids, stored.ID)
	}
	return ids
}

func (s *Server) variationPayload(stored, parent *Item, edit bool) map[string]any {
	var manageStock any = stored.ManageStock
	if !stored.ManageStock && parent.ManageStock {
		manageStock = "parent"
	}
	attributes := make([]map[string]any, 0, len(stored.Choices))
	for _, choice := range stored.Choices {
		attributes = append(attributes, map[string]any{"id": choice.ID, "name": choice.Name, "option": choice.Option})
	}
	return map[string]any{
		"id":                stored.ID,
		"parent_id":         parent.ID,
		"sku":               stored.SKU,
		"price":             priceOf(stored),
		"regular_price":     stored.RegularPrice,
		"sale_price":        stored.SalePrice,
		"stock_status":      stockStatusOf(stored),
		"manage_stock":      manageStock,
		"stock_quantity":    stored.StockQuantity,
		"description":       rendered(stored.Content, edit),
		"permalink":         s.permalink(parent),
		"status":            stored.Status,
		"date_modified_gmt": stored.Modified.UTC().Format(wpTimeLayout),
		"attributes":        attributes,
		"meta_data":         metaDataPayload(stored.MetaData),
	}
}

func priceOf(stored *Item) string {
	if stored.SalePrice != "" {
		return stored.SalePrice
	}
	return stored.RegularPrice
}

func stockStatusOf(stored *Item) string {
	if stored.StockStatus == "" {
		return "instock"
	}
	return stored.StockStatus
}

func metaDataPayload(entries []MetaEntry) []map[string]any {
	payload := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		payload = append(payload, map[string]any{"id": entry.ID, "key": entry.Key, "value": entry.Value})
	}
	return payload
}

func pageSetting(id int64) string {
	if id <= 0 {
		return ""
	}
	return strconv.FormatInt(id, 10)
}

func (s *Server) storeSettingsRefused(w http.ResponseWriter) bool {
	s.mu.Lock()
	refused := s.noStoreSettings
	s.mu.Unlock()

	if refused {
		s.fail(w, http.StatusForbidden, "woocommerce_rest_cannot_view", "Sorry, you cannot list resources.")
	}
	return refused
}

func (s *Server) handleShopPageSetting(w http.ResponseWriter, _ *http.Request) {
	if s.storeSettingsRefused(w) {
		return
	}
	s.mu.Lock()
	shop := s.storePages.Shop
	s.mu.Unlock()

	s.respond(w, http.StatusOK, map[string]any{
		"id":    "woocommerce_shop_page_id",
		"label": "Shop page",
		"type":  "single_select_page",
		"value": pageSetting(shop),
	})
}

func (s *Server) handleAdvancedSettings(w http.ResponseWriter, _ *http.Request) {
	if s.storeSettingsRefused(w) {
		return
	}
	s.mu.Lock()
	pages := s.storePages
	s.mu.Unlock()

	settings := make([]map[string]any, 0, 4)
	for _, setting := range []struct {
		id string
		at int64
	}{
		{id: "woocommerce_cart_page_id", at: pages.Cart},
		{id: "woocommerce_checkout_page_id", at: pages.Checkout},
		{id: "woocommerce_myaccount_page_id", at: pages.Account},
		{id: "woocommerce_terms_page_id", at: pages.Terms},
	} {
		settings = append(settings, map[string]any{"id": setting.id, "type": "single_select_page", "value": pageSetting(setting.at)})
	}
	s.respond(w, http.StatusOK, settings)
}
