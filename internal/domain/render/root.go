package render

import (
	"regexp"
	"slices"
	"sort"
	"strings"
	"unicode"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/content"
)

const rawShare = 0.6

var unseenElements = []string{"head", "script", "style", "noscript", "template"}

var loneShortcode = regexp.MustCompile(`(?s)^\[([a-zA-Z_][\w-]*)(?:\s[^\]]*)?/?\](?:.*\[/([a-zA-Z_][\w-]*)\])?$`)

type segment struct {
	start int
	end   int
	node  *html.Node
}

type token struct {
	start int
	end   int
}

type stream struct {
	text     strings.Builder
	segments []segment
	tokens   []token
	words    []string
	spans    map[*html.Node][2]int
}

type run struct {
	block int
	start int
	end   int
}

func newStream(nodes ...*html.Node) *stream {
	s := &stream{spans: make(map[*html.Node][2]int)}
	for _, node := range nodes {
		s.walk(node)
	}
	s.tokenize()
	return s
}

func parseFragment(fragment string) []*html.Node {
	if strings.TrimSpace(fragment) == "" {
		return nil
	}
	body := &html.Node{Type: html.ElementNode, Data: "body", DataAtom: atom.Body}
	nodes, err := html.ParseFragment(strings.NewReader(fragment), body)
	if err != nil {
		return nil
	}
	return nodes
}

func wordsOf(nodes ...*html.Node) []string {
	return newStream(nodes...).words
}

func (s *stream) walk(node *html.Node) {
	switch node.Type {
	case html.TextNode:
		start := s.text.Len()
		s.text.WriteString(node.Data)
		s.segments = append(s.segments, segment{start: start, end: s.text.Len(), node: node})
		return
	case html.ElementNode:
		if slices.Contains(unseenElements, node.Data) {
			s.text.WriteByte(' ')
			return
		}
	case html.DocumentNode:
	default:
		return
	}

	separated := node.Type == html.ElementNode && content.IsBlockElement(node.Data)
	start := s.text.Len()
	if separated {
		s.text.WriteByte(' ')
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		s.walk(child)
	}
	if separated {
		s.text.WriteByte(' ')
	}
	s.spans[node] = [2]int{start, s.text.Len()}
}

func (s *stream) tokenize() {
	text := s.text.String()
	start := -1
	for i, r := range text {
		letter := unicode.IsLetter(r) || unicode.IsDigit(r)
		switch {
		case letter && start < 0:
			start = i
		case !letter && start >= 0:
			s.emit(text, start, i)
			start = -1
		}
	}
	if start >= 0 {
		s.emit(text, start, len(text))
	}
}

func (s *stream) emit(text string, start, end int) {
	for _, word := range content.WordsOf(text[start:end]) {
		s.tokens = append(s.tokens, token{start: start, end: end})
		s.words = append(s.words, word)
	}
}

func (s *stream) firstAt(offset int) int {
	return sort.Search(len(s.tokens), func(i int) bool { return s.tokens[i].start >= offset })
}

func (s *stream) wordRange(node *html.Node) (from, to int) {
	span, ok := s.spans[node]
	if !ok {
		return 0, 0
	}
	return s.firstAt(span[0]), s.firstAt(span[1])
}

func (s *stream) count(node *html.Node) int {
	from, to := s.wordRange(node)
	return to - from
}

func (s *stream) nodeAt(offset int) *html.Node {
	i := sort.Search(len(s.segments), func(i int) bool { return s.segments[i].end > offset })
	if i == len(s.segments) {
		return nil
	}
	return s.segments[i].node
}

func (s *stream) firstNode(word int) *html.Node {
	return s.nodeAt(s.tokens[word].start)
}

func (s *stream) lastNode(word int) *html.Node {
	return s.nodeAt(s.tokens[word].end - 1)
}

func (s *stream) occurrences(needle []string, from, to int) []int {
	out := make([]int, 0)
	if len(needle) == 0 {
		return out
	}
	for i := from; i+len(needle) <= to; i++ {
		if slices.Equal(s.words[i:i+len(needle)], needle) {
			out = append(out, i)
		}
	}
	return out
}

func lowestCommonAncestor(a, b *html.Node) *html.Node {
	seen := make(map[*html.Node]bool)
	for n := a; n != nil; n = n.Parent {
		seen[n] = true
	}
	for n := b; n != nil; n = n.Parent {
		if seen[n] {
			return n
		}
	}
	return nil
}

func elementOf(node *html.Node) *html.Node {
	for node != nil && node.Type != html.ElementNode {
		node = node.Parent
	}
	return node
}

func (p *page) locate(needle, wanted []string, from, to int) (int, bool) {
	best, bestScore := -1, -1
	for _, at := range p.text.occurrences(needle, from, to) {
		if score := p.linksWithin(at, at+len(needle), wanted); score > bestScore {
			best, bestScore = at, score
		}
	}
	return best, best >= 0
}

