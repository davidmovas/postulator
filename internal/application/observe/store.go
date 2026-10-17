package observe

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

func (s *Service) store(ctx context.Context, state *State, readers Readers, batch int) (bool, error) {
	if readers.Store == nil {
		return true, nil
	}
	owner, err := s.deps.Sites.Get(ctx, state.SiteID)
	if err != nil {
		return false, err
	}
	install := pagemap.NewSite(owner.BaseURL)
	listed, err := readers.Store.ListProducts(ctx, ProductQuery{ModifiedAfter: state.Since, Page: state.StorePage + 1, PerPage: batch})
	if err != nil {
		return false, err
	}
	state.StorePage++

	items := make([]item, 0, len(listed.Items))
	products := make(map[int64]Product, len(listed.Items))
	for i := range listed.Items {
		if converted, ok := fromProduct(listed.Items[i], install); ok {
			items = append(items, converted)
			products[listed.Items[i].ID] = listed.Items[i]
		}
	}
	if err := s.absorb(ctx, state, readers, owner, install, items); err != nil {
		return false, err
	}

	tracked, err := s.findings(ctx, state.SiteID, modelCodes...)
	if err != nil {
		return false, err
	}
	pages, err := s.deps.Pages.ListBySite(ctx, state.SiteID)
	if err != nil {
		return false, err
	}
	rows := rowsOf(pages)
	a := &absorber{service: s, state: state, readers: readers, owner: owner, install: install, rows: rows, tracked: tracked, now: s.now()}
	err = s.inUnit(ctx, func(c context.Context) error {
		for id := range products {
			page, known := rows.byWPID[keyOf(pagemap.WPProduct, id)]
			if !known {
				continue
			}
			if facetsErr := a.productFields(c, page, products[id]); facetsErr != nil {
				return facetsErr
			}
			state.Counts.Products++
		}
		return nil
	})
	if err != nil {
		return false, err
	}
	return !listed.HasMore, nil
}

func (a *absorber) productFields(ctx context.Context, page pagemap.Page, product Product) error {
	values := []field.Value{
		field.NewValue(page.ID, field.ProductDescription, product.Description, a.now),
		field.NewValue(page.ID, field.ProductShortDescription, product.ShortDescription, a.now),
		field.NewValue(page.ID, field.Price, product.Price, a.now),
		field.NewValue(page.ID, field.RegularPrice, product.RegularPrice, a.now),
		field.NewValue(page.ID, field.SalePrice, product.SalePrice, a.now),
		field.NewValue(page.ID, field.SKU, product.SKU, a.now),
		field.NewValue(page.ID, field.StockStatus, product.StockStatus, a.now),
		field.NewValue(page.ID, field.StockQuantity, quantity(product.StockQuantity), a.now),
		field.NewValue(page.ID, field.ManageStock, product.ManageStock, a.now),
		field.NewValue(page.ID, field.Attributes, string(product.Attributes), a.now),
	}
	for number := 1; ; number++ {
		variations, err := a.readers.Store.ListVariations(ctx, product.ID, number)
		if err != nil {
			return err
		}
		for i := range variations.Items {
			variation := variations.Items[i]
			instance := strconv.FormatInt(variation.ID, 10)
			facts, marshalErr := json.Marshal(variationFacts{
				SKU: variation.SKU, Price: variation.Price, RegularPrice: variation.RegularPrice, SalePrice: variation.SalePrice,
				StockStatus: variation.StockStatus, StockQuantity: quantity(variation.StockQuantity), ManageStock: variation.ManageStock,
				Status: variation.Status, Attributes: variation.Attributes,
			})
			if marshalErr != nil {
				return marshalErr
			}
			values = append(values,
				field.NewValue(page.ID, field.Instance(field.VariationDescription, instance), variation.Description, a.now),
				field.NewValue(page.ID, field.Instance(field.Variation, instance), string(facts), a.now),
			)
		}
		if !variations.HasMore {
			break
		}
	}
	if err := a.observeFields(ctx, page, values); err != nil {
		return err
	}
	if a.readers.plugin(capabilityTerms) || len(product.Categories) == 0 {
		return nil
	}
	return a.service.deps.Identity.ReplaceTerms(ctx, page.ID, product.Categories)
}

type variationFacts struct {
	SKU           string          `json:"sku"`
	Price         string          `json:"price"`
	RegularPrice  string          `json:"regularPrice"`
	SalePrice     string          `json:"salePrice"`
	StockStatus   string          `json:"stockStatus"`
	StockQuantity string          `json:"stockQuantity"`
	ManageStock   string          `json:"manageStock"`
	Status        string          `json:"status"`
	Attributes    json.RawMessage `json:"attributes"`
}

func quantity(value *int64) string {
	if value == nil {
		return ""
	}
	return strconv.FormatInt(*value, 10)
}
