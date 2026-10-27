//go:build !race

package scale_test

import (
	"strconv"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/application/pagetypes"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/kernel/dto"
)

const retypedBlock = "/c010/"

func (s *scaled) firstRule(t *testing.T, name, typeID string, conditions pagetype.Conditions) {
	t.Helper()

	saved, err := s.core.PageTypes.SaveRule(t.Context(), pagetypes.SaveRuleRequest{
		SiteID: s.siteID, Name: name, Enabled: true, TypeID: typeID, Conditions: conditions,
	})
	if err != nil {
		t.Fatalf("save the rule %q: %v", name, err)
	}
	listed, err := s.core.PageTypes.ListRules(t.Context(), pagetypes.ListRulesRequest{SiteID: s.siteID})
	if err != nil {
		t.Fatalf("list the rules: %v", err)
	}
	order := []string{saved.ID}
	for i := range listed.Rules {
		if listed.Rules[i].ID != saved.ID {
			order = append(order, listed.Rules[i].ID)
		}
	}
	if _, err = s.core.PageTypes.ReorderRules(t.Context(), pagetypes.ReorderRulesRequest{SiteID: s.siteID, RuleIDs: order}); err != nil {
		t.Fatalf("move the rule %q first: %v", name, err)
	}
}

func (s *scaled) retyped(t *testing.T) map[string]measured {
	t.Helper()

	guide := s.typeOf(t, pagetype.Guide)
	s.firstRule(t, "One block is Guide", guide, pagetype.Conditions{URLs: []pagetype.URLMatch{{Mode: pagetype.URLPrefix, Value: retypedBlock}}})
	out := map[string]measured{}
	out["classify: the facts after the walks"] = s.measure(t, "classify: the facts after the walks", func() error {
		_, err := s.core.SiteModel.ClassificationFacts(t.Context(), s.siteID)
		return err
	})
	var preview pagetypes.PreviewResponse
	out["classify: one block's preview"] = s.measure(t, "classify: one block's preview", func() error {
		var err error
		preview, err = s.core.Classification.Preview(t.Context(), pagetypes.PreviewRequest{SiteID: s.siteID})
		return err
	})
	if preview.Moving == 0 || len(preview.Groups) == 0 {
		t.Fatalf("the preview of a rule over %s = %+v, want the block's pages moving", retypedBlock, preview)
	}
	group := preview.Groups[0]
	out["classify: a group's first page"] = s.measure(t, "classify: a group's first page", func() error {
		_, err := s.core.Classification.PreviewPages(t.Context(), pagetypes.PreviewPagesRequest{
			SiteID: s.siteID, From: group.From, To: group.To, ListRequest: dto.ListRequest{Limit: listLimit},
		})
		return err
	})
	var applied pagetypes.ApplyResponse
	out["classify: one block's apply"] = s.once(t, "classify: one block's apply", func() error {
		var err error
		applied, err = s.core.Classification.Apply(t.Context(), pagetypes.ApplyRequest{SiteID: s.siteID, Hash: preview.Hash})
		return err
	})
	if applied.Moved != preview.Moving+preview.Refreshed || applied.Changed != 0 {
		t.Errorf("the apply of %s = %+v, want the %d pages its preview moved and refreshed", retypedBlock, applied, preview.Moving+preview.Refreshed)
	}
	s.writerWaits(t, s.typeOf(t, pagetype.Info))
	return out
}

func (s *scaled) writerWaits(t *testing.T, info string) {
	t.Helper()

	s.firstRule(t, "Every page is Info", info, pagetype.Conditions{URLs: []pagetype.URLMatch{{Mode: pagetype.URLGlob, Value: "/**"}}})
	preview, err := s.core.Classification.Preview(t.Context(), pagetypes.PreviewRequest{SiteID: s.siteID})
	if err != nil {
		t.Fatalf("preview every page as Info: %v", err)
	}
	technical := s.typeOf(t, pagetype.Technical)
	done := make(chan error, 1)
	started := time.Now()
	go func() {
		_, applyErr := s.core.Classification.Apply(t.Context(), pagetypes.ApplyRequest{SiteID: s.siteID, Hash: preview.Hash})
		done <- applyErr
	}()
	var longest time.Duration
	writes := 0
	for applying := true; applying; {
		select {
		case applyErr := <-done:
			applying = false
			if applyErr != nil {
				t.Fatalf("apply every page as Info: %v", applyErr)
			}
		default:
			began := time.Now()
			name := "Technical " + strconv.Itoa(writes)
			if _, saveErr := s.core.PageTypes.Save(t.Context(), pagetypes.SaveRequest{ID: technical, Name: &name}); saveErr != nil {
				t.Fatalf("a second writer during the apply: %v", saveErr)
			}
			longest = max(longest, time.Since(began))
			writes++
		}
	}
	t.Logf("an apply of %d pages at %d pages took %s in batches of %d; a second writer wrote %d times beside it and waited at most %s",
		preview.Moving+preview.Refreshed, s.pages(), time.Since(started).Round(time.Millisecond), pagetypes.ApplyBatch, writes,
		longest.Round(time.Millisecond))
}
