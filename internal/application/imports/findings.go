package imports

import (
	"cmp"
	"context"
	"encoding/json"
	"slices"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

type lasting struct {
	code    finding.Code
	key     string
	pageID  *string
	subject any
	options []finding.Option
}

type ambiguousRow struct {
	Sheet      string   `json:"sheet"`
	Rule       string   `json:"rule"`
	Candidates []string `json:"candidates"`
	PageIDs    []string `json:"pageIds"`
	Product    bool     `json:"product"`
}

type plannedRow struct {
	Sheet       string `json:"sheet"`
	PlannedPath string `json:"plannedPath"`
	Product     bool   `json:"product"`
}

var rowCodes = []finding.Code{finding.RowAmbiguous, finding.RowWithoutPage}

func (l *lasting) finding(siteID string, now time.Time) (finding.Finding, error) {
	subject, err := json.Marshal(l.subject)
	if err != nil {
		return finding.Finding{}, err
	}
	return finding.New(finding.Finding{
		ID: id.New(), SiteID: siteID, PageID: l.pageID, Code: l.code, SubjectKey: l.key, Subject: subject, Options: l.options,
		CreatedAt: now, UpdatedAt: now,
	})
}

func (p *plan) sawRow(sheet, key string) {
	seen, known := p.keys[sheet]
	if !known {
		seen = make(map[string]struct{})
		p.keys[sheet] = seen
	}
	if key != "" {
		seen[key] = struct{}{}
	}
}

func (p *plan) rowsOf(key string) (map[string]struct{}, bool) {
	for sheet, rows := range p.keys {
		if strings.HasPrefix(key, pagemap.RowKeyPrefix(sheet)) {
			return rows, true
		}
	}
	return nil, false
}

func (p *plan) keepAmbiguous(row *rowDraft) {
	if row.key == "" {
		return
	}
	candidates := slices.Clone(row.match.Candidates)
	slices.SortFunc(candidates, func(a, b pagemap.Page) int {
		return cmp.Or(strings.Compare(a.Path, b.Path), strings.Compare(a.ID, b.ID))
	})
	subject := ambiguousRow{
		Sheet: p.rowSheet(row.at), Rule: string(row.match.Rule), Candidates: make([]string, 0, len(candidates)),
		PageIDs: make([]string, 0, len(candidates)), Product: row.wantsProduct,
	}
	for i := range candidates {
		subject.Candidates = append(subject.Candidates, candidates[i].Path)
		subject.PageIDs = append(subject.PageIDs, candidates[i].ID)
		subject.Product = subject.Product && candidates[i].WPType == pagemap.WPProduct
	}
	p.lasting = append(p.lasting, lasting{code: finding.RowAmbiguous, key: row.key, subject: subject})
}

func (p *plan) keepPlanned(row *rowDraft, page *pagemap.Page) {
	if row.key == "" {
		return
	}
	pageID := page.ID
	p.lasting = append(p.lasting, lasting{
		code: finding.RowWithoutPage, key: row.key, pageID: &pageID,
		subject: plannedRow{Sheet: p.rowSheet(row.at), PlannedPath: page.Path, Product: page.WPType == pagemap.WPProduct},
	})
}

func (p *plan) keepTrail(page *pagemap.Page, diff pagemap.TrailDiff) {
	pageID := page.ID
	p.lasting = append(p.lasting, lasting{
		code: finding.ParentDiffers, key: finding.SubjectPage(page.ID), pageID: &pageID, subject: diff,
		options: pagemap.TrailOptions(page.WPType),
	})
}

func (b *builder) keepPlanned(draft *pageDraft, page *pagemap.Page) {
	if draft.generated || len(draft.rows) == 0 {
		return
	}
	b.p.keepPlanned(&b.rows[draft.rows[0]], page)
}

func (s *Service) keep(ctx context.Context, p *plan) error {
	now := s.now()
	raised := make(map[finding.Code]map[string]struct{}, len(p.lasting))
	for i := range p.lasting {
		held, err := p.lasting[i].finding(p.siteID, now)
		if err != nil {
			return err
		}
		if upsertErr := s.deps.Findings.Upsert(ctx, held); upsertErr != nil {
			return upsertErr
		}
		if raised[held.Code] == nil {
			raised[held.Code] = make(map[string]struct{})
		}
		raised[held.Code][held.SubjectKey] = struct{}{}
	}
	for _, pageID := range p.agreed {
		if err := s.deps.Findings.Clear(ctx, p.siteID, finding.ParentDiffers, finding.SubjectPage(pageID),
			finding.ResolvedByImport, finding.ResolutionObservedClear, now); err != nil {
			return err
		}
	}
	return s.clearRows(ctx, p, raised, now)
}

func (s *Service) clearRows(ctx context.Context, p *plan, raised map[finding.Code]map[string]struct{}, now time.Time) error {
	for _, code := range rowCodes {
		open, err := s.deps.Findings.OpenBySite(ctx, p.siteID, code)
		if err != nil {
			return err
		}
		for i := range open {
			key := open[i].SubjectKey
			rows, read := p.rowsOf(key)
			if _, again := raised[code][key]; !read || again {
				continue
			}
			resolution := finding.ResolutionRowLeftSheet
			if _, still := rows[key]; still {
				resolution = finding.ResolutionRowMatched
			}
			if clearErr := s.deps.Findings.Clear(ctx, p.siteID, code, key, finding.ResolvedByImport, resolution, now); clearErr != nil {
				return clearErr
			}
		}
	}
	return nil
}
