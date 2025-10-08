package dbx

import (
	"github.com/Masterminds/squirrel"
	"github.com/lann/builder"
)

var ST = squirrel.StatementBuilderType(builder.EmptyBuilder).PlaceholderFormat(squirrel.Dollar)
