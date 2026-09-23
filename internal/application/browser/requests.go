package browser

type OpenRequest struct {
	URL string `json:"url"`
}

type OpenResponse struct{}

type LocateRequest struct{}

type LocateResponse struct {
	Path      string `json:"path"`
	Source    string `json:"source"`
	Installed bool   `json:"installed"`
}
