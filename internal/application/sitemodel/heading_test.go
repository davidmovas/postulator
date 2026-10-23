package sitemodel_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/domain/render"
)

func TestTheReadLayerSaysWhetherOurH1IsOwed(t *testing.T) {
	t.Parallel()

	heading := func(pageID string, h1 render.ThemeH1) render.Record {
		return render.Record{PageID: pageID, Extraction: render.Extraction{ThemeH1: h1}}
	}
	planned := sitePage("new", "/new/", "New", 0, pagemap.WPPage, pagetype.Entity, "")
	planned.WPID, planned.Status = nil, pagemap.StatusPlanned
	archived := sitePage("old", "/old/", "Old", 40, pagemap.WPPage, pagetype.Entity, "")
	archived.Status = pagemap.StatusArchived

	cases := []struct {
		name    string
		pageID  string
		renders []render.Record
		want    bool
	}{
		{name: "an existing page whose theme prints the title", pageID: "bpc", renders: []render.Record{heading("bpc", render.ThemeH1Printed), heading("hub", render.ThemeH1Absent)}},
		{name: "an existing page that prints none", pageID: "bpc", renders: []render.Record{heading("bpc", render.ThemeH1Absent), heading("hub", render.ThemeH1Printed)}, want: true},
		{name: "an existing page never rendered", pageID: "bpc", renders: []render.Record{heading("hub", render.ThemeH1Absent)}},
		{name: "a page we create among pages of its type that print none", pageID: "new", renders: []render.Record{
			heading("hub", render.ThemeH1Absent), heading("bpc", render.ThemeH1Absent), heading("tb", render.ThemeH1Printed), heading("old", render.ThemeH1Printed),
		}, want: true},
		{name: "a page we create where one page of its type prints the title", pageID: "new", renders: []render.Record{
			heading("hub", render.ThemeH1Absent), heading("far", render.ThemeH1Printed),
		}},
		{name: "a page we create with nothing observed", pageID: "new"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := site1()
			m.Pages = append(m.Pages, planned, archived)
			m.Renders = tc.renders
			owed, err := m.Service().OwesH1(t.Context(), tc.pageID)
			if err != nil || owed != tc.want {
				t.Errorf("OwesH1 = %t, %v; want %t", owed, err, tc.want)
			}
		})
	}
}
