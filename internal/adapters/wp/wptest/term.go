package wptest

import (
	"cmp"
	"html"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
)

const (
	taxonomyCategory     = "category"
	duplicateTermMessage = "A term with the name provided already exists with this parent."
)

var termEscaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")

type termRef struct {
	Name   string
	Slug   string
	ID     int64
	Parent int64
}

type refusal struct {
	code    string
	message string
	status  int
}

func (r refusal) body(extra map[string]any) map[string]any {
	data := map[string]any{"status": r.status}
	for key, value := range extra {
		data[key] = value
	}
	return map[string]any{"code": r.code, "message": r.message, "data": data}
}

type termRules struct {
	forbidden refusal
	parent    refusal
	empty     refusal
	idKey     string
	invalid   string
}

var termRulesOf = map[string]termRules{
	taxonomyCategory: {
		forbidden: refusal{status: http.StatusForbidden, code: "rest_cannot_create", message: "Sorry, you are not allowed to create terms in this taxonomy."},
		parent:    refusal{status: http.StatusBadRequest, code: "rest_term_invalid", message: "Parent term does not exist."},
		empty:     refusal{status: http.StatusInternalServerError, code: "empty_term_name", message: "A name is required for this term."},
		idKey:     "term_id",
		invalid:   "rest_invalid_param",
	},
	TypeProductCategory: {
		forbidden: refusal{status: http.StatusForbidden, code: "woocommerce_rest_cannot_create", message: "Sorry, you are not allowed to create resources."},
		parent:    refusal{status: http.StatusNotFound, code: "woocommerce_rest_term_invalid", message: "Parent resource does not exist."},
		empty:     refusal{status: http.StatusBadRequest, code: "empty_term_name", message: "A name is required for this term."},
		idKey:     "resource_id",
		invalid:   "woocommerce_rest_invalid_param",
	},
}

func storedTermName(raw string) string {
	return termEscaper.Replace(plainTermName(tagPattern.ReplaceAllString(raw, "")))
}

func plainTermName(name string) string {
	return strings.Join(strings.Fields(html.UnescapeString(name)), " ")
}

func sameTermName(left, right string) bool {
	return strings.EqualFold(plainTermName(left), plainTermName(right))
}

func termSlug(name string) string {
	return slugify(plainTermName(name))
}

func (s *Server) termsOf(taxonomy string) []termRef {
	if taxonomy == TypeProductCategory {
		refs := make([]termRef, 0, len(s.termOrder))
		for _, id := range s.termOrder {
			stored := s.terms[id]
			if stored.Type == TypeProductCategory {
				refs = append(refs, termRef{ID: stored.ID, Parent: stored.Parent, Name: stored.Title, Slug: stored.Slug})
			}
		}
		return refs
	}

	refs := make([]termRef, 0, len(s.categoryOrder))
	for _, id := range s.categoryOrder {
		stored := s.categories[id]
		refs = append(refs, termRef{ID: stored.ID, Parent: stored.Parent, Name: stored.Name, Slug: stored.Slug})
	}
	return refs
}

func (s *Server) termPayload(taxonomy string, id int64) map[string]any {
	if taxonomy == TypeProductCategory {
		return s.productCategoryPayload(s.terms[id], true)
	}
	return categoryPayload(s.categories[id])
}

func (s *Server) listTerms(w http.ResponseWriter, r *http.Request, taxonomy string) {
	rules := termRulesOf[taxonomy]
	query := r.URL.Query()
	page, perPage, bad := listWindow(query)
	if bad != "" {
		s.fail(w, http.StatusBadRequest, rules.invalid, "Invalid parameter(s): "+bad)
		return
	}

	s.mu.Lock()
	matched, bad := matchTerms(s.termsOf(taxonomy), query)
	if bad != "" {
		s.mu.Unlock()
		s.fail(w, http.StatusBadRequest, rules.invalid, "Invalid parameter(s): "+bad)
		return
	}

	total := len(matched)
	start := min((page-1)*perPage, total)
	end := min(start+perPage, total)
	payload := make([]map[string]any, 0, end-start)
	for _, term := range matched[start:end] {
		payload = append(payload, narrowFields(s.termPayload(taxonomy, term.ID), query.Get("_fields")))
	}
	s.mu.Unlock()

	w.Header().Set("X-WP-Total", strconv.Itoa(total))
	w.Header().Set("X-WP-TotalPages", strconv.Itoa((total+perPage-1)/perPage))
	s.respond(w, http.StatusOK, payload)
}

func matchTerms(terms []termRef, query url.Values) (matched []termRef, invalid string) {
	var parent *int64
	if raw := query.Get("parent"); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || value < 0 {
			return nil, "parent"
		}
		parent = &value
	}

	include := make([]int64, 0)
	for _, raw := range listValues(query, "include") {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return nil, "include"
		}
		include = append(include, id)
	}
	slugs := listValues(query, "slug")
	search := strings.ToLower(strings.TrimSpace(query.Get("search")))

	matched = make([]termRef, 0, len(terms))
	for _, term := range terms {
		switch {
		case parent != nil && term.Parent != *parent:
		case len(include) > 0 && !slices.Contains(include, term.ID):
		case len(slugs) > 0 && !slices.Contains(slugs, term.Slug):
		case search != "" && !strings.Contains(strings.ToLower(term.Name), search) && !strings.Contains(term.Slug, search):
		default:
			matched = append(matched, term)
		}
	}
	return matched, ""
}

