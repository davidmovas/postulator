package wp

import (
	"context"
	"html"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type Taxonomy string

const (
	TaxonomyCategory        Taxonomy = "category"
	TaxonomyProductCategory Taxonomy = "product_cat"

	termFields = "id,name,slug,parent,count"
)

func (t Taxonomy) route() (namespace, path string, err error) {
	switch t {
	case TaxonomyCategory:
		return coreNamespace, "/categories", nil
	case TaxonomyProductCategory:
		return wooNamespace, "/products/categories", nil
	default:
		return "", "", errors.New(errors.Invalid, "unknown WordPress taxonomy").WithDetail("taxonomy", string(t))
	}
}

type Term struct {
	Name   string
	Slug   string
	ID     int64
	Parent int64
	Count  int
}

type TermPage = Page[Term]

type TermQuery struct {
	Parent  *int64
	Include []int64
	Page    int
	PerPage int
}

func (q TermQuery) values() url.Values {
	query := url.Values{}
	query.Set("page", strconv.Itoa(pageNumber(q.Page)))
	query.Set("per_page", strconv.Itoa(perPageSize(q.PerPage)))
	query.Set("orderby", "id")
	query.Set("order", "asc")
	query.Set("_fields", termFields)
	if q.Parent != nil {
		query.Set("parent", strconv.FormatInt(*q.Parent, 10))
	}
	if len(q.Include) > 0 {
		ids := make([]string, 0, len(q.Include))
		for _, id := range q.Include {
			ids = append(ids, strconv.FormatInt(id, 10))
		}
		query.Set("include", strings.Join(ids, ","))
	}
	return query
}

type termPayload struct {
	Name   string `json:"name"`
	Slug   string `json:"slug"`
	ID     int64  `json:"id"`
	Parent int64  `json:"parent"`
	Count  int    `json:"count"`
}

func (p termPayload) term() Term {
	return Term{ID: p.ID, Parent: p.Parent, Name: html.UnescapeString(p.Name), Slug: p.Slug, Count: p.Count}
}

func (c *Client) ListTerms(ctx context.Context, taxonomy Taxonomy, query TermQuery) (TermPage, error) {
	namespace, path, err := taxonomy.route()
	if err != nil {
		return TermPage{}, err
	}

	resp, body, err := c.do(ctx, request{
		method:    http.MethodGet,
		namespace: namespace,
		path:      path,
		query:     query.values(),
	})
	if err != nil {
		return TermPage{}, err
	}

	var payload []termPayload
	if err := decodeJSON(body, &payload); err != nil {
		return TermPage{}, err
	}
	terms := make([]Term, 0, len(payload))
	for _, entry := range payload {
		terms = append(terms, entry.term())
	}
	return newPage(resp, pageNumber(query.Page), terms), nil
}

func (c *Client) GetTerm(ctx context.Context, taxonomy Taxonomy, id int64) (Term, error) {
	namespace, path, err := taxonomy.route()
	if err != nil {
		return Term{}, err
	}

	_, body, err := c.do(ctx, request{
		method:    http.MethodGet,
		namespace: namespace,
		path:      resourcePath(path, id),
		query:     url.Values{"_fields": {termFields}},
	})
	if err != nil {
		return Term{}, err
	}
	return decodeTerm(body)
}

func (c *Client) CreateTerm(ctx context.Context, taxonomy Taxonomy, name string, parent int64) (Term, error) {
	namespace, path, err := taxonomy.route()
	if err != nil {
		return Term{}, err
	}
	if err := requireTermName(name); err != nil {
		return Term{}, err
	}

	attributes := map[string]any{"name": name}
	if parent != 0 {
		attributes["parent"] = parent
	}
	body, err := encodeJSON(attributes)
	if err != nil {
		return Term{}, err
	}

	_, raw, err := c.do(ctx, request{
		method:      http.MethodPost,
		namespace:   namespace,
		path:        path,
		body:        body,
		contentType: contentTypeJSON,
	})
	if err != nil {
		return Term{}, err
	}
	return decodeTerm(raw)
}

func (c *Client) FindTerm(ctx context.Context, taxonomy Taxonomy, name string, parent int64) (Term, bool, error) {
	if _, _, err := taxonomy.route(); err != nil {
		return Term{}, false, err
	}
	if err := requireTermName(name); err != nil {
		return Term{}, false, err
	}

	query := TermQuery{Parent: &parent, PerPage: maxPerPage, Page: 1}
	for {
		page, err := c.ListTerms(ctx, taxonomy, query)
		if err != nil {
			return Term{}, false, err
		}
		for _, term := range page.Items {
			if SameTermName(term.Name, name) {
				return term, true, nil
			}
		}
		if !page.HasMore || len(page.Items) == 0 {
			return Term{}, false, nil
		}
		query.Page++
	}
}

func (c *Client) EnsureTerm(ctx context.Context, taxonomy Taxonomy, name string, parent int64) (Term, bool, error) {
	found, ok, err := c.FindTerm(ctx, taxonomy, name, parent)
	if err != nil || ok {
		return found, false, err
	}

	created, err := c.CreateTerm(ctx, taxonomy, name, parent)
	if err == nil {
		return created, true, nil
	}
	existing, exists := TermExists(err)
	if !exists {
		return Term{}, false, err
	}

	adopted, err := c.GetTerm(ctx, taxonomy, existing)
	if err != nil {
		return Term{}, false, err
	}
	return adopted, false, nil
}

func SameTermName(left, right string) bool {
	plainLeft, plainRight := plainTermName(left), plainTermName(right)
	return plainLeft != "" && strings.EqualFold(plainLeft, plainRight)
}

func TermExists(err error) (int64, bool) {
	if !errors.IsCode(err, errors.Invalid) || detailString(err, "code") != "term_exists" {
		return 0, false
	}
	value, ok := detailValue(err, "termId")
	if !ok {
		return 0, false
	}
	id, ok := value.(int64)
	return id, ok && id > 0
}

func plainTermName(name string) string {
	return strings.Join(strings.Fields(html.UnescapeString(name)), " ")
}

func requireTermName(name string) error {
	if plainTermName(name) == "" {
		return errors.New(errors.Invalid, "a WordPress term needs a name")
	}
	return nil
}

func decodeTerm(body []byte) (Term, error) {
	var payload termPayload
	if err := decodeJSON(body, &payload); err != nil {
		return Term{}, err
	}
	return payload.term(), nil
}
