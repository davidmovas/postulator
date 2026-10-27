package content_test

import (
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/content"
)

func TestAMustLinkTargetIsRequiredWithItsAnchorFirst(t *testing.T) {
	t.Parallel()

	lc := content.LinkContext{PageID: "p", Targets: []content.LinkTarget{
		{PageID: "hub", URL: "/hub/", Anchors: []string{"the hub", "hubs"}, Relation: content.RelationUp},
		{PageID: "sibling", URL: "/sibling/", Anchors: []string{"a sibling"}, Relation: content.RelationSibling},
	}}
	insisted, unplanned := content.Insist(lc, []content.Must{
		{PageID: "sibling", Anchor: "the closest sibling"}, {PageID: "hub", Anchor: "hubs"}, {PageID: "elsewhere", Anchor: "x"},
	})
	if !slices.Equal(unplanned, []string{"elsewhere"}) {
		t.Errorf("unplanned = %v, want the target the graph does not connect", unplanned)
	}
	sibling, _ := insisted.ByPageID("sibling")
	if !sibling.Required || !slices.Equal(sibling.Anchors, []string{"the closest sibling", "a sibling"}) {
		t.Errorf("the sibling = %+v, want it required with the directive's anchor first", sibling)
	}
	hub, _ := insisted.ByPageID("hub")
	if !hub.Required || !slices.Equal(hub.Anchors, []string{"hubs", "the hub"}) {
		t.Errorf("the hub = %+v, want an anchor it already held moved first, not repeated", hub)
	}
	if lc.Targets[0].Required || len(lc.Targets[1].Anchors) != 1 {
		t.Errorf("Insist changed the context it was given: %+v", lc.Targets)
	}
}
