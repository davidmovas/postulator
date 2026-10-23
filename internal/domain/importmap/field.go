package importmap

import "github.com/davidmovas/postulator/internal/kernel/errors"

type Field string

const (
	FieldPath            Field = "path"
	FieldTitle           Field = "title"
	FieldH1              Field = "h1"
	FieldPrimaryKeyword  Field = "primary_keyword"
	FieldKeywords        Field = "keywords"
	FieldAnchors         Field = "anchors"
	FieldEntity          Field = "entity"
	FieldPageType        Field = "page_type"
	FieldParentEntity    Field = "parent_entity"
	FieldRelated         Field = "related"
	FieldMetaTitle       Field = "meta_title"
	FieldMetaDescription Field = "meta_description"
	FieldWPType          Field = "wp_type"
	FieldOwnEntity       Field = "own_entity"
	FieldSKU             Field = "sku"
)

var fields = []Field{
	FieldPath, FieldTitle, FieldH1, FieldPrimaryKeyword, FieldKeywords, FieldAnchors, FieldEntity,
	FieldPageType, FieldParentEntity, FieldRelated, FieldMetaTitle, FieldMetaDescription, FieldWPType, FieldOwnEntity,
	FieldSKU,
}

func Fields() []Field {
	out := make([]Field, len(fields))
	copy(out, fields)
	return out
}

func (f Field) Valid() bool {
	for _, known := range fields {
		if known == f {
			return true
		}
	}
	return false
}

func invalid(message, field string) *errors.Error {
	return errors.New(errors.Invalid, message).WithDetail("field", field)
}
