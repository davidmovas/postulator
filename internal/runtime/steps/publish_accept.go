package steps

import (
	"slices"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/field"
)

func (s *publishState) narrow(accepted []field.Key, body field.Key) {
	if len(accepted) == 0 {
		return
	}
	offered := make([]field.Key, 0, len(s.Result.Fields)+1)
	for i := range s.Result.Fields {
		offered = append(offered, s.Result.Fields[i].Key)
	}
	if body != "" {
		offered = append(offered, body)
	}
	declined := make([]field.Key, 0, len(offered))
	for _, key := range offered {
		if !slices.Contains(accepted, key) && !slices.Contains(declined, key) {
			declined = append(declined, key)
		}
	}
	if len(declined) == 0 {
		return
	}
	slices.Sort(declined)

	plan := s.Plan
	for _, key := range declined {
		plan.decline(key)
	}
	plan.Kept = slices.Compact(slices.Sorted(slices.Values(append(plan.Kept, declined...))))
	plan.Writes = slices.DeleteFunc(plan.Writes, func(unit string) bool { return !plan.carries(unit) })
	s.Result.Fields = slices.DeleteFunc(s.Result.Fields, func(change field.Change) bool {
		return slices.Contains(declined, change.Key)
	})
	if plan.Request.KeepBody {
		s.Result.Blocks = nil
	}
	if plan.Store != nil && s.Result.PreviousProduct != nil {
		snapshot := plan.Store.Snapshot
		s.Result.PreviousProduct = &snapshot
	}
	s.Result.Declined = declined
}

func (p *publishPlan) decline(key field.Key) {
	switch key {
	case field.Title:
		p.Request.KeepTitle = true
	case field.Excerpt:
		p.Request.Excerpt = nil
	case field.SEOTitle:
		p.SEO.Meta.Title, p.SEO.Meta.OGTitle = "", ""
	case field.SEODescription:
		p.SEO.Meta.Description, p.SEO.Meta.OGDescription = "", ""
	case field.SEOCanonical:
		p.SEO.Meta.Canonical, p.SEO.Canonical = "", false
	case field.Body, field.ProductDescription:
		p.Request.KeepBody = true
	case field.ProductShortDescription:
		if p.Store != nil {
			p.Store.ShortDescription, p.Store.Snapshot.ShortWritten = nil, false
		}
	case field.Upsells, field.CrossSells:
		if p.Store != nil {
			p.Store.Lists = slices.DeleteFunc(p.Store.Lists, func(list variationPlan) bool { return list.Key == key })
			p.Store.Snapshot.Lists = slices.DeleteFunc(p.Store.Snapshot.Lists, func(list FieldWrite) bool { return list.Key == key })
		}
	default:
		p.Variations = slices.DeleteFunc(p.Variations, func(variation variationPlan) bool { return variation.Key == key })
		p.Custom = slices.DeleteFunc(p.Custom, func(custom customPlan) bool { return custom.Key == key })
		for i := range p.Media {
			p.Media[i].Writes = slices.DeleteFunc(p.Media[i].Writes, func(write variationPlan) bool { return write.Key == key })
		}
		p.Media = slices.DeleteFunc(p.Media, func(media mediaPlan) bool { return len(media.Writes) == 0 })
	}
}

func (p *publishPlan) carries(unit string) bool {
	switch {
	case unit == unitCheck:
		return true
	case unit == unitItem:
		return p.Request.carries()
	case unit == unitStore:
		return p.Store != nil && len(p.Store.writes()) > 0
	case unit == unitSEO:
		return len(p.SEO.writes()) > 0
	case unit == unitRaw:
		return !p.Request.KeepBody
	case unit == unitCustomPlugin, unit == unitCustomStore:
		return slices.Contains(p.customUnits(), unit)
	case strings.HasPrefix(unit, unitMedia):
		return slices.ContainsFunc(p.Media, func(media mediaPlan) bool { return media.unit() == unit })
	default:
		return slices.ContainsFunc(p.Variations, func(variation variationPlan) bool { return string(variation.Key) == unit })
	}
}
