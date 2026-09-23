package application_test

import (
	"strconv"
	"testing"

	"github.com/davidmovas/postulator/internal/application"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

func TestPageRequest(t *testing.T) {
	t.Parallel()

	got := application.PageRequest(dto.ListRequest{Cursor: "abc", Limit: 25})
	if got.After != paging.Cursor("abc") || got.Before != "" || got.Limit != 25 {
		t.Fatalf("PageRequest = %+v", got)
	}
	if empty := application.PageRequest(dto.ListRequest{}); empty.After != "" || empty.Limit != 0 {
		t.Fatalf("PageRequest of an empty request = %+v", empty)
	}
}

func TestMapList(t *testing.T) {
	t.Parallel()

	in := paging.List[int]{Cursors: paging.Cursors{Next: "n", Prev: "p"}, Items: []int{1, 2}, HasMore: true}
	out := application.MapList(in, strconv.Itoa)
	if out.Next != "n" || out.Prev != "p" || !out.HasMore || len(out.Items) != 2 || out.Items[0] != "1" || out.Items[1] != "2" {
		t.Fatalf("MapList = %+v", out)
	}

	empty := application.MapList(paging.List[int]{}, strconv.Itoa)
	if empty.Items == nil || len(empty.Items) != 0 {
		t.Fatalf("MapList of an empty list must materialize an empty slice, got %#v", empty.Items)
	}
}
