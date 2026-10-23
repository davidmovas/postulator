package observe

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

type findings struct {
	store  findingStore
	siteID string
	now    time.Time
	open   map[string]finding.Finding
}

func (s *Service) findings(ctx context.Context, siteID string, codes ...finding.Code) (*findings, error) {
	f := &findings{store: s.deps.Findings, siteID: siteID, now: s.now(), open: make(map[string]finding.Finding)}
	for _, code := range codes {
		open, err := s.deps.Findings.OpenBySite(ctx, siteID, code)
		if err != nil {
			return nil, err
		}
		for i := range open {
			f.open[string(code)+"|"+open[i].SubjectKey] = open[i]
		}
	}
	return f, nil
}

func (f *findings) raise(ctx context.Context, code finding.Code, pageID *string, subjectKey string, subject any, options ...finding.Option) error {
	encoded, err := json.Marshal(subject)
	if err != nil {
		return err
	}
	if open, raised := f.open[string(code)+"|"+subjectKey]; raised && string(open.Subject) == string(encoded) {
		return nil
	}
	record, err := finding.New(finding.Finding{
		ID: id.New(), SiteID: f.siteID, PageID: pageID, Code: code, SubjectKey: subjectKey, Subject: encoded,
		Options: options, CreatedAt: f.now, UpdatedAt: f.now,
	})
	if err != nil {
		return err
	}
	if err := f.store.Upsert(ctx, record); err != nil {
		return err
	}
	f.open[string(code)+"|"+subjectKey] = record
	return nil
}

func (f *findings) clear(ctx context.Context, code finding.Code, subjectKey string) error {
	key := string(code) + "|" + subjectKey
	if _, open := f.open[key]; !open {
		return nil
	}
	delete(f.open, key)
	return f.store.Clear(ctx, f.siteID, code, subjectKey, finding.ResolvedByObservation, finding.ResolutionObservedClear, f.now)
}

func (f *findings) openFor(code finding.Code, prefix string) []finding.Finding {
	out := make([]finding.Finding, 0)
	for key := range f.open {
		if f.open[key].Code == code && strings.HasPrefix(f.open[key].SubjectKey, prefix) {
			out = append(out, f.open[key])
		}
	}
	return out
}
