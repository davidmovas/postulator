package app

var (
	Version   = "2.0.0-dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

type BuildInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"buildDate"`
}

func Build() BuildInfo {
	return BuildInfo{Version: Version, Commit: Commit, BuildDate: BuildDate}
}
