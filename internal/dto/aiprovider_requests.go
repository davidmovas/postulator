package dto

// AIProviderCreate is used to create a provider (includes API key)
type AIProviderCreate struct {
	Name     string `json:"name"`
	APIKey   string `json:"apiKey"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
	IsActive bool   `json:"isActive"`
}

// AIProviderUpdate is used to update a provider
// API key is optional on update; send empty to keep existing
// (Service layer may decide policy; here we just forward when provided)
type AIProviderUpdate struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	APIKey   string `json:"apiKey,omitempty"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
	IsActive bool   `json:"isActive"`
}
