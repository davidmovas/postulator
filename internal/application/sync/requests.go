package sync

const Filename = "postulator-companion.zip"

type SyncSiteRequest struct {
	SiteID string `json:"siteId"`
}

type SyncSiteResponse struct {
	RunID string `json:"runId"`
}

type CheckPluginRequest struct {
	SiteID string `json:"siteId"`
}

type CheckPluginResponse struct {
	Plugin Plugin `json:"plugin"`
}

type PluginPackageRequest struct{}

type PluginPackageResponse struct {
	Filename string `json:"filename"`
	Bytes    []byte `json:"bytes"`
}
