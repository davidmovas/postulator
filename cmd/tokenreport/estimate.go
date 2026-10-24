package main

import (
	"cmp"
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"math"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/davidmovas/postulator/internal/adapters/llm/catalog"
	"github.com/davidmovas/postulator/internal/application/sitemodel"
	"github.com/davidmovas/postulator/internal/application/sitemodel/sitemodeltest"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	stepWriter     = "writer"
	stepWriterPack = "writer's pack"
	stepLinker     = "linker call"
	stepJudge      = "judge"
	stepJudgePack  = "judge's pack"
	stepMeta       = "meta"

	spreadWriter   = "writer input, real ÷ the run estimate"
	spreadLinker   = "linker call, real ÷ the run estimate"
	spreadJudge    = "judge input, real ÷ the run estimate"
	spreadMeta     = "meta input, real ÷ the run estimate"
	spreadPack     = "pack of the writer, real ÷ llm.Tokens"
	spreadEscaped  = "body as JSON-escaped HTML, tokens per word"
	spreadSections = "section of a writer's pack, real ÷ llm.Tokens"

	minProseWords     = 50
	ceilingWords      = 3.0
	packRunesPerToken = 3.7
	tolerance         = 0.1
	estimateTolerance = 0.25
	reportedDecimal   = 100
	wholeBudget       = 1 << 20
)

func fitBudgets() []int {
	return []int{4096, 3000, 1500, 1000, 750, 500, 300}
}

type estimateReport struct {
	Inputs   []inputRow `json:"inputs"`
	Fits     []fitRow   `json:"fits"`
	Words    []wordsRow `json:"words"`
	Spreads  []spread   `json:"spreads"`
	Proposal []string   `json:"proposal"`
	Notes    []string   `json:"notes"`
}

type inputRow struct {
	Stand     string  `json:"stand"`
	Path      string  `json:"path"`
	Step      string  `json:"step"`
	Estimated int     `json:"estimated"`
	Real      int     `json:"real"`
	Ratio     float64 `json:"ratio"`
	Counted   bool    `json:"counted"`
}

type fitRow struct {
	Stand     string `json:"stand"`
	Path      string `json:"path"`
	Budget    int    `json:"budget"`
	Estimated int    `json:"estimated"`
	Real      int    `json:"real"`
	Whole     int    `json:"whole"`
	Dropped   int    `json:"dropped"`
	Over      bool   `json:"over"`
	Refused   string `json:"refused,omitempty"`
}

type wordsRow struct {
	Stand   string `json:"stand"`
	Path    string `json:"path"`
	Words   int    `json:"words"`
	Text    int    `json:"text"`
	HTML    int    `json:"html"`
	Escaped int    `json:"escaped"`
}

type spread struct {
	Name   string  `json:"name"`
	Count  int     `json:"count"`
	Min    float64 `json:"min"`
	Median float64 `json:"median"`
	Max    float64 `json:"max"`
	Pooled float64 `json:"pooled"`
	Over   int     `json:"over,omitempty"`
}

type tally struct {
	ratios  []float64
	counts  int
	against int
}

func (t *tally) add(counts, against int) {
	t.ratios = append(t.ratios, float64(counts)/float64(max(against, 1)))
	t.counts += counts
	t.against += against
}

func (t *tally) spread(name string) spread {
	if len(t.ratios) == 0 {
		return spread{Name: name}
	}
	sorted := slices.Clone(t.ratios)
	slices.Sort(sorted)
	middle := len(sorted) / 2
	median := sorted[middle]
	if len(sorted)%2 == 0 {
		median = (sorted[middle-1] + sorted[middle]) / 2
	}
	return spread{
		Name: name, Count: len(sorted), Min: sorted[0], Median: median, Max: sorted[len(sorted)-1],
		Pooled: float64(t.counts) / float64(max(t.against, 1)),
	}
}

func measureEstimate(ctx context.Context, held *counters, models *catalog.Catalog, rows []pageRow) (estimateReport, error) {
	writer, err := models.Default(llm.RoleWriter)
	if err != nil {
		return estimateReport{}, err
	}
	c, err := held.of(writer)
	if err != nil {
		return estimateReport{}, err
	}
	var built estimateReport
	built.Inputs = inputsOf(rows)
	if built.Fits, err = fitsOf(ctx, c); err != nil {
		return estimateReport{}, err
	}
	if built.Words, err = wordsOf(c); err != nil {
		return estimateReport{}, err
	}
	built.Spreads = spreadsOf(built, rows)
	built.Proposal = proposalOf(built, rows)
	built.Notes = notes()
	return built, nil
}