func listValues(query url.Values, key string) []string {
	raws := append(slices.Clone(query[key]), query[key+"[]"]...)
	values := make([]string, 0, len(raws))
	for _, raw := range raws {
		values = append(values, splitList(raw)...)
	}
	return values
}

func (s *Server) createTerm(w http.ResponseWriter, r *http.Request, taxonomy string) {
	body, ok := s.decodeBody(w, r)
	if !ok {
		return
	}
	name, named := body["name"].(string)
	if !named {
		s.respond(w, http.StatusBadRequest, refusal{
			status: http.StatusBadRequest, code: "rest_missing_callback_param", message: "Missing parameter(s): name",
		}.body(nil))
		return
	}

	s.mu.Lock()
	status, payload := s.insertTerm(taxonomy, name, body)
	s.mu.Unlock()

	s.respond(w, status, payload)
}

func (s *Server) insertTerm(taxonomy, raw string, body map[string]any) (status int, payload map[string]any) {
	rules := termRulesOf[taxonomy]
	if s.noTermEdit {
		return rules.forbidden.status, rules.forbidden.body(nil)
	}

	terms := s.termsOf(taxonomy)
	parent := intField(body, "parent")
	if parent != 0 && !slices.ContainsFunc(terms, func(term termRef) bool { return term.ID == parent }) {
		return rules.parent.status, rules.parent.body(nil)
	}

	name := storedTermName(raw)
	if name == "" {
		return rules.empty.status, rules.empty.body(nil)
	}

	slug := ""
	if explicit := stringField(body, "slug"); explicit != "" {
		slug = slugify(explicit)
	}
	if existing, found := duplicateTerm(terms, name, slug, parent); found {
		duplicate := refusal{status: http.StatusBadRequest, code: "term_exists", message: duplicateTermMessage}
		return duplicate.status, duplicate.body(map[string]any{rules.idKey: existing})
	}
	if slug == "" {
		slug = termSlug(name)
	}

	created := termRef{Name: name, Slug: uniqueTermSlug(terms, slug, parent), Parent: parent}
	return http.StatusCreated, s.storeTerm(taxonomy, created, stringField(body, "description"))
}

func (s *Server) storeTerm(taxonomy string, term termRef, description string) map[string]any {
	if taxonomy == TypeProductCategory {
		created := s.add(Item{Type: TypeProductCategory, Title: term.Name, Slug: term.Slug, Parent: term.Parent, Content: description})
		return s.productCategoryPayload(s.terms[created.ID], true)
	}

	s.nextTermID++
	stored := &Category{ID: s.nextTermID, Name: term.Name, Slug: term.Slug, Parent: term.Parent, Description: description}
	s.categories[stored.ID] = stored
	s.categoryOrder = append(s.categoryOrder, stored.ID)
	return categoryPayload(stored)
}

func duplicateTerm(terms []termRef, name, slug string, parent int64) (int64, bool) {
	index := slices.IndexFunc(terms, func(term termRef) bool {
		return term.Parent == parent && sameTermName(term.Name, name)
	})
	if index < 0 {
		return 0, false
	}
	if slug == "" || slug == terms[index].Slug {
		return terms[index].ID, true
	}

	sibling := slices.IndexFunc(terms, func(term termRef) bool { return term.Parent == parent && term.Slug == slug })
	if sibling < 0 {
		return 0, false
	}
	return terms[sibling].ID, true
}

func uniqueTermSlug(terms []termRef, base string, parent int64) string {
	taken := func(slug string) bool {
		return slices.ContainsFunc(terms, func(term termRef) bool { return term.Slug == slug })
	}
	if !taken(base) {
		return base
	}

	slug := base
	ancestor := parent
	for depth := 0; ancestor != 0 && depth < maxPathDepth; depth++ {
		index := slices.IndexFunc(terms, func(term termRef) bool { return term.ID == ancestor })
		if index < 0 {
			break
		}
		slug += "-" + terms[index].Slug
		if !taken(slug) {
			return slug
		}
		ancestor = terms[index].Parent
	}

	candidate := slug
	for suffix := 2; taken(candidate); suffix++ {
		candidate = slug + "-" + strconv.Itoa(suffix)
	}
	return candidate
}

func (s *Server) assignedTerms(taxonomy string, ids []int64) []int64 {
	terms := s.termsOf(taxonomy)
	kept := make([]termRef, 0, len(ids))
	for _, id := range ids {
		index := slices.IndexFunc(terms, func(term termRef) bool { return term.ID == id })
		if index < 0 || slices.ContainsFunc(kept, func(term termRef) bool { return term.ID == id }) {
			continue
		}
		kept = append(kept, terms[index])
	}

	slices.SortStableFunc(kept, func(left, right termRef) int {
		return cmp.Or(
			strings.Compare(strings.ToLower(plainTermName(left.Name)), strings.ToLower(plainTermName(right.Name))),
			cmp.Compare(left.ID, right.ID),
		)
	})

	assigned := make([]int64, 0, len(kept))
	for _, term := range kept {
		assigned = append(assigned, term.ID)
	}
	return assigned
}
