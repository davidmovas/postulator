package observe_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/application/observe"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/render"
)

func TestACrumbResolvesThroughTheRedirectItsLinkLedTo(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	s.serveTheSite(false)
	s.serve(tb500Page(false, crumb{name: "Home", path: "/"}, crumb{name: "Peptides", path: "/old-peptides/"}, crumb{name: "TB-500", path: "/tb-500/"}))
	s.visitor.redirects[baseURL+"/old-peptides/"] = baseURL + "/peptides/"

	s.run(t, observe.Options{})

	tb500 := s.pageAt(t, "/tb-500/")
	peptides := s.pageAt(t, "/peptides/")
	if tb500.ParentPageID == nil || *tb500.ParentPageID != peptides.ID || tb500.ParentReason != pagemap.ParentJSONLD || tb500.ParentMethod != pagemap.MethodRedirect {
		t.Errorf("tb-500's parent = %v %s by %q, want the hub its crumb redirects to", tb500.ParentPageID, tb500.ParentReason, tb500.ParentMethod)
	}
	lookups := 0
	for _, looked := range s.visitor.lookedUp() {
		if looked == baseURL+"/old-peptides/" {
			lookups++
		}
	}
	if lookups != 1 {
		t.Errorf("the crumb was looked up %d times, want once for the JSON-LD list and the nav together", lookups)
	}
}

func TestAPageWhoseVisitLandsElsewhereKeepsNothingOfThatPage(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	s.serveTheSite(false)
	s.visitor.redirects[baseURL+"/account/"] = baseURL + "/shop/"

	s.run(t, observe.Options{})

	account := s.pageAt(t, "/account/")
	record := s.stores.renders[account.ID]
	if record.FinalPath != "/shop/" || record.Root != render.RootDynamicOnly || len(record.Trails) != 0 || len(record.Regions) != 0 || record.Robots != "" {
		t.Errorf("the account's render = %+v, want only its body's state and where it landed", record)
	}
	if links := s.stores.links[account.ID][pagemap.SourceRender]; len(links) != 0 {
		t.Errorf("the account keeps %d render links of the shop", len(links))
	}
	raised, ok := s.finding(finding.RenderRedirected, finding.SubjectPage(account.ID))
	if !ok || raised.Status != finding.Open || string(raised.Subject) != `{"target":"/shop/"}` {
		t.Errorf("render_redirected = %+v %t", raised, ok)
	}

	delete(s.visitor.redirects, baseURL+"/account/")
	s.run(t, observe.Options{Full: true, RenderRescan: true})
	if cleared, _ := s.finding(finding.RenderRedirected, finding.SubjectPage(account.ID)); cleared.Status != finding.Resolved {
		t.Errorf("render_redirected once the visit stays = %s, want resolved", cleared.Status)
	}
	if s.stores.renders[account.ID].FinalPath != "" {
		t.Error("the render kept its old landing")
	}
}

func TestAFailedCrumbLookupLeavesThePageUnreachableForTheWalk(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	s.serveTheSite(false)
	s.visitor.failed[baseURL+"/healing-peptides/"] = true

	state := s.run(t, observe.Options{})

	bpc := s.pageAt(t, "/bpc-157/")
	if _, rendered := s.stores.renders[bpc.ID]; rendered || state.Counts.Unreachable == 0 {
		t.Errorf("bpc-157 rendered %t with %d unreachable, want no render after its crumb could not be looked up", rendered, state.Counts.Unreachable)
	}
	if f, ok := s.finding(finding.RenderUnreachable, finding.SubjectPage(bpc.ID)); !ok || f.Status != finding.Open {
		t.Errorf("render_unreachable for bpc-157 = %+v %t", f, ok)
	}
}
