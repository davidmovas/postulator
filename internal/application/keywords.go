package application

import (
	"strconv"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/keyword"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func KeywordViews(list keyword.List) []dto.Keyword {
	views := make([]dto.Keyword, 0, len(list))
	for _, item := range keyword.New(list) {
		views = append(views, dto.Keyword{Text: item.Text, Volume: item.Volume})
	}
	return views
}

func KeywordList(items []dto.Keyword, field string) (keyword.List, error) {
	listed := make([]keyword.Keyword, 0, len(items))
	for at, item := range items {
		where := field + "[" + strconv.Itoa(at) + "]"
		switch {
		case strings.TrimSpace(item.Text) == "":
			return nil, errors.New(errors.Invalid, "a keyword needs its phrase").WithDetail("field", where+".text")
		case item.Volume != nil && *item.Volume < 0:
			return nil, errors.New(errors.Invalid, "a search volume must not be negative").WithDetail("field", where+".volume")
		}
		listed = append(listed, keyword.Keyword{Text: item.Text, Volume: item.Volume})
	}
	return keyword.New(listed), nil
}
