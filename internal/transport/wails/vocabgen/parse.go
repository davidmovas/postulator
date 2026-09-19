package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func moduleRoot(start string) (string, error) {
	dir := start
	if dir == "" {
		working, err := os.Getwd()
		if err != nil {
			return "", errors.Wrap(err, errors.Internal, "read the working directory")
		}
		dir = working
	}

	absolute, err := filepath.Abs(dir)
	if err != nil {
		return "", errors.Wrap(err, errors.Internal, "resolve the working directory")
	}

	for {
		if _, statErr := os.Stat(filepath.Join(absolute, "go.mod")); statErr == nil {
			return absolute, nil
		}
		parent := filepath.Dir(absolute)
		if parent == absolute {
			return "", errors.New(errors.NotFound, "the generator must run inside the module").
				WithDetail("from", dir)
		}
		absolute = parent
	}
}

func sourceFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, errors.Wrap(err, errors.NotFound, "read the package directory "+dir)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		names = append(names, name)
	}
	slices.Sort(names)
	return names, nil
}

func constValues(dir, typeName string) ([]string, error) {
	names, err := sourceFiles(dir)
	if err != nil {
		return nil, err
	}

	fset := token.NewFileSet()
	values := make([]string, 0, 8)
	for _, name := range names {
		file, parseErr := parser.ParseFile(fset, filepath.Join(dir, name), nil, 0)
		if parseErr != nil {
			return nil, errors.Wrap(parseErr, errors.Invalid, "parse "+filepath.Join(dir, name))
		}
		found, collectErr := constValuesOf(file, typeName)
		if collectErr != nil {
			return nil, collectErr
		}
		values = append(values, found...)
	}

	if len(values) == 0 {
		return nil, errors.New(errors.NotFound, "no string constant of the named type is declared").
			WithDetail("package", dir).WithDetail("type", typeName)
	}
	return values, nil
}

func constValuesOf(file *ast.File, typeName string) ([]string, error) {
	values := make([]string, 0, 8)
	for _, decl := range file.Decls {
		block, ok := decl.(*ast.GenDecl)
		if !ok || block.Tok != token.CONST {
			continue
		}
		for _, spec := range block.Specs {
			declared, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			named, ok := declared.Type.(*ast.Ident)
			if !ok || named.Name != typeName {
				continue
			}
			for _, expr := range declared.Values {
				literal, ok := expr.(*ast.BasicLit)
				if !ok || literal.Kind != token.STRING {
					return nil, errors.New(errors.Invalid, "a vocabulary constant must be a string literal").
						WithDetail("type", typeName)
				}
				unquoted, err := strconv.Unquote(literal.Value)
				if err != nil {
					return nil, errors.Wrap(err, errors.Invalid, "read the constant "+literal.Value)
				}
				values = append(values, unquoted)
			}
		}
	}
	return values, nil
}
