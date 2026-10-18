package site

import "time"

const (
	URLFormAuto     = "auto"
	URLFormAbsolute = "absolute"
	URLFormRelative = "relative"

	defaultRenderConcurrency = 4
	maxRenderConcurrency     = 16
	defaultFullWalkEvery     = 10
)

type Settings struct {
	LinkURLForm         string           `json:"linkUrlForm"`
	UserAgent           string           `json:"userAgent"`
	Observe             ObserveSettings  `json:"observe"`
	Builders            []BuilderRule    `json:"builders"`
	TechnicalShortcodes []string         `json:"technicalShortcodes"`
	TechnicalBlocks     []string         `json:"technicalBlocks"`
	Technical           TechnicalPages   `json:"technical"`
	Observation         ObservationState `json:"observation"`
}

type ObserveSettings struct {
	RenderConcurrency int `json:"renderConcurrency"`
	FullWalkEvery     int `json:"fullWalkEvery"`
}

type BuilderRule struct {
	Name              string   `json:"name"`
	DataAttributes    []string `json:"dataAttributes"`
	ClassPrefixes     []string `json:"classPrefixes"`
	ShortcodePrefixes []string `json:"shortcodePrefixes"`
}

type TechnicalPages struct {
	Front    int64 `json:"front"`
	Posts    int64 `json:"posts"`
	Privacy  int64 `json:"privacy"`
	Shop     int64 `json:"shop"`
	Cart     int64 `json:"cart"`
	Checkout int64 `json:"checkout"`
	Account  int64 `json:"account"`
	Terms    int64 `json:"terms"`
}

type ObservationState struct {
	LastStartedAt    *time.Time `json:"lastStartedAt"`
	LastFinishedAt   *time.Time `json:"lastFinishedAt"`
	LastFullWalkAt   *time.Time `json:"lastFullWalkAt"`
	LastModifiedSeen *time.Time `json:"lastModifiedSeen"`
	SinceFullWalk    int        `json:"sinceFullWalk"`
}

func DefaultSettings() Settings {
	return Settings{
		LinkURLForm: URLFormAuto,
		Observe:     ObserveSettings{RenderConcurrency: defaultRenderConcurrency, FullWalkEvery: defaultFullWalkEvery},
		Builders: []BuilderRule{
			{Name: "elementor", DataAttributes: []string{"data-elementor-type", "data-elementor-id"}, ClassPrefixes: []string{"elementor-"}, ShortcodePrefixes: []string{}},
			{Name: "wpbakery", DataAttributes: []string{}, ClassPrefixes: []string{"vc_row", "vc_column"}, ShortcodePrefixes: []string{"vc_"}},
			{Name: "divi", DataAttributes: []string{}, ClassPrefixes: []string{"et_pb_"}, ShortcodePrefixes: []string{"et_pb_"}},
			{Name: "beaver-builder", DataAttributes: []string{}, ClassPrefixes: []string{"fl-builder"}, ShortcodePrefixes: []string{}},
			{Name: "oxygen", DataAttributes: []string{}, ClassPrefixes: []string{"ct-section"}, ShortcodePrefixes: []string{"ct_"}},
			{Name: "bricks", DataAttributes: []string{}, ClassPrefixes: []string{"brxe-"}, ShortcodePrefixes: []string{}},
			{Name: "breakdance", DataAttributes: []string{}, ClassPrefixes: []string{"breakdance"}, ShortcodePrefixes: []string{}},
		},
		TechnicalShortcodes: []string{"woocommerce_cart", "woocommerce_checkout", "woocommerce_my_account"},
		TechnicalBlocks:     []string{"woocommerce/cart", "woocommerce/checkout"},
	}
}

func (s Settings) Normalized() Settings {
	defaults := DefaultSettings()
	if s.LinkURLForm == "" {
		s.LinkURLForm = defaults.LinkURLForm
	}
	if s.Observe.RenderConcurrency == 0 {
		s.Observe.RenderConcurrency = defaults.Observe.RenderConcurrency
	}
	if s.Observe.FullWalkEvery == 0 {
		s.Observe.FullWalkEvery = defaults.Observe.FullWalkEvery
	}
	if s.Builders == nil {
		s.Builders = defaults.Builders
	}
	if s.TechnicalShortcodes == nil {
		s.TechnicalShortcodes = defaults.TechnicalShortcodes
	}
	if s.TechnicalBlocks == nil {
		s.TechnicalBlocks = defaults.TechnicalBlocks
	}
	return s
}

func (s Settings) Validate() error {
	switch s.LinkURLForm {
	case URLFormAuto, URLFormAbsolute, URLFormRelative:
	default:
		return invalid("link url form must be auto, absolute or relative", "settings.linkUrlForm")
	}
	if s.Observe.RenderConcurrency < 1 || s.Observe.RenderConcurrency > maxRenderConcurrency {
		return invalid("render concurrency must be between 1 and 16", "settings.observe.renderConcurrency")
	}
	if s.Observe.FullWalkEvery < 1 {
		return invalid("a full walk must come at least every observation", "settings.observe.fullWalkEvery")
	}
	for i := range s.Builders {
		if s.Builders[i].Name == "" {
			return invalid("a builder rule needs a name", "settings.builders")
		}
	}
	return nil
}
