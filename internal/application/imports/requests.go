package imports

type InspectRequest struct {
	SiteID string `json:"siteId"`
	Path   string `json:"path"`
}

type InspectResponse struct {
	Headers  []string   `json:"headers"`
	Sample   [][]string `json:"sample"`
	Rows     int        `json:"rows"`
	Detected Mapping    `json:"detected"`
	Saved    []Mapping  `json:"saved"`
}

type PreviewRequest struct {
	SiteID  string  `json:"siteId"`
	Path    string  `json:"path"`
	Mapping Mapping `json:"mapping"`
}

type PreviewResponse struct {
	Report PreviewReport `json:"report"`
}

type ApplyOptions struct {
	SaveMappingAs string `json:"saveMappingAs,omitempty"`
}

type ApplyRequest struct {
	SiteID  string       `json:"siteId"`
	Path    string       `json:"path"`
	Mapping Mapping      `json:"mapping"`
	Options ApplyOptions `json:"options"`
}

type ApplyResponse struct {
	Report PreviewReport `json:"report"`
	Counts Counts        `json:"counts"`
}

type ExportRequest struct {
	SiteID string `json:"siteId"`
	Path   string `json:"path"`
}

type ExportResponse struct {
	Path     string `json:"path"`
	Pages    int    `json:"pages"`
	Entities int    `json:"entities"`
}

type SaveMappingRequest struct {
	Mapping Mapping `json:"mapping"`
}

type SaveMappingResponse struct {
	Mapping Mapping `json:"mapping"`
}

type ListMappingsRequest struct {
	SiteID string `json:"siteId"`
}

type ListMappingsResponse struct {
	Mappings []Mapping `json:"mappings"`
}

type DeleteMappingRequest struct {
	ID string `json:"id"`
}

type DeleteMappingResponse struct{}
