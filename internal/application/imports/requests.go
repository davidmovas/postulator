package imports

type InspectRequest struct {
	SiteID string   `json:"siteId"`
	Path   string   `json:"path"`
	Sheets []string `json:"sheets,omitempty"`
}

type InspectResponse struct {
	Headers  []string   `json:"headers"`
	Sample   [][]string `json:"sample"`
	Rows     int        `json:"rows"`
	Sheets   []Sheet    `json:"sheets"`
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
	Path   string `json:"path" description:"The absolute path on this machine to write the file to"`
	Format string `json:"format,omitempty" enum:"xlsx,csv" description:"Which format to write; leave it out to take it from the file extension"`
}

type ExportResponse struct {
	Path     string `json:"path"`
	Format   string `json:"format"`
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
	ID string `json:"id" description:"The id of the saved mapping to remove, exactly as imports_list_mappings returned it"`
}

type DeleteMappingResponse struct{}
