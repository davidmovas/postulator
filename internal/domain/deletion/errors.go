package deletion

import (
	"fmt"
	"strings"
)

// DependencyType represents the type of entity that depends on the target
type DependencyType string

const (
	DepJob      DependencyType = "job"
	DepSite     DependencyType = "site"
	DepTopic    DependencyType = "topic"
	DepPrompt   DependencyType = "prompt"
	DepProvider DependencyType = "provider"
	DepCategory DependencyType = "category"
	DepArticle  DependencyType = "article"
)

// Dependency represents a single dependency that blocks deletion
type Dependency struct {
	Type DependencyType
	ID   int64
	Name string
}

// ConflictError is returned when deletion is blocked by existing dependencies
type ConflictError struct {
	EntityType   DependencyType
	EntityID     int64
	EntityName   string
	Dependencies []Dependency
}

func (e *ConflictError) Error() string {
	if len(e.Dependencies) == 0 {
		return fmt.Sprintf("cannot delete %s '%s' (id=%d): unknown dependency", e.EntityType, e.EntityName, e.EntityID)
	}

	depTypes := make(map[DependencyType]int)
	for _, dep := range e.Dependencies {
		depTypes[dep.Type]++
	}

	var parts []string
	for depType, count := range depTypes {
		if count == 1 {
			parts = append(parts, fmt.Sprintf("%d %s", count, depType))
		} else {
			parts = append(parts, fmt.Sprintf("%d %ss", count, depType))
		}
	}

	return fmt.Sprintf("cannot delete %s '%s': used by %s", e.EntityType, e.EntityName, strings.Join(parts, ", "))
}

// UserMessage returns a user-friendly error message
func (e *ConflictError) UserMessage() string {
	if len(e.Dependencies) == 0 {
		return fmt.Sprintf("Cannot delete this %s because it is being used", e.EntityType)
	}

	depTypes := make(map[DependencyType]int)
	for _, dep := range e.Dependencies {
		depTypes[dep.Type]++
	}

	var parts []string
	for depType, count := range depTypes {
		if count == 1 {
			parts = append(parts, fmt.Sprintf("%d %s", count, depType))
		} else {
			parts = append(parts, fmt.Sprintf("%d %ss", count, depType))
		}
	}

	return fmt.Sprintf("Cannot delete this %s because it is used by %s", e.EntityType, strings.Join(parts, ", "))
}

// DependencyNames returns names of all dependencies for detailed error info
func (e *ConflictError) DependencyNames() []string {
	names := make([]string, 0, len(e.Dependencies))
	for _, dep := range e.Dependencies {
		if dep.Name != "" {
			names = append(names, fmt.Sprintf("%s: %s", dep.Type, dep.Name))
		} else {
			names = append(names, fmt.Sprintf("%s #%d", dep.Type, dep.ID))
		}
	}
	return names
}

// NewConflictError creates a new deletion conflict error
func NewConflictError(entityType DependencyType, entityID int64, entityName string, deps []Dependency) *ConflictError {
	return &ConflictError{
		EntityType:   entityType,
		EntityID:     entityID,
		EntityName:   entityName,
		Dependencies: deps,
	}
}

// IsConflictError checks if the error is a deletion conflict error
func IsConflictError(err error) bool {
	_, ok := err.(*ConflictError)
	return ok
}
