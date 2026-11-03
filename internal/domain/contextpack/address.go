package contextpack

import (
	"cmp"
	"slices"
	"strings"
)

const RootFolder = "/"

type Folder struct {
	Path     string `json:"path"`
	PageID   string `json:"pageId,omitempty"`
	Children int    `json:"children"`
	Pages    int    `json:"pages"`
}

func IsFolder(path string) bool {
	if path == RootFolder {
		return true
	}
	return len(path) > 2 && strings.HasPrefix(path, "/") && strings.HasSuffix(path, "/") && !strings.Contains(path, "//")
}

func (r MapRow) Addressed() bool {
	return r.Row || r.HubID != ""
}

func Folders(rows []MapRow, under string) []Folder {
	rows = slices.DeleteFunc(slices.Clone(rows), func(row MapRow) bool { return !row.Addressed() })
	if under == "" {
		if len(rows) == 0 {
			return []Folder{}
		}
		root := Folder{Path: RootFolder, Children: len(folded(rows, RootFolder)), Pages: len(rows)}
		for i := range rows {
			if rows[i].Path == RootFolder {
				root.PageID = rows[i].PageID
			}
		}
		return []Folder{root}
	}
	tallied := folded(rows, under)
	out := make([]Folder, 0, len(tallied))
	for _, held := range tallied {
		out = append(out, Folder{Path: held.path, PageID: held.pageID, Children: len(held.below), Pages: held.pages})
	}
	slices.SortFunc(out, func(a, b Folder) int { return cmp.Compare(a.Path, b.Path) })
	return out
}

type tally struct {
	path   string
	pageID string
	pages  int
	below  map[string]struct{}
}

func folded(rows []MapRow, under string) map[string]*tally {
	out := make(map[string]*tally)
	for i := range rows {
		path := rows[i].Path
		if path == under || !strings.HasPrefix(path, under) {
			continue
		}
		rest := path[len(under):]
		segment, deeper, _ := strings.Cut(rest, "/")
		if segment == "" {
			continue
		}
		folder := under + segment + "/"
		held, known := out[folder]
		if !known {
			held = &tally{path: folder, below: map[string]struct{}{}}
			out[folder] = held
		}
		held.pages++
		if path == folder {
			held.pageID = rows[i].PageID
		}
		if next, _, _ := strings.Cut(deeper, "/"); next != "" {
			held.below[next] = struct{}{}
		}
	}
	return out
}