func (p *page) linksWithin(from, to int, wanted []string) int {
	if len(wanted) == 0 {
		return 0
	}
	remaining := make(map[string]int, len(wanted))
	for _, key := range wanted {
		remaining[key]++
	}
	score := 0
	for _, a := range p.anchors {
		first, last := p.text.wordRange(a.node)
		if first >= last || first < from || last > to || remaining[a.key] == 0 {
			continue
		}
		remaining[a.key]--
		score++
	}
	return score
}

func (p *page) matchBlocks(blocks []rawBlock, from, to int) []run {
	runs := make([]run, 0, len(blocks))
	for i := range blocks {
		b := &blocks[i]
		at, ok := p.locate(b.words, b.links, from, to)
		if !ok {
			continue
		}
		runs = append(runs, run{block: b.index, start: at, end: at + len(b.words)})
		from = at + len(b.words)
	}
	return runs
}

func (p *page) spanning(runs []run) *html.Node {
	var root *html.Node
	for _, r := range runs {
		for _, node := range []*html.Node{p.text.firstNode(r.start), p.text.lastNode(r.end - 1)} {
			switch {
			case node == nil:
			case root == nil:
				root = node
			default:
				root = lowestCommonAncestor(root, node)
			}
		}
	}
	return elementOf(root)
}

func (p *page) findRoot() {
	if nodes := parseFragment(p.in.Filtered); len(nodes) > 0 {
		if filtered := wordsOf(nodes...); len(filtered) > 0 {
			if at, ok := p.locate(filtered, p.linkKeys(nodes...), 0, len(p.text.words)); ok {
				p.setRoot(p.spanning([]run{{start: at, end: at + len(filtered)}}))
				return
			}
		}
		if root := p.byBlocks(p.fragmentBlocks(nodes)); root != nil {
			p.setRoot(root)
			return
		}
	}
	if root := p.byBlocks(p.blocks); root != nil {
		p.setRoot(root)
		return
	}
	p.root = p.landmark()
	p.state = BodyState(p.in.Body, p.in.Archive, p.in.Dynamic)
}

func (p *page) byBlocks(blocks []rawBlock) *html.Node {
	if len(blocks) == 0 {
		return nil
	}
	runs := p.matchBlocks(blocks, 0, len(p.text.words))
	if float64(len(runs)) < rawShare*float64(len(blocks)) {
		return nil
	}
	if root := p.spanning(runs); root != nil && !named(root, "body", "html") {
		return root
	}
	return nil
}

func (p *page) fragmentBlocks(nodes []*html.Node) []rawBlock {
	for {
		holders := make([]*html.Node, 0, 1)
		for _, node := range nodes {
			if len(wordsOf(node)) > 0 {
				holders = append(holders, node)
			}
		}
		if len(holders) != 1 || !wraps(holders[0]) {
			break
		}
		nodes = make([]*html.Node, 0)
		for child := holders[0].FirstChild; child != nil; child = child.NextSibling {
			nodes = append(nodes, child)
		}
	}

	blocks := make([]rawBlock, 0, len(nodes))
	for i, node := range nodes {
		if words := wordsOf(node); len(words) > 0 {
			blocks = append(blocks, rawBlock{index: i, words: words, links: p.linkKeys(node)})
		}
	}
	return blocks
}

func wraps(node *html.Node) bool {
	if node.Type != html.ElementNode {
		return false
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode && content.IsBlockElement(child.Data) {
			return true
		}
	}
	return false
}

func (p *page) setRoot(root *html.Node) {
	p.root = root
	p.state = stateOf(p.in.Archive, p.body, true)
}

func BodyState(body block.Body, archive bool, dynamic []string) RootState {
	return stateOf(archive, technicalBody(body, dynamic), false)
}

func stateOf(archive bool, body block.Body, located bool) RootState {
	switch {
	case archive:
		return RootArchive
	case body.Empty():
		return RootEmptyRaw
	case body.DynamicOnly():
		return RootDynamicOnly
	case located:
		return RootFound
	default:
		return RootNotFound
	}
}

func technicalBody(body block.Body, names []string) block.Body {
	blocks := slices.Clone(body.Blocks)
	for i := range blocks {
		if !blocks[i].Dynamic && technical(&blocks[i], names) {
			blocks[i].Dynamic = true
		}
	}
	return block.Body{Kind: body.Kind, Blocks: blocks}
}

func technical(b *block.Block, names []string) bool {
	if b.Name != "" && slices.Contains(names, b.Name) {
		return true
	}
	match := loneShortcode.FindStringSubmatch(strings.TrimSpace(b.Text()))
	return len(match) == 3 && slices.Contains(names, match[1]) && (match[2] == "" || match[2] == match[1])
}

func (p *page) landmark() *html.Node {
	if found := p.first(isMain); found != nil {
		return found
	}
	if found := p.first(func(n *html.Node) bool { return named(n, "article") }); found != nil {
		return found
	}
	if found := p.first(func(n *html.Node) bool { return named(n, "body") }); found != nil {
		return found
	}
	return p.doc
}

func (p *page) widest(node *html.Node) *html.Node {
	words := p.text.count(node)
	for up := node.Parent; up != nil && up.Type == html.ElementNode && !named(up, "body", "html"); up = up.Parent {
		if p.text.count(up) != words {
			break
		}
		node = up
	}
	return node
}
