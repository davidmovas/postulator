package application

import (
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

func PageRequest(req dto.ListRequest) paging.Request {
	return paging.Request{After: paging.Cursor(req.Cursor), Limit: req.Limit}
}

func MapList[T, V any](list paging.List[T], convert func(T) V) paging.List[V] {
	items := make([]V, 0, len(list.Items))
	for i := range list.Items {
		items = append(items, convert(list.Items[i]))
	}
	return paging.List[V]{Cursors: list.Cursors, Items: items, HasMore: list.HasMore}
}
