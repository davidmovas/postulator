package imports

type InspectRequest struct {
	SiteID   string   `json:"siteId"`
	Path     string   `json:"path" description:"Absolute path of the .xlsx or .csv file"`
	Sheets   []string `json:"sheets,omitempty" description:"Sheets to sample; default the first. Every sheet is listed"`
	NoHeader bool     `json:"noHeader,omitempty" description:"No header row: columns go by letter"`
}

type InspectResponse struct {
	Headers  []string   `json:"headers"`
	Sample   [][]string `json:"sample"`
	Rows     int        `json:"rows"`
	Sheets   []Sheet    `json:"sheets"`
	Detected Mapping    `json:"detected"`
	Saved    []Mapping  `json:"saved"`
}

type SheetMapping struct {
	Sheet   string  `json:"sheet"`
	Mapping Mapping `json:"mapping"`
}

type PreviewRequest struct {
	SiteID  string         `json:"siteId"`
	Path    string         `json:"path"`
	Mapping Mapping        `json:"mapping"`
	Sheets  []SheetMapping `json:"sheets,omitempty"`
}

type PreviewResponse struct {
	Report PreviewReport `json:"report"`
}

type ApplyOptions struct {
	SaveMappingAs string `json:"saveMappingAs,omitempty"`
}

type ApplyRequest struct {
	SiteID  string         `json:"siteId"`
	Path    string         `json:"path"`
	Mapping Mapping        `json:"mapping"`
	Sheets  []SheetMapping `json:"sheets,omitempty"`
	Options ApplyOptions   `json:"options"`
}

func (r ApplyRequest) preview() PreviewRequest {
	return PreviewRequest{SiteID: r.SiteID, Path: r.Path, Mapping: r.Mapping, Sheets: r.Sheets}
}

type ApplyResponse struct {
	Report PreviewReport `json:"report"`
	Counts Counts        `json:"counts"`
}

type ExportRequest struct {
	SiteID string `json:"siteId"`
	Path   string `json:"path" description:"Absolute path to write"`
	Format string `json:"format,omitempty" enum:"xlsx,csv" description:"Default from the file extension"`
}

type ExportResponse struct {
	Path     string    `json:"path"`
	Format   string    `json:"format"`
	Pages    int       `json:"pages"`
	Entities int       `json:"entities"`
	Warnings []Finding `json:"warnings"`
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
	ID string `json:"id" description:"Saved mapping id"`
}

type DeleteMappingResponse struct{}
