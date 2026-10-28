package observe

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"slices"
	"time"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const detailClosed = "closedToUser"

func ClosedToUser(err error) error {
	return errors.New(errors.Unauthorized, "the site's user may not edit this page").
		WithDetail(detailClosed, true).WithInternal(err)
}

func closedToUser(err error) bool {
	var kernel *errors.Error
	if !stderrors.As(err, &kernel) || kernel == nil {
		return false
	}
	closed, ok := kernel.Details[detailClosed].(bool)
	return ok && closed
}

type PostType struct {
	Slug         string
	RestBase     string
	Viewable     bool
	Hierarchical bool
}

type ItemQuery struct {
	Statuses []string
	Page     int
	PerPage  int
}

type Item struct {
	Modified time.Time
	Type     string
	Link     string
	Slug     string
	Status   string
	Title    string
	Content  string
	Excerpt  string
	Terms    []pagemap.Term
	ID       int64
	Parent   int64
}

type ItemPage struct {
	Items   []Item
	HasMore bool
}

type SiteSettings struct {
	ShowOnFront string
	FrontPageID int64
	PostsPageID int64
}

type CoreReader interface {
	PostTypes(ctx context.Context) ([]PostType, error)
	ListItems(ctx context.Context, postType string, query ItemQuery) (ItemPage, error)
	SiteSettings(ctx context.Context) (SiteSettings, error)
}

type ContentQuery struct {
	Since  *time.Time
	Types  []string
	Cursor string
	Limit  int
}

type ContentLink struct {
	Href   string
	Anchor string
}

type SEOMeta struct {
	Title       string
	Description string
	Canonical   string
}

func (m SEOMeta) Empty() bool {
	return m.Title == "" && m.Description == "" && m.Canonical == ""
}

type ContentItem struct {
	Modified    time.Time
	Type        string
	Path        string
	Slug        string
	Status      string
	Title       string
	H1          string
	ContentHash string
	Excerpt     string
	Meta        SEOMeta
	Links       []ContentLink
	Terms       []pagemap.Term
	ID          int64
	Parent      int64
}

type ContentPage struct {
	Items      []ContentItem
	NextCursor string
}

type Raw struct {
	Content string
	Hash    string
}

type Rendered struct {
	Content string
	Hash    string
}

type PluginReader interface {
	ListContent(ctx context.Context, query ContentQuery) (ContentPage, error)
	GetRaw(ctx context.Context, postType string, id int64) (Raw, error)
	GetRendered(ctx context.Context, postType string, id int64) (Rendered, error)
	GetSEOMeta(ctx context.Context, postType string, id int64) (SEOMeta, error)
	GetTerms(ctx context.Context, postType string, id int64) ([]pagemap.Term, error)
	GetMeta(ctx context.Context, postType string, id int64, keys []string) (map[string]string, error)
	Settings(ctx context.Context) (site.TechnicalPages, error)
	PreviewLink(ctx context.Context, postType string, id int64) (string, error)
}

type ProductQuery struct {
	ModifiedAfter *time.Time
	Page          int
	PerPage       int
}

type Product struct {
	Modified         time.Time
	Permalink        string
	Slug             string
	Status           string
	Name             string
	Type             string
	Description      string
	ShortDescription string
	Price            string
	RegularPrice     string
	SalePrice        string
	SKU              string
	StockStatus      string
	ManageStock      string
	Attributes       json.RawMessage
	Categories       []pagemap.Term
	VariationIDs     []int64
	UpsellIDs        []int64
	CrossSellIDs     []int64
	Images           []ProductImage
	StockQuantity    *int64
	ID               int64
}

type ProductImage struct {
	Alt   string
	Title string
	ID    int64
}

type ProductPage struct {
	Items   []Product
	HasMore bool
}

type Variation struct {
	Modified      time.Time
	Description   string
	Status        string
	SKU           string
	Price         string
	RegularPrice  string
	SalePrice     string
	StockStatus   string
	ManageStock   string
	Attributes    json.RawMessage
	StockQuantity *int64
	ID            int64
	ParentID      int64
}

type VariationPage struct {
	Items   []Variation
	HasMore bool
}

type StoreReader interface {
	ListProducts(ctx context.Context, query ProductQuery) (ProductPage, error)
	ListVariations(ctx context.Context, productID int64, page int) (VariationPage, error)
	StorePages(ctx context.Context) (site.TechnicalPages, error)
}

type Visit struct {
	Body   string
	Status int
	URL    string
}

type Resolution struct {
	URL    string
	Status int
}

type RenderReader interface {
	Visit(ctx context.Context, url string) (Visit, error)
	Resolve(ctx context.Context, url string) (Resolution, error)
}

type Manifest struct {
	Version      string
	Required     string
	Capabilities []string
	PostTypes    []string
	Taxonomies   []string
	Outdated     bool
}

func (m Manifest) Has(capability string) bool {
	return slices.Contains(m.Capabilities, capability)
}

func (m Manifest) listsTerms() bool {
	return len(m.PostTypes) > 0
}

type Readers struct {
	Core     CoreReader
	Store    StoreReader
	Plugin   PluginReader
	Render   RenderReader
	Manifest Manifest
}

const (
	capabilityBulk     = "bulk"
	capabilityRaw      = "raw"
	capabilityRendered = "rendered"
	capabilityTerms    = "terms"
	capabilitySettings = "settings"
	capabilityPreview  = "preview"
	capabilityMeta     = "meta"
)

func (r Readers) plugin(capability string) bool {
	return r.Plugin != nil && r.Manifest.Has(capability)
}