func inputsOf(rows []pageRow) []inputRow {
	out := make([]inputRow, 0)
	for i := range rows {
		row := &rows[i]
		if row.Skipped != "" {
			continue
		}
		add := func(step string, estimated, counts int, counted bool) {
			out = append(out, inputRow{
				Stand: row.Stand, Path: row.Path, Step: step, Estimated: estimated, Real: counts,
				Ratio: float64(counts) / float64(max(estimated, 1)), Counted: counted,
			})
		}
		add(stepWriter, row.Writer.Estimated, row.Writer.Total.Tokens, true)
		add(stepWriterPack, row.Writer.Pack.Estimated, row.Writer.Pack.Tokens, true)
		if row.Linker.Call != nil {
			add(stepLinker, row.Linker.Call.Estimated, row.Linker.Call.Total.Tokens, !row.NoBody)
		}
		add(stepJudge, row.Judge.Estimated, row.Judge.Total.Tokens, true)
		add(stepJudgePack, row.Judge.Pack.Estimated, row.Judge.Pack.Tokens, true)
		add(stepMeta, row.Meta.Estimated, row.Meta.Total.Tokens, true)
	}
	return out
}

func fitsOf(ctx context.Context, c counter) ([]fitRow, error) {
	out := make([]fitRow, 0)
	for _, name := range sitemodeltest.Stands() {
		model, err := sitemodeltest.Stand(name)
		if err != nil {
			return nil, err
		}
		service := model.Service()
		pages := slices.DeleteFunc(slices.Clone(model.Pages), func(page pagemap.Page) bool { return page.Status == pagemap.StatusArchived })
		slices.SortFunc(pages, func(a, b pagemap.Page) int { return cmp.Compare(a.Path, b.Path) })
		for i := range pages {
			whole, err := fitOf(ctx, c, service, name, pages[i], wholeBudget)
			if err != nil {
				return nil, err
			}
			for _, budget := range fitBudgets() {
				row, err := fitOf(ctx, c, service, name, pages[i], budget)
				if err != nil {
					return nil, err
				}
				row.Whole = whole.Real
				out = append(out, row)
			}
		}
	}
	return out, nil
}

func fitOf(ctx context.Context, c counter, service *sitemodel.Service, stand string, page pagemap.Page, budget int) (fitRow, error) {
	row := fitRow{Stand: stand, Path: page.Path, Budget: budget}
	seen, err := service.Picture(ctx, sitemodel.PackRequest{PageID: page.ID, Budget: budget})
	if refused, typed := stderrors.AsType[*errors.Error](err); typed && refused.Details["reason"] == contextpack.ReasonBudgetTooSmall {
		row.Refused = refused.Message
		return row, nil
	}
	if err != nil {
		return fitRow{}, err
	}
	counts, err := c.count(seen.Fitted.Text())
	if err != nil {
		return fitRow{}, err
	}
	row.Estimated, row.Real, row.Dropped, row.Over = seen.Fitted.Tokens, counts, len(seen.Fitted.Dropped), counts > budget
	return row, nil
}

func wordsOf(c counter) ([]wordsRow, error) {
	out := make([]wordsRow, 0)
	seen := map[string]bool{}
	for _, name := range sitemodeltest.Stands() {
		model, err := sitemodeltest.Stand(name)
		if err != nil {
			return nil, err
		}
		paths := map[string]string{}
		for i := range model.Pages {
			paths[model.Pages[i].ID] = model.Pages[i].Path
		}
		for i := range model.Bodies {
			body := &model.Bodies[i]
			if body.Side != block.Current || seen[block.BodyHash(body.Raw)] {
				continue
			}
			row, prose, err := proseOf(c, name, paths[body.PageID], body.Raw)
			if err != nil {
				return nil, err
			}
			if prose {
				seen[block.BodyHash(body.Raw)] = true
				out = append(out, row)
			}
		}
	}
	slices.SortFunc(out, func(a, b wordsRow) int { return cmp.Compare(a.Path, b.Path) })
	return out, nil
}

