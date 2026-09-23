package template

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
)

//go:embed seed/*.json
var seedFiles embed.FS

type seedFile struct {
	Name     string       `json:"name"`
	PageKind string       `json:"pageKind"`
	Spec     TemplateSpec `json:"spec"`
}

func Seed() []Template {
	names, err := fs.Glob(seedFiles, "seed/*.json")
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
		decoder.DisallowUnknownFields()
		if decodeErr := decoder.Decode(&file); decodeErr != nil {
			panic(fmt.Errorf("decode seed template %s: %w", name, decodeErr))
		}
		out = append(out, Template{Scope: ScopeGlobal, Name: file.Name, PageKind: file.PageKind, Version: 1, Spec: file.Spec})
	}
	return out
}
