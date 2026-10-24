package finding_test

import (
	"slices"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestANewFindingTakesItsCodesSeverityAndOptions(t *testing.T) {
	t.Parallel()

	cases := map[finding.Code]struct {
		severity finding.Severity
		options  []finding.Option
	}{
		finding.ParentUnresolved:   {finding.Warning, []finding.Option{finding.Ignore}},
		finding.ParentAmbiguous:    {finding.Warning, []finding.Option{finding.Ignore}},
		finding.ParentCycle:        {finding.Warning, []finding.Option{finding.Ignore}},
		finding.ParentNameTwin:     {finding.Info, []finding.Option{finding.Ignore}},
		finding.RenderRedirected:   {finding.Info, []finding.Option{finding.Rescan, finding.Ignore}},
		finding.ParentDiffers:      {finding.Warning, []finding.Option{finding.AcceptSite, finding.AcceptWorkbook, finding.Ignore}},
		finding.PluginOutdated:     {finding.Warning, []finding.Option{finding.Ignore}},
		finding.FieldChangedOnSite: {finding.Warning, []finding.Option{finding.AcceptSite, finding.RewriteField, finding.Ignore}},
		finding.BodyChangedOnSite:  {finding.Warning, []finding.Option{finding.AcceptSite, finding.RelinkPage, finding.Ignore}},
		finding.RenderUnreachable:  {finding.Warning, []finding.Option{finding.Rescan, finding.Ignore}},
		finding.BuilderDetected:    {finding.Info, []finding.Option{finding.Ignore}},
		finding.TechnicalDetected:  {finding.Info, []finding.Option{finding.SetType, finding.Ignore}},
		finding.PageTitleDiffers:   {finding.Warning, []finding.Option{finding.AcceptSite, finding.RewriteField, finding.Ignore}},
		finding.PageH1Differs:      {finding.Warning, []finding.Option{finding.AcceptSite, finding.Ignore}},
		finding.URLFormMixed:       {finding.Info, []finding.Option{finding.Ignore}},
		finding.EntityCardProposed: {finding.Info, []finding.Option{finding.Accept, finding.Ignore}},
		finding.RowAmbiguous:       {finding.Warning, []finding.Option{finding.MapToPage, finding.Ignore}},
		finding.RowWithoutPage:     {finding.Info, []finding.Option{finding.MapToPage, finding.Ignore}},
	}
	at := time.Date(2026, time.October, 17, 12, 0, 0, 0, time.UTC)
	page := "page-1"
	for code, want := range cases {
		got, err := finding.New(finding.Finding{
			ID: "f", SiteID: "site", PageID: &page, Code: code, SubjectKey: finding.SubjectPage(page), CreatedAt: at, UpdatedAt: at,
		})
		if err != nil {
			t.Fatalf("New(%s): %v", code, err)
		}
		if got.Severity != want.severity {
			t.Errorf("%s severity = %s, want %s", code, got.Severity, want.severity)
		}
		if !slices.Equal(got.Options, want.options) {
			t.Errorf("%s options = %v, want %v", code, got.Options, want.options)
		}
		if got.Status != finding.Open || string(got.Subject) != "{}" {
			t.Errorf("%s status = %s subject = %s, want open and {}", code, got.Status, got.Subject)
		}
	}
}

func TestAnOptionIsAModelChangeOrARun(t *testing.T) {
	t.Parallel()

	cases := map[finding.Option]finding.OptionKind{
		finding.Ignore:         finding.ModelChange,
		finding.Accept:         finding.ModelChange,
		finding.AcceptSite:     finding.ModelChange,
		finding.AcceptWorkbook: finding.ModelChange,
		finding.MapToPage:      finding.ModelChange,
		finding.SetType:        finding.ModelChange,
		finding.RewriteField:   finding.Run,
		finding.RelinkPage:     finding.Run,
		finding.Rescan:         finding.Run,
	}
	for option, want := range cases {
		if got := option.Kind(); got != want {
			t.Errorf("%s is a %s, want a %s", option, got, want)
		}
	}
	if kind := finding.Option("made_up").Kind(); kind != "" {
		t.Errorf("an unknown option is a %q, want none", kind)
	}
	if listed := finding.Options(); len(listed) != len(cases) || !slices.IsSorted(listed) {
		t.Errorf("Options() = %v, want every option once, sorted", listed)
	}
}

func TestOnlyAModelChangeWithoutAChoicePerFindingIsTakenInBulk(t *testing.T) {
	t.Parallel()

	for _, option := range finding.Options() {
		want := slices.Contains([]finding.Option{finding.Ignore, finding.AcceptSite, finding.AcceptWorkbook, finding.SetType}, option)
		if option.Bulk() != want {
			t.Errorf("%s in bulk = %v, want %v", option, option.Bulk(), want)
		}
		if option.Bulk() && option.Kind() != finding.ModelChange {
			t.Errorf("%s is taken in bulk but is a %s", option, option.Kind())
		}
	}
}

func TestEveryCodeIsListedOnceAndOffersOnlyListedOptions(t *testing.T) {
	t.Parallel()

	codes := finding.Codes()
	if !slices.IsSorted(codes) || len(slices.Compact(slices.Clone(codes))) != len(codes) {
		t.Fatalf("Codes() = %v, want each code once, sorted", codes)
	}
	for _, code := range codes {
		if !code.Valid() {
			t.Errorf("%s is listed and not valid", code)
		}
		for _, option := range finding.OptionsOf(code) {
			if !slices.Contains(finding.Options(), option) {
				t.Errorf("%s offers %s, which Options() does not list", code, option)
			}
		}
	}
	if !slices.Contains(codes, finding.RowWithoutPage) || slices.Contains(codes, finding.Code("made_up")) {
		t.Errorf("Codes() = %v", codes)
	}
}

func TestAnUnknownCodeIsRefused(t *testing.T) {
	t.Parallel()

	valid := finding.Finding{ID: "f", SiteID: "site", Code: finding.PluginOutdated, SubjectKey: finding.SubjectSite}
	if _, err := finding.New(valid); err != nil {
		t.Fatalf("a valid finding is refused: %v", err)
	}
	cases := map[string]func(*finding.Finding){
		"an unknown code":     func(f *finding.Finding) { f.Code = "made_up" },
		"an empty subject":    func(f *finding.Finding) { f.SubjectKey = "" },
		"no site":             func(f *finding.Finding) { f.SiteID = "" },
		"no id":               func(f *finding.Finding) { f.ID = "" },
		"an unknown status":   func(f *finding.Finding) { f.Status = "parked" },
		"a foreign option":    func(f *finding.Finding) { f.Options = []finding.Option{finding.SetType} },
		"a malformed subject": func(f *finding.Finding) { f.Subject = []byte("{") },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			f := valid
			mutate(&f)
			if _, err := finding.New(f); !errors.IsCode(err, errors.Invalid) {
				t.Errorf("accepted: %v", err)
			}
		})
	}
}

func TestSubjectKeysNameWhatAFindingIsAbout(t *testing.T) {
	t.Parallel()

	if finding.SubjectPage("p") != "page:p" {
		t.Errorf("SubjectPage = %q", finding.SubjectPage("p"))
	}
	if finding.SubjectPageField("p", "title") != "page:p:field:title" {
		t.Errorf("SubjectPageField = %q", finding.SubjectPageField("p", "title"))
	}
	if finding.SubjectCrumb("p", "/healing-peptides/") != "page:p:crumb:/healing-peptides/" {
		t.Errorf("SubjectCrumb = %q", finding.SubjectCrumb("p", "/healing-peptides/"))
	}
	if finding.SubjectSite != "site" {
		t.Errorf("SubjectSite = %q", finding.SubjectSite)
	}
}