func proseOf(c counter, stand, path, raw string) (wordsRow, bool, error) {
	doc, err := content.Parse(raw)
	if err != nil {
		return wordsRow{}, false, err
	}
	words := doc.Words()
	if len(words) < minProseWords {
		return wordsRow{}, false, nil
	}
	answered, err := withoutComments(raw)
	if err != nil {
		return wordsRow{}, false, err
	}
	var escaped strings.Builder
	encoder := json.NewEncoder(&escaped)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(answered); err != nil {
		return wordsRow{}, false, err
	}
	row := wordsRow{Stand: stand, Path: path, Words: len(words)}
	for _, counted := range []struct {
		into *int
		text string
	}{{&row.Text, doc.Text()}, {&row.HTML, answered}, {&row.Escaped, strings.TrimSuffix(escaped.String(), "\n")}} {
		if *counted.into, err = c.count(counted.text); err != nil {
			return wordsRow{}, false, err
		}
	}
	return row, true, nil
}

func withoutComments(raw string) (string, error) {
	nodes, err := html.ParseFragment(strings.NewReader(raw), &html.Node{Type: html.ElementNode, Data: "body", DataAtom: atom.Body})
	if err != nil {
		return "", err
	}
	var out strings.Builder
	for _, node := range nodes {
		if node.Type == html.CommentNode {
			continue
		}
		dropComments(node)
		if err := html.Render(&out, node); err != nil {
			return "", err
		}
	}
	return out.String(), nil
}

func dropComments(node *html.Node) {
	for child := node.FirstChild; child != nil; {
		next := child.NextSibling
		if child.Type == html.CommentNode {
			node.RemoveChild(child)
		} else {
			dropComments(child)
		}
		child = next
	}
}

func spreadsOf(built estimateReport, rows []pageRow) []spread {
	inputs := []struct{ step, name string }{
		{stepWriter, spreadWriter},
		{stepLinker, spreadLinker},
		{stepJudge, spreadJudge},
		{stepMeta, spreadMeta},
		{stepWriterPack, spreadPack},
		{stepJudgePack, "pack of the judge, real ÷ llm.Tokens"},
	}
	shapes := []struct {
		name  string
		count func(wordsRow) int
	}{
		{"body text, tokens per word", func(w wordsRow) int { return w.Text }},
		{"body HTML, tokens per word", func(w wordsRow) int { return w.HTML }},
		{spreadEscaped, func(w wordsRow) int { return w.Escaped }},
	}
	out := make([]spread, 0, len(inputs)+1+len(fitBudgets())+len(shapes))

	for _, input := range inputs {
		var counted tally
		for _, row := range built.Inputs {
			if row.Step == input.step && row.Counted {
				counted.add(row.Real, row.Estimated)
			}
		}
		out = append(out, counted.spread(input.name))
	}

	var sections tally
	for i := range rows {
		if rows[i].Writer == nil {
			continue
		}
		for _, section := range rows[i].Writer.Sections {
			sections.add(section.Tokens, section.Estimated)
		}
	}
	out = append(out, sections.spread(spreadSections))

	for _, budget := range fitBudgets() {
		var fitted tally
		over := 0
		for _, fit := range built.Fits {
			if fit.Budget != budget || fit.Refused != "" {
				continue
			}
			fitted.add(fit.Real, budget)
			if fit.Over {
				over++
			}
		}
		at := fitted.spread("Fit at " + strconv.Itoa(budget) + ", real ÷ budget")
		at.Over = over
		out = append(out, at)
	}

	for _, shape := range shapes {
		var words tally
		for _, row := range built.Words {
			words.add(shape.count(row), row.Words)
		}
		out = append(out, words.spread(shape.name))
	}
	return out
}

func named(spreads []spread, name string) spread {
	for _, s := range spreads {
		if s.Name == name {
			return s
		}
	}
	return spread{}
}

func rounded(value float64) float64 {
	return math.Round(value*reportedDecimal) / reportedDecimal
}

