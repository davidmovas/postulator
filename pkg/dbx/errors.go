package dbx

import (
	"database/sql"
	stdErrors "errors"
	"strings"
)

func IsNoRows(err error) bool {
	if err == nil {
		return false
	}

	return stdErrors.Is(err, sql.ErrNoRows)
}

func IsUniqueViolation(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique constraint failed") ||
		strings.Contains(msg, "primary key must be unique") ||
		(strings.Contains(msg, "constraint") && strings.Contains(msg, "unique"))
}

func IsForeignKeyViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "foreign key constraint failed")
}

func IsNotNullViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not null constraint failed")
}

func IsConstraintViolation(err error) bool {
	if err == nil {
		return false
	}
	if IsUniqueViolation(err) || IsForeignKeyViolation(err) || IsNotNullViolation(err) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "constraint failed") || strings.Contains(msg, "check constraint failed")
}

func IsDatabaseLocked(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "database is locked") || strings.Contains(msg, "database table is locked")
}

func IsDatabaseBusy(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "database is busy")
}

func IsTableAlreadyExists(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "table ") && strings.Contains(msg, " already exists")
}

func IsNoSuchTable(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no such table")
}

func IsNoSuchColumn(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no such column")
}

func IsReadonly(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "attempt to write a readonly")
}
