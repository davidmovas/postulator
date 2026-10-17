package field

import "strings"

type Key string

const (
	Title                   Key = "title"
	Excerpt                 Key = "excerpt"
	Body                    Key = "body"
	SEOTitle                Key = "seo_title"
	SEODescription          Key = "seo_description"
	SEOCanonical            Key = "seo_canonical"
	ProductDescription      Key = "product_description"
	ProductShortDescription Key = "product_short_description"
	VariationDescription    Key = "variation_description"
	ImageAlt                Key = "image_alt"
	ImageTitle              Key = "image_title"
	Upsells                 Key = "upsells"
	CrossSells              Key = "cross_sells"
	Meta                    Key = "meta"
	Slug                    Key = "slug"
	Status                  Key = "status"
	Parent                  Key = "parent"
	Price                   Key = "price"
	RegularPrice            Key = "regular_price"
	SalePrice               Key = "sale_price"
	SKU                     Key = "sku"
	StockStatus             Key = "stock_status"
	StockQuantity           Key = "stock_quantity"
	ManageStock             Key = "manage_stock"
	Attributes              Key = "attributes"
	Variation               Key = "variation"
	Terms                   Key = "terms"
)

type Source string

const (
	Core   Source = "core"
	Store  Source = "store"
	Plugin Source = "plugin"
	Render Source = "render"
	None   Source = "none"
)

type Spec struct {
	Key       Key
	Instanced bool
	Read      Source
	Write     Source
	Denied    bool
	Text      bool
}

var catalog = []Spec{
	{Key: Title, Read: Core, Write: Core, Text: true},
	{Key: Excerpt, Read: Core, Write: Core, Text: true},
	{Key: Body, Read: Plugin, Write: Plugin, Text: true},
	{Key: SEOTitle, Read: Plugin, Write: Plugin, Text: true},
	{Key: SEODescription, Read: Plugin, Write: Plugin, Text: true},
	{Key: SEOCanonical, Read: Plugin, Write: Plugin, Text: true},
	{Key: ProductDescription, Read: Store, Write: Plugin, Text: true},
	{Key: ProductShortDescription, Read: Store, Write: Store, Text: true},
	{Key: VariationDescription, Instanced: true, Read: Store, Write: Store, Text: true},
	{Key: ImageAlt, Instanced: true, Read: Core, Write: Core, Text: true},
	{Key: ImageTitle, Instanced: true, Read: Core, Write: Core, Text: true},
	{Key: Upsells, Read: Store, Write: Store},
	{Key: CrossSells, Read: Store, Write: Store},
	{Key: Meta, Instanced: true, Read: Plugin, Write: Plugin, Text: true},
	{Key: Slug, Read: Core, Write: None, Denied: true},
	{Key: Status, Read: Core, Write: None, Denied: true},
	{Key: Parent, Read: Core, Write: None, Denied: true},
	{Key: Price, Read: Store, Write: None, Denied: true},
	{Key: RegularPrice, Read: Store, Write: None, Denied: true},
	{Key: SalePrice, Read: Store, Write: None, Denied: true},
	{Key: SKU, Read: Store, Write: None, Denied: true},
	{Key: StockStatus, Read: Store, Write: None, Denied: true},
	{Key: StockQuantity, Read: Store, Write: None, Denied: true},
	{Key: ManageStock, Read: Store, Write: None, Denied: true},
	{Key: Attributes, Read: Store, Write: None, Denied: true},
	{Key: Variation, Instanced: true, Read: Store, Write: None, Denied: true},
	{Key: Terms, Read: Core, Write: None, Denied: true},
}

func Catalog() []Spec {
	out := make([]Spec, len(catalog))
	copy(out, catalog)
	return out
}

func Lookup(key Key) (Spec, bool) {
	kind, instance := key.Kind(), key.Instance()
	for _, spec := range catalog {
		if spec.Key != kind {
			continue
		}
		if instance != "" && !spec.Instanced {
			return Spec{}, false
		}
		return spec, true
	}
	return Spec{}, false
}

func Denied() []Key {
	out := make([]Key, 0, len(catalog))
	for _, spec := range catalog {
		if spec.Denied {
			out = append(out, spec.Key)
		}
	}
	return out
}

func Instance(kind Key, instance string) Key {
	return Key(string(kind) + ":" + instance)
}

func (k Key) Kind() Key {
	kind, _, _ := strings.Cut(string(k), ":")
	return Key(kind)
}

func (k Key) Instance() string {
	_, instance, _ := strings.Cut(string(k), ":")
	return instance
}
