package field

import (
	"slices"
	"strconv"
	"strings"
)

func IDList(ids []int64) string {
	seen := make(map[int64]struct{}, len(ids))
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		if _, held := seen[id]; held {
			continue
		}
		seen[id] = struct{}{}
		parts = append(parts, strconv.FormatInt(id, 10))
	}
	return strings.Join(parts, ",")
}

func ParseIDList(text string) ([]int64, bool) {
	if text == "" {
		return []int64{}, true
	}
	parts := strings.Split(text, ",")
	ids := make([]int64, 0, len(parts))
	for _, part := range parts {
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil || id <= 0 || part != strconv.FormatInt(id, 10) {
			return nil, false
		}
		if !slices.Contains(ids, id) {
			ids = append(ids, id)
		}
	}
	return ids, true
}
