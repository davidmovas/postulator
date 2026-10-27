package render

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

type RootState string

const (
	RootFound       RootState = "found"
	RootEmptyRaw    RootState = "empty_raw"
	RootDynamicOnly RootState = "dynamic_only"
	RootNotFound    RootState = "not_found"
	RootArchive     RootState = "archive"
)

type RegionKind string

const (
	RegionBreadcrumb     RegionKind = "breadcrumb"
	RegionListing        RegionKind = "listing"
	RegionNavigation     RegionKind = "navigation"
	RegionContent        RegionKind = "content"
	RegionContentDynamic RegionKind = "content_dynamic"
	RegionGenerated      RegionKind = "generated"
	RegionUnknown        RegionKind = "unknown"
)

const (
	SignalJSONLD    = "jsonld"
	SignalARIA      = "aria"
	SignalMicrodata = "microdata"
	SignalLandmark  = "landmark"
	SignalLinks     = "links"
	SignalRel       = "rel"
)

const (
	URLFormAbsolute = "absolute"
	URLFormRelative = "relative"
)

type Region struct {
	Kind   RegionKind `json:"kind"`
	Inside bool       `json:"inside"`
	Links  int        `json:"links"`
	Words  int        `json:"words"`
	Signal string     `json:"signal"`
}

type Link struct {
	Href     string           `json:"href"`
	Path     string           `json:"path"`
	Kind     pagemap.LinkKind `json:"kind"`
	Anchor   string           `json:"anchor"`
	Type     pagemap.LinkType `json:"type"`
	Region   RegionKind       `json:"region"`
	Position int              `json:"position"`
}

type Heading struct {
	Level     int    `json:"level"`
	Text      string `json:"text"`
	Generated bool   `json:"generated"`
}

type Builder struct {
	Name              string
	DataAttributes    []string
	ClassPrefixes     []string
	ShortcodePrefixes []string
}

type Input struct {
	Public   string
	Filtered string
	Body     block.Body
	Site     pagemap.Site
	Document string
	Builders []Builder
	Lookup   func(path string) (pagemap.WPType, bool)
	Archive  bool
	Dynamic  []string
}

type Extraction struct {
	Hash           string                       `json:"hash"`
	Root           RootState                    `json:"root"`
	Regions        []Region                     `json:"regions"`
	Trails         []pagemap.Trail              `json:"trails"`
	JSONLD         map[string][]json.RawMessage `json:"jsonld"`
	Links          []Link                       `json:"links"`
	Headings       []Heading                    `json:"headings"`
	WordsContent   int                          `json:"wordsContent"`
	WordsGenerated int                          `json:"wordsGenerated"`
	Builder        string                       `json:"builder"`
	URLForm        string                       `json:"urlForm"`
	Robots         string                       `json:"robots"`
	ThemeH1        ThemeH1                      `json:"themeH1"`
	H1             string                       `json:"h1"`
	H1Source       H1Source                     `json:"h1Source"`
}

func (e Extraction) NoIndex() bool {
	return strings.Contains(e.Robots, "noindex")
}

type Record struct {
	PageID string `json:"pageId"`
	Extraction
	ExtractedAt time.Time `json:"extractedAt"`
	FinalPath   string    `json:"finalPath"`
}
