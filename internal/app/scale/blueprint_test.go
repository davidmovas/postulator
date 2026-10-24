//go:build !race

package scale_test

import (
	"fmt"
	"strconv"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

const (
	blockPages           = 40
	variationsPerProduct = 2
	productsPerCategory  = 6
	termTaxonomy         = "product_cat"
	firstPostID          = 1000
	firstTermID          = 100000
)

type kind int

const (
	kindFront kind = iota
	kindShop
	kindCart
	kindHub
	kindTopic
	kindArticle
	kindPost
	kindTerm
	kindProduct
	kindArchived
)

type spec struct {
	kind     kind
	block    int
	wpType   pagemap.WPType
	path     string
	title    string
	status   string
	wpID     int64
	termID   int64
	parentWP int64
	parent   string
	crumbs   []crumb
	listing  []string
	links    []string
	terms    []pagemap.Term
	entity   string
	scope    string
	related  string
	written  bool
	drifted  bool
	edited   bool
	touchedA bool
	touchedB bool
}

func (s *spec) key() string {
	return fmt.Sprintf("%s:%09d", s.wpType, s.wpID+s.termID)
}

func (s *spec) titleAt(revision int) string {
	if revision == 0 {
		return s.title
	}
	return s.title + " revised " + strconv.Itoa(revision)
}

func (s *spec) excerpt() string {
	return "A short word about " + s.title + "."
}

func (s *spec) description() string {
	return "Everything about " + s.title + " in one place."
}

func (s *spec) sku() string {
	return fmt.Sprintf("SKU-%06d", s.wpID)
}

func (s *spec) section() string {
	return fmt.Sprintf("/c%03d/", s.block)
}

func (s *spec) body(revision int) string {
	if s.wpType.Term() {
		return ""
	}
	text := "<!-- wp:paragraph -->\n<p>" + s.titleAt(revision) + " opens with the client's own words"
	for i, path := range s.links {
		text += ` and <a href="` + path + `">link ` + strconv.Itoa(i) + "</a>"
	}
	text += ".</p>\n<!-- /wp:paragraph -->\n\n<!-- wp:paragraph -->\n<p>A second paragraph about " + s.title +
		" closes the page.</p>\n<!-- /wp:paragraph -->"
	return text
}

type blueprint struct {
	specs  []*spec
	byPath map[string]*spec
	nextID int64
	nextT  int64
}

func (b *blueprint) add(s *spec) *spec {
	if s.status == "" {
		s.status = statusPublish
	}
	switch {
	case s.kind == kindArchived:
	case s.wpType.Term():
		s.termID = b.nextT
		b.nextT++
	default:
		s.wpID = b.nextID
		b.nextID++
	}
	b.specs = append(b.specs, s)
	b.byPath[s.path] = s
	return s
}

func plan(blocks int) *blueprint {
	b := &blueprint{byPath: map[string]*spec{}, nextID: firstPostID, nextT: firstTermID}
	home := crumb{name: "Home", path: "/"}
	b.add(&spec{kind: kindFront, wpType: pagemap.WPPage, path: "/", title: "Home"})
	b.add(&spec{kind: kindShop, wpType: pagemap.WPPage, path: "/shop/", title: "Shop", crumbs: []crumb{home}})
	b.add(&spec{kind: kindCart, wpType: pagemap.WPPage, path: "/cart/", title: "Cart", crumbs: []crumb{home}})
	for at := range blocks {
		b.block(at, home)
	}
	return b
}

func (b *blueprint) block(at int, home crumb) {
	name := fmt.Sprintf("Category %03d", at)
	hubPath := fmt.Sprintf("/c%03d/", at)
	termPath := fmt.Sprintf("/product-category/c%03d/", at)
	hub := b.add(&spec{
		kind: kindHub, block: at, wpType: pagemap.WPPage, path: hubPath, title: name, parent: "/", crumbs: []crumb{home},
		entity: name,
	})
	term := b.add(&spec{kind: kindTerm, block: at, wpType: pagemap.WPProductCategory, path: termPath, title: name, crumbs: []crumb{home}})
	for i := range 2 {
		subName := fmt.Sprintf("%s %d", name, i)
		subPath := fmt.Sprintf("%ss%d/", hubPath, i)
		subTermPath := fmt.Sprintf("%ss%d/", termPath, i)
		sub := b.add(&spec{
			kind: kindHub, block: at, wpType: pagemap.WPPage, path: subPath, title: subName, parentWP: hub.wpID, parent: hubPath,
			crumbs: []crumb{home, {name: name, path: hubPath}}, entity: subName, scope: name,
		})
		hub.listing = append(hub.listing, subPath)
		subTerm := b.add(&spec{
			kind: kindTerm, block: at, wpType: pagemap.WPProductCategory, path: subTermPath, title: subName, parent: termPath,
			crumbs: []crumb{home, {name: name, path: termPath}},
		})
		term.listing = append(term.listing, subTermPath)
		category := pagemap.Term{Taxonomy: termTaxonomy, TermID: subTerm.termID, Name: subName, Slug: pagemap.Slug(subTermPath), ParentTermID: term.termID}
		for j := range 2 {
			topicName := fmt.Sprintf("Topic %03d %d %d", at, i, j)
			topicPath := fmt.Sprintf("%st%d/", subPath, j)
			topic := b.add(&spec{
				kind: kindTopic, block: at, wpType: pagemap.WPPage, path: topicPath, title: topicName, parentWP: sub.wpID, parent: subPath,
				crumbs: []crumb{home, {name: name, path: hubPath}, {name: subName, path: subPath}}, entity: topicName, scope: subName,
			})
			sub.listing = append(sub.listing, topicPath)
			for k := range 3 {
				articleName := fmt.Sprintf("Article %03d %d %d %d", at, i, j, k)
				articlePath := fmt.Sprintf("%sp%d/", topicPath, k)
				article := &spec{
					kind: kindArticle, block: at, wpType: pagemap.WPPage, path: articlePath, title: articleName, parentWP: topic.wpID,
					parent: topicPath, crumbs: []crumb{home, {name: name, path: hubPath}, {name: subName, path: subPath}, {name: topicName, path: topicPath}},
					links:   []string{fmt.Sprintf("%sp%d/", topicPath, (k+1)%3), fmt.Sprintf("/product/item-%03d-%d-%d/", at, i, k)},
					written: i == 0 && j == 1, drifted: i == 1 && j == 1 && k == 0, edited: i == 0 && j == 0 && k == 0,
					touchedA: i == 1 && j == 0 && k == 0, touchedB: i == 1 && k == 1,
				}
				if k < 2 {
					article.entity, article.scope = articleName, topicName
				}
				if k == 1 {
					article.related = fmt.Sprintf("Article %03d %d %d 0", at, i, j)
				}
				b.add(article)
				topic.listing = append(topic.listing, articlePath)
			}
		}
		for k := range productsPerCategory {
			itemName := fmt.Sprintf("Item %03d %d %d", at, i, k)
			itemPath := fmt.Sprintf("/product/item-%03d-%d-%d/", at, i, k)
			b.add(&spec{
				kind: kindProduct, block: at, wpType: pagemap.WPProduct, path: itemPath, title: itemName, parent: subTermPath,
				crumbs: []crumb{home, {name: name, path: termPath}, {name: subName, path: subTermPath}},
				links:  []string{fmt.Sprintf("/product/item-%03d-%d-%d/", at, i, (k+1)%productsPerCategory)},
				terms:  []pagemap.Term{category}, entity: itemName, scope: subName,
				touchedA: i == 1 && k == 0, touchedB: i == 1 && (k == 2 || k == 3),
			})
			subTerm.listing = append(subTerm.listing, itemPath)
		}
	}
	for k := range 3 {
		postName := fmt.Sprintf("Post %03d %d", at, k)
		post := &spec{
			kind: kindPost, block: at, wpType: pagemap.WPPost, path: fmt.Sprintf("/blog/post-%03d-%d/", at, k), title: postName,
			parent: hubPath, crumbs: []crumb{home, {name: "Blog", path: "/blog/"}, {name: name, path: hubPath}},
			links: []string{hubPath},
		}
		if k < 2 {
			post.entity, post.scope = postName, name
		}
		b.add(post)
	}
	b.add(&spec{
		kind: kindArchived, block: at, wpType: pagemap.WPPage, path: fmt.Sprintf("%sold/", hubPath), title: fmt.Sprintf("Old %03d", at),
		status: "archived", parent: hubPath,
	})
}

func (b *blueprint) touched(second bool) []string {
	out := make([]string, 0)
	for _, s := range b.specs {
		if (second && s.touchedB) || (!second && s.touchedA) {
			out = append(out, s.path)
		}
	}
	return out
}

const sentinel = "/cart/"

func plannedRows(blocks int) string {
	sheet := "path,title,entity,parent\n"
	for at := range blocks {
		for k := range 2 {
			sheet += fmt.Sprintf("/c%03d/s0/t0/new-%d/,New %03d %d,New %03d %d,Topic %03d 0 0\n", at, k, at, k, at, k, at)
		}
	}
	return sheet
}
