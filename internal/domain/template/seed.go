package template

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"
)

//go:embed seed/*.json seed/superseded/*.json
var seedFiles embed.FS

const (
	seedPattern       = "seed/*.json"
	supersededPattern = "seed/superseded/*.json"
)

type seedFile struct {
	Name     string       `json:"name"`
	PageKind string       `json:"pageKind"`
	Spec     TemplateSpec `json:"spec"`
}

func Seed() []Template {
	return seedsAt(seedPattern, true)
}

func Superseded() []Template {
	return seedsAt(supersededPattern, false)
}

func Supersedes(seed, stored Template) bool {
	if stored.Scope != ScopeGlobal || !strings.EqualFold(stored.Name, seed.Name) || stored.PageKind != seed.PageKind {
		return false
	}
	if sameSpec(stored.Spec, seed.Spec) {
		return false
	}
	for _, earlier := range Superseded() {
		if strings.EqualFold(earlier.Name, seed.Name) && sameSpec(stored.Spec, earlier.Spec) {
			return true
		}
	}
	return false
}

func sameSpec(a, b TemplateSpec) bool {
	left, leftErr := json.Marshal(a)
	right, rightErr := json.Marshal(b)
	return leftErr == nil && rightErr == nil && bytes.Equal(left, right)
}

func seedsAt(pattern string, strict bool) []Template {
	names, err := fs.Glob(seedFiles, pattern)
	if err != nil {
		panic(fmt.Errorf("list seed templates: %w", err))
	}

	out := make([]Template, 0, len(names))
	for _, name := range names {
		body, readErr := seedFiles.ReadFile(name)
		if readErr != nil {
			panic(fmt.Errorf("read seed template %s: %w", name, readErr))
		}

		var file seedFile
		decoder := json.NewDecoder(bytes.NewReader(body))
		if strict {
			decoder.DisallowUnknownFields()
		}
		if decodeErr := decoder.Decode(&file); decodeErr != nil {
			panic(fmt.Errorf("decode seed template %s: %w", name, decodeErr))
		}
		out = append(out, Template{Scope: ScopeGlobal, Name: file.Name, PageKind: file.PageKind, Version: 1, Spec: file.Spec})
	}
	return out
}