func proposalOf(built estimateReport, rows []pageRow) []string {
	out := make([]string, 0)
	words := 0
	for _, row := range built.Words {
		words += row.Words
	}
	answer := float64(llm.WordTokens(reportedDecimal)) / reportedDecimal
	escaped := named(built.Spreads, spreadEscaped)
	sample := fmt.Sprintf("%d bodies of %d words or more, %d words in all, %.2f to %.2f", escaped.Count, minProseWords, words, escaped.Min, escaped.Max)
	switch {
	case escaped.Count == 0:
		out = append(out, "no captured body is long enough to say anything about tokens per word")
	case math.Abs(escaped.Pooled-answer) > answer*tolerance:
		out = append(out, fmt.Sprintf("llm.WordTokens prices an answer at %.1f tokens per word: the captured prose, as a writer answers it (JSON-escaped HTML), "+
			"runs %.2f (%s); it stays at %.1f on purpose, the output side, until a funded key's ledger shows how long the model's answers run",
			answer, escaped.Pooled, sample, answer))
	default:
		out = append(out, fmt.Sprintf("llm.WordTokens' %.1f tokens per word holds: the captured prose answers at %.2f (%s)", answer, escaped.Pooled, sample))
	}
	if escaped.Count > 0 && escaped.Max > ceilingWords {
		out = append(out, fmt.Sprintf("steps.tokensPerWord, the writer's ceiling of %.0f tokens per word: propose %.0f, the densest body runs %.2f",
			ceilingWords, math.Ceil(escaped.Max), escaped.Max))
	} else if escaped.Count > 0 {
		out = append(out, fmt.Sprintf("steps.tokensPerWord, the writer's ceiling of %.0f tokens per word, holds: the densest body runs %.2f", ceilingWords, escaped.Max))
	}

	pack := named(built.Spreads, spreadPack)
	packWords, packRunes, packTokens := 0, 0, 0
	for i := range rows {
		if rows[i].Writer != nil {
			packWords, packRunes, packTokens = packWords+rows[i].Writer.Pack.Words, packRunes+rows[i].Writer.Pack.Runes, packTokens+rows[i].Writer.Pack.Tokens
		}
	}
	perWord, perToken := float64(packTokens)/float64(max(packWords, 1)), float64(packRunes)/float64(max(packTokens, 1))
	read := float64(llm.ReadTokens(reportedDecimal)) / reportedDecimal
	if math.Abs(pack.Pooled-1) > tolerance {
		out = append(out, fmt.Sprintf("llm.Tokens prices a pack at %.1f tokens per word or %.1f runes per token, line by line, whichever is larger: the real count is %.2f of it; "+
			"the packs run %.2f tokens per word and %.2f runes per token, so propose %.1f and %.1f", read, packRunesPerToken, pack.Pooled, perWord, perToken,
			rounded(perWord*10)/10, rounded(perToken*10)/10))
	} else {
		out = append(out, fmt.Sprintf("llm.Tokens' %.1f tokens per word and %.1f runes per token hold within %.0f%%: the packs run %.2f and %.2f",
			read, packRunesPerToken, tolerance*100, perWord, perToken))
	}

	for _, step := range []struct{ name, said string }{
		{spreadWriter, "the writer's input"},
		{spreadLinker, "a linker call"},
		{spreadJudge, "the judge's input"},
		{spreadMeta, "the meta writer's input"},
	} {
		priced := named(built.Spreads, step.name)
		if priced.Count == 0 {
			continue
		}
		verdict := "within"
		if math.Abs(1/priced.Min-1) > estimateTolerance || math.Abs(1/priced.Max-1) > estimateTolerance {
			verdict = "OUTSIDE"
		}
		out = append(out, fmt.Sprintf("the run estimate prices %s at %.2f times the real count, pooled (%.2f to %.2f), %s the ±%.0f%% the tests hold",
			step.said, 1/priced.Pooled, 1/priced.Max, 1/priced.Min, verdict, estimateTolerance*100))
	}
	return append(out,
		"the input side was adopted on 2026-10-24 from these measurements: llm.Tokens' own rates, each step's own prompt overhead, "+
			"no share of the answer, and the judge's body at llm.ReadTokens; the output side stays conservative until a funded key's ledger settles it")
}

