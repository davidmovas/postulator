package sqlite_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

func edgeBetween(t *testing.T, siteID, from, to string, kind graph.EdgeKind, status graph.EdgeStatus, at time.Time) graph.Edge {
	t.Helper()
	edge, err := graph.NewEdge(graph.Edge{ID: id.New(), SiteID: siteID, FromEntityID: from, ToEntityID: to, Kind: kind, Weight: 0.6, Source: graph.SourceAI, Status: status, CreatedAt: at})
	if err != nil {
		t.Fatalf("NewEdge: %v", err)
	}
	return edge
}

func edgeIDs(edges []graph.Edge) []string {
	out := make([]string, 0, len(edges))
	for i := range edges {
		out = append(out, edges[i].ID)
	}
	return out
}

func TestEdgeRepoRoundTrip(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	hub := sqlitetest.Entity(t, store, owner.ID, "Shoes")
	child := sqlitetest.Entity(t, store, owner.ID, "Boots")
	repo := sqlite.NewEdgeRepo(store)

	want := edgeBetween(t, owner.ID, child.ID, hub.ID, graph.EdgeParent, graph.StatusProposed, sqlitetest.Stamp)
	if err := repo.Insert(t.Context(), want); err != nil {
		t.Fatalf("Insert: %v", err)
	}
	got, err := repo.Get(t.Context(), want.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Get = %+v\nwant %+v", got, want)
	}

	again := edgeBetween(t, owner.ID, child.ID, hub.ID, graph.EdgeParent, graph.StatusApproved, sqlitetest.Stamp)
	if err = repo.Insert(t.Context(), again); !errors.IsCode(err, errors.Conflict) {
		t.Errorf("duplicate edge code = %q, want CONFLICT", errors.CodeOf(err))
	}
	stranger := edgeBetween(t, owner.ID, child.ID, "no-such-entity", graph.EdgeParent, graph.StatusApproved, sqlitetest.Stamp)
	if err = repo.Insert(t.Context(), stranger); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("unknown endpoint code = %q, want INVALID", errors.CodeOf(err))
	}

	if err = repo.SetStatus(t.Context(), want.ID, graph.StatusApproved); err != nil {
		t.Fatalf("SetStatus: %v", err)
	}
	got, err = repo.Get(t.Context(), want.ID)
	if err != nil || got.Status != graph.StatusApproved {
		t.Errorf("status after SetStatus = %q, %v", got.Status, err)
	}
	if err = repo.SetStatus(t.Context(), "missing", graph.StatusRejected); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("SetStatus missing code = %q, want NOT_FOUND", errors.CodeOf(err))
	}

	if err = repo.Delete(t.Context(), want.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err = repo.Delete(t.Context(), want.ID); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("Delete twice code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if _, err = repo.Get(t.Context(), want.ID); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("Get missing code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
}

func TestEdgeRepoListAndFilters(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	hub := sqlitetest.Entity(t, store, owner.ID, "Shoes")
	boots := sqlitetest.Entity(t, store, owner.ID, "Boots")
	sandals := sqlitetest.Entity(t, store, owner.ID, "Sandals")
	repo := sqlite.NewEdgeRepo(store)

	edges := []graph.Edge{
		edgeBetween(t, owner.ID, boots.ID, hub.ID, graph.EdgeParent, graph.StatusApproved, sqlitetest.Stamp),
		edgeBetween(t, owner.ID, sandals.ID, hub.ID, graph.EdgeParent, graph.StatusProposed, sqlitetest.Stamp.Add(time.Minute)),
		edgeBetween(t, owner.ID, boots.ID, sandals.ID, graph.EdgeRelated, graph.StatusApproved, sqlitetest.Stamp.Add(2*time.Minute)),
	}
	for i := range edges {
		if err := repo.Insert(t.Context(), edges[i]); err != nil {
			t.Fatalf("Insert: %v", err)
		}
	}

	all, err := repo.ListBySite(t.Context(), owner.ID)
	if err != nil {
		t.Fatalf("ListBySite: %v", err)
	}
	if got := edgeIDs(all); !reflect.DeepEqual(got, edgeIDs(edges)) {
		t.Errorf("ListBySite = %v, want creation order", got)
	}

	bySite := graph.EdgeQuery{SiteID: owner.ID}
	first, err := repo.List(t.Context(), bySite, paging.Request{Limit: 2})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got := edgeIDs(first.Items); !reflect.DeepEqual(got, edgeIDs(edges[:2])) || !first.HasMore {
		t.Fatalf("first page = %v", got)
	}
	rest, err := repo.List(t.Context(), bySite, paging.Request{After: first.Next, Limit: 2})
	if err != nil {
		t.Fatalf("List after: %v", err)
	}
	if got := edgeIDs(rest.Items); !reflect.DeepEqual(got, edgeIDs(edges[2:])) || rest.HasMore {
		t.Fatalf("second page = %v", got)
	}
	back, err := repo.List(t.Context(), bySite, paging.Request{Before: rest.Prev, Limit: 2})
	if err != nil {
		t.Fatalf("List before: %v", err)
	}
	if got := edgeIDs(back.Items); !reflect.DeepEqual(got, edgeIDs(edges[:2])) {
		t.Fatalf("backward page = %v", got)
	}

	relatedKind := graph.EdgeRelated
	related, err := repo.List(t.Context(), graph.EdgeQuery{SiteID: owner.ID, Kind: &relatedKind}, paging.Request{Limit: 10})
	if err != nil || len(related.Items) != 1 || related.Items[0].Kind != graph.EdgeRelated {
		t.Errorf("related filter = %+v, %v", related.Items, err)
	}
	proposed := graph.StatusProposed
	pending, err := repo.List(t.Context(), graph.EdgeQuery{SiteID: owner.ID, Status: &proposed}, paging.Request{Limit: 10})
	if err != nil || len(pending.Items) != 1 || pending.Items[0].ID != edges[1].ID {
		t.Errorf("status filter = %+v, %v", pending.Items, err)
	}
	touching, err := repo.List(t.Context(), graph.EdgeQuery{SiteID: owner.ID, EntityID: sandals.ID}, paging.Request{Limit: 10})
	if err != nil || len(touching.Items) != 2 {
		t.Errorf("entity filter must match either endpoint, got %d, %v", len(touching.Items), err)
	}
	newest, err := repo.List(t.Context(), graph.EdgeQuery{SiteID: owner.ID, Desc: true}, paging.Request{Limit: 1})
	if err != nil || len(newest.Items) != 1 || newest.Items[0].ID != edges[2].ID {
		t.Errorf("descending = %+v, %v", newest.Items, err)
	}
}
