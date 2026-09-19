package main

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"strconv"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const lineBudget = 110

type group struct {
	export string
	values []string
}

type block struct {
	export      string
	tsType      string
	aliasExport string
	aliasType   string
	values      []string
	derived     []group
}

func collect(root string) ([]block, error) {
	entries := catalog()
	blocks := make([]block, 0, len(entries))
	seen := make(map[string]struct{}, len(entries))

	for _, entry := range entries {
		if _, duplicate := seen[entry.export]; duplicate {
			return nil, errors.New(errors.Conflict, "the catalog exports a name twice").
				WithDetail("export", entry.export)
		}
		seen[entry.export] = struct{}{}

		if entry.aliasExport != "" {
			blocks = append(blocks, block{
				export: entry.export, tsType: entry.tsType,
				aliasExport: entry.aliasExport, aliasType: entry.aliasType,
			})
			continue
		}

		values, err := constValues(filepath.Join(root, filepath.FromSlash(entry.pkg)), entry.typeName)
		if err != nil {
			return nil, err
		}

		derived := make([]group, 0, len(entry.derived))
		for _, rule := range entry.derived {
			if _, duplicate := seen[rule.export]; duplicate {
				return nil, errors.New(errors.Conflict, "the catalog exports a name twice").
					WithDetail("export", rule.export)
			}
			seen[rule.export] = struct{}{}

			kept := make([]string, 0, len(values))
			for _, value := range values {
				if rule.keep(value) {
					kept = append(kept, value)
				}
			}
			derived = append(derived, group{export: rule.export, values: kept})
		}

		blocks = append(blocks, block{export: entry.export, tsType: entry.tsType, values: values, derived: derived})
	}
	return blocks, nil
}

func render(w io.Writer, blocks []block) error {
	if len(blocks) == 0 {
		return errors.New(errors.Invalid, "the vocabulary module would be empty")
	}

	var out bytes.Buffer
	for _, entry := range blocks {
		if entry.aliasExport != "" {
			fmt.Fprintf(&out, "export const %s = %s;\n", entry.export, entry.aliasExport)
			fmt.Fprintf(&out, "export type %s = %s;\n\n", entry.tsType, entry.aliasType)
			continue
		}

		writeList(&out, "export const "+entry.export+" = ", entry.values, " as const;")
		fmt.Fprintf(&out, "export type %s = (typeof %s)[number];\n\n", entry.tsType, entry.export)

		for _, derived := range entry.derived {
			writeList(&out, "export const "+derived.export+": readonly "+entry.tsType+"[] = ", derived.values, ";")
			out.WriteString("\n")
		}
	}

	out.WriteString("export function isOneOf<T extends string>(values: readonly T[], candidate: string): candidate is T {\n")
	out.WriteString("    return (values as readonly string[]).includes(candidate);\n")
	out.WriteString("}\n")

	_, err := w.Write(out.Bytes())
	return err
}

func writeList(out *bytes.Buffer, prefix string, values []string, suffix string) {
	quoted := make([]string, 0, len(values))
	width := len(prefix) + len(suffix) + 2
	for _, value := range values {
		literal := strconv.Quote(value)
		quoted = append(quoted, literal)
		width += len(literal) + 2
	}

	out.WriteString(prefix)
	if width <= lineBudget {
		out.WriteString("[")
		for index, literal := range quoted {
			if index > 0 {
				out.WriteString(", ")
			}
			out.WriteString(literal)
		}
		out.WriteString("]")
		out.WriteString(suffix)
		out.WriteString("\n")
		return
	}

	out.WriteString("[\n")
	for _, literal := range quoted {
		out.WriteString("    ")
		out.WriteString(literal)
		out.WriteString(",\n")
	}
	out.WriteString("]")
	out.WriteString(suffix)
	out.WriteString("\n")
}
