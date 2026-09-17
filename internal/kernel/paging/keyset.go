package paging

import (
	"slices"
	"strings"

	"github.com/Masterminds/squirrel"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type Keyset[T any] struct {
	ID       func(T) string
	IDColumn string
	Keys     []SortKey[T]
	Desc     bool
}

func (k Keyset[T]) order() Order {
	if k.Desc {
		return Desc
	}
	return Asc
}

func (k Keyset[T]) fields() string {
	names := make([]string, 0, len(k.Keys))
	for _, key := range k.Keys {
		names = append(names, key.Field)
	}
	return strings.Join(names, ",")
}

func (k Keyset[T]) validate() error {
	switch {
	case len(k.Keys) == 0:
		return errors.New(errors.Internal, "keyset declares no sort keys")
	case k.IDColumn == "":
		return errors.New(errors.Internal, "keyset declares no tie-breaker column")
	case k.ID == nil:
		return errors.New(errors.Internal, "keyset declares no id accessor")
	}

	for _, key := range k.Keys {
		if key.Field == "" || key.Column == "" {
			return errors.New(errors.Internal, "keyset sort key needs a field and a column")
		}
		if key.Value == nil {
			return errors.New(errors.Internal, "keyset sort key "+key.Field+" needs a value accessor")
		}
	}
	return nil
}

func (k Keyset[T]) Encode(item T) (Cursor, error) {
	if err := k.validate(); err != nil {
		return "", err
	}

	values := make([]any, 0, len(k.Keys))
	for _, key := range k.Keys {
		value, ok := key.literal(key.Value(item))
		if !ok {
			return "", errors.New(errors.Internal, "sort key "+key.Field+" holds a value that cannot be encoded in a cursor")
		}
		values = append(values, value)
	}

	return encodeCursor(Position{
		OrderBy: k.fields(),
		Order:   k.order(),
		Values:  values,
		ID:      k.ID(item),
	})
}

func (k Keyset[T]) Position(c Cursor) (Position, error) {
	if err := k.validate(); err != nil {
		return Position{}, err
	}

	position, err := decodeCursor(c)
	if err != nil {
		return Position{}, err
	}
	if position.ID == "" {
		return Position{}, errors.New(errors.Invalid, "cursor carries no row id")
	}
	if position.OrderBy != k.fields() {
		return Position{}, errors.New(errors.Invalid, "cursor was issued for a different sort order")
	}
	if position.Order != k.order() {
		return Position{}, errors.New(errors.Invalid, "cursor was issued for a different sort direction")
	}
	if len(position.Values) != len(k.Keys) {
		return Position{}, errors.New(errors.Invalid, "cursor carries the wrong number of sort values")
	}

	coerced := make([]any, 0, len(k.Keys))
	for i, key := range k.Keys {
		value, ok := key.literal(position.Values[i])
		if !ok {
			return Position{}, errors.New(errors.Invalid, "cursor value for "+key.Field+" does not match the sort key type")
		}
		coerced = append(coerced, value)
	}
	position.Values = coerced

	return position, nil
}

func (k Keyset[T]) Apply(builder squirrel.SelectBuilder, request Request) (squirrel.SelectBuilder, error) {
	if err := k.validate(); err != nil {
		return builder, err
	}
	if request.After != "" && request.Before != "" {
		return builder, errors.New(errors.Invalid, "a page request carries either after or before, never both")
	}

	request = request.Normalize()
	order := k.order()
	if request.backward() {
		order = order.reversed()
	}

	if cursor := request.cursor(); cursor != "" {
		position, err := k.Position(cursor)
		if err != nil {
			return builder, err
		}
		builder = builder.Where(squirrel.Expr(k.predicate(order), append(position.Values, position.ID)...))
	}

	return builder.OrderBy(k.orderBy(order)...).Limit(uint64(request.Limit) + 1), nil
}

func (k Keyset[T]) predicate(order Order) string {
	columns := make([]string, 0, len(k.Keys)+1)
	for _, key := range k.Keys {
		columns = append(columns, key.Column)
	}
	columns = append(columns, k.IDColumn)

	placeholders := strings.TrimSuffix(strings.Repeat("?, ", len(columns)), ", ")
	return "(" + strings.Join(columns, ", ") + ") " + order.comparison() + " (" + placeholders + ")"
}

func (k Keyset[T]) orderBy(order Order) []string {
	clauses := make([]string, 0, len(k.Keys)+1)
	for _, key := range k.Keys {
		clauses = append(clauses, key.Column+" "+order.sql())
	}
	return append(clauses, k.IDColumn+" "+order.sql())
}

func (k Keyset[T]) Cut(rows []T, request Request) (List[T], error) {
	if err := k.validate(); err != nil {
		return List[T]{}, err
	}

	request = request.Normalize()
	backward := request.backward()

	hasMore := len(rows) > request.Limit
	if hasMore {
		rows = rows[:request.Limit]
	}
	if backward {
		rows = slices.Clone(rows)
		slices.Reverse(rows)
	}

	list := List[T]{Items: rows, HasMore: hasMore}
	if len(rows) == 0 {
		return list, nil
	}

	first, last := rows[0], rows[len(rows)-1]

	if backward {
		next, err := k.Encode(last)
		if err != nil {
			return List[T]{}, err
		}
		list.Next = next
		if hasMore {
			prev, err := k.Encode(first)
			if err != nil {
				return List[T]{}, err
			}
			list.Prev = prev
		}
		return list, nil
	}

	if hasMore {
		next, err := k.Encode(last)
		if err != nil {
			return List[T]{}, err
		}
		list.Next = next
	}
	if request.After != "" {
		prev, err := k.Encode(first)
		if err != nil {
			return List[T]{}, err
		}
		list.Prev = prev
	}
	return list, nil
}
