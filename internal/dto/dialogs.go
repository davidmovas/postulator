package dto

// FileFilter represents a file filter for file dialogs
type FileFilter struct {
	DisplayName string `json:"displayName"` // e.g., "Text Files"
	Pattern     string `json:"pattern"`     // e.g., "*.txt;*.csv"
}
