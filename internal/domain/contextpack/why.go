package contextpack

import (
	"encoding/json"

	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

type WhyReason string

const (
	WhyRoot            WhyReason = "root"
	WhyParentArchived  WhyReason = "parent_archived"
	WhyCycleBroken     WhyReason = "cycle_broken"
	WhyCrumbAmbiguous  WhyReason = "crumb_ambiguous"
	WhyCrumbUnresolved WhyReason = "crumb_unresolved"
	WhyPlanned         WhyReason = "planned"
	WhyRenderFailed    WhyReason = "render_failed"
	WhyNotDerived      WhyReason = "not_derived"
	WhyNoSignal        WhyReason = "no_signal"
)

type ParentWhy struct {
	Reason     WhyReason `json:"reason"`
	Crumb      string    `json:"crumb,omitempty"`
	Candidates []string  `json:"candidates,omitempty"`
	Loop       []string  `json:"loop,omitempty"`
	Parent     string    `json:"parent,omitempty"`
	FindingID  string    `json:"findingId,omitempty"`
}

type ParentEvidence struct {
	Parent   *pagemap.Page
	Findings []finding.Finding
	Page     pagemap.Page
	Rendered bool
}

type crumbSubject struct {
	Name       string   `json:"name"`
	Path       string   `json:"path"`
	Candidates []string `json:"candidates"`
	Paths      []string `json:"paths"`
}

func WhyNoParent(e ParentEvidence) ParentWhy {
	switch {
	case e.Page.Path == pagemap.RootPath && onSite(e.Page):
		return ParentWhy{Reason: WhyRoot}
	case e.Parent != nil:
		return ParentWhy{Reason: WhyParentArchived, Parent: e.Parent.Path}
	}
	if held, subject, found := openOf(e.Findings, finding.ParentCycle); found {
		return ParentWhy{Reason: WhyCycleBroken, Loop: subject.Paths, FindingID: held.ID}
	}
	if held, subject, found := openOf(e.Findings, finding.ParentAmbiguous); found {
		return ParentWhy{Reason: WhyCrumbAmbiguous, Crumb: subject.Name, Candidates: subject.Candidates, FindingID: held.ID}
	}
	if held, subject, found := openOf(e.Findings, finding.ParentUnresolved); found {
		crumb := subject.Path
		if crumb == "" {
			crumb = subject.Name
		}
		return ParentWhy{Reason: WhyCrumbUnresolved, Crumb: crumb, FindingID: held.ID}
	}
	if !onSite(e.Page) {
		return ParentWhy{Reason: WhyPlanned}
	}
	if held, _, found := openOf(e.Findings, finding.RenderUnreachable); found {
		return ParentWhy{Reason: WhyRenderFailed, FindingID: held.ID}
	}
	if !e.Rendered {
		return ParentWhy{Reason: WhyNotDerived}
	}
	return ParentWhy{Reason: WhyNoSignal}
}

func openOf(findings []finding.Finding, code finding.Code) (finding.Finding, crumbSubject, bool) {
	for i := range findings {
		if findings[i].Code != code || findings[i].Status != finding.Open {
			continue
		}
		var subject crumbSubject
		if len(findings[i].Subject) > 0 && json.Unmarshal(findings[i].Subject, &subject) != nil {
			subject = crumbSubject{}
		}
		return findings[i], subject, true
	}
	return finding.Finding{}, crumbSubject{}, false
}