func notes() []string {
	return []string{
		"OpenAI renders a function definition and a structured answer's schema in its own internal format before the model reads it, so the JSON tokens counted here approximate a schema's real cost: close, not exact",
		"the ledger already records the real input_tokens of every call: when a funded key has run, compare it with these counts",
		"the judge reads a stand-in body, the captured prose cut to the template's target words, because a run judges the body its writer has just written, " +
			"not the page's current one; its estimate prices that body at llm.ReadTokens, and the meta writer reads a stand-in draft of the template's headings",
		"a linker call over a page with no body is listed and left out of its spread; the writer, the judge and the meta writer are always counted",
		"one product is measured, the replica's /product/bpc-157-peptide/, which its capture gave an entity through the site's proposal; " +
			"the other products are unmapped, and the steps refuse a page with no entity",
		"the briefs carry no link totals, because a capture holds no link audit",
		"bodies under " + strconv.Itoa(minProseWords) + " words are left out of the tokens-per-word sample, and a body both stands carry is counted once; " +
			"their HTML is counted without the block editor's comments, as a writer answers it, and as a JSON string with only what JSON must escape",
		"Fit keeps a pack under its budget by llm.Tokens; its spread covers every pack at each budget, so the real count says whether the pack is under in fact",
	}
}

func writeEstimate(out io.Writer, built estimateReport) error {
	return tables(out, func(s *sheet) {
		s.heading("THE ESTIMATE AGAINST THE REAL COUNT (tokens)", "step", "estimated", "real", "real ÷ estimated", "")
		for _, input := range built.Inputs {
			said := ""
			if !input.Counted {
				said = "no body, not in the spread"
			}
			s.row(input.Stand+" "+input.Path, input.Step, strconv.Itoa(input.Estimated), strconv.Itoa(input.Real), ratioOf(input.Ratio), said)
		}
	}, func(s *sheet) {
		s.heading("FIT AGAINST ITS BUDGET, WHERE IT CUT OR WENT OVER (tokens)", "budget", "llm.Tokens", "real", "sections cut", "")
		whole := 0
		for _, fit := range built.Fits {
			switch {
			case fit.Refused != "":
				s.row(fit.Stand+" "+fit.Path, strconv.Itoa(fit.Budget), "", "", "", "refused: "+fit.Refused)
			case fit.Dropped == 0 && !fit.Over:
				whole++
			default:
				said := "under by " + strconv.Itoa(fit.Budget-fit.Real)
				if fit.Over {
					said = "OVER by " + strconv.Itoa(fit.Real-fit.Budget)
				}
				s.row(fit.Stand+" "+fit.Path, strconv.Itoa(fit.Budget), strconv.Itoa(fit.Estimated), strconv.Itoa(fit.Real), strconv.Itoa(fit.Dropped), said)
			}
		}
		s.row(strconv.Itoa(whole) + " more packs fit whole and under their budget")
	}, func(s *sheet) {
		s.heading("TOKENS PER WORD OF THE CAPTURED PROSE", "words", "text", "HTML", "JSON-escaped HTML")
		for _, row := range built.Words {
			s.row(row.Stand+" "+row.Path, strconv.Itoa(row.Words), perWord(row.Text, row.Words), perWord(row.HTML, row.Words), perWord(row.Escaped, row.Words))
		}
	}, func(s *sheet) {
		s.heading("THE SPREAD", "count", "min", "median", "max", "pooled", "over")
		for _, sp := range built.Spreads {
			over := ""
			if sp.Over > 0 {
				over = strconv.Itoa(sp.Over)
			}
			s.row(sp.Name, strconv.Itoa(sp.Count), ratioOf(sp.Min), ratioOf(sp.Median), ratioOf(sp.Max), ratioOf(sp.Pooled), over)
		}
	}, func(s *sheet) {
		s.heading("PROPOSED")
		for _, line := range built.Proposal {
			s.row("- " + line)
		}
	}, func(s *sheet) {
		s.heading("NOTES")
		for i, line := range built.Notes {
			s.row(strconv.Itoa(i+1) + ". " + line)
		}
	})
}

func ratioOf(value float64) string {
	return strconv.FormatFloat(value, 'f', 2, 64)
}

func perWord(tokens, words int) string {
	return strconv.Itoa(tokens) + " (" + ratioOf(float64(tokens)/float64(words)) + ")"
}
