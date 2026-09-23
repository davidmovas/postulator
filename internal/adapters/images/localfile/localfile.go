package localfile

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/davidmovas/postulator/internal/adapters/images"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const defaultLimit = 3

type Source struct {
	dir string
}

func New(dir string) *Source {
	return &Source{dir: strings.TrimSpace(dir)}
}

func (s *Source) Pick(ctx context.Context, query images.Query) ([]images.Image, error) {
	if err := ctx.Err(); err != nil {
		return nil, errors.New(errors.Cancelled, "the local image search was cancelled").WithInternal(err)
	}
	if s.dir == "" {
		return nil, errors.New(errors.Invalid, "no local image folder is configured").
			WithDetail("setting", "images.localDir")
	}

	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, errors.Wrap(err, errors.External, "read the local image folder "+s.dir)
	}

	words := keywords(query.Term)
	limit := query.Limit
	if limit <= 0 {
		limit = defaultLimit
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !images.Supported(entry.Name()) {
			continue
		}
		if len(words) > 0 && !matches(entry.Name(), words) {
			continue
		}
		names = append(names, entry.Name())
	}
	slices.Sort(names)

	picked := make([]images.Image, 0, min(limit, len(names)))
	for _, name := range names {
		if len(picked) == limit {
			break
		}
		body, readErr := os.ReadFile(filepath.Join(s.dir, name))
		if readErr != nil {
			return nil, errors.Wrap(readErr, errors.External, "read the local image "+name)
		}
		picked = append(picked, images.Image{
			Filename:    name,
			ContentType: images.ContentType(name),
			Alt:         query.Term,
			Bytes:       body,
		})
	}
	return picked, nil
}

func keywords(term string) []string {
	fields := strings.FieldsFunc(strings.ToLower(term), func(symbol rune) bool {
		return (symbol < 'a' || symbol > 'z') && (symbol < '0' || symbol > '9')
	})

	out := make([]string, 0, len(fields))
	for _, field := range fields {
		if len(field) > 2 && !slices.Contains(out, field) {
			out = append(out, field)
		}
	}
	return out
}

func matches(name string, words []string) bool {
	normalized := strings.ToLower(strings.TrimSuffix(name, filepath.Ext(name)))
	for _, word := range words {
		if strings.Contains(normalized, word) {
			return true
		}
	}
	return false
}
