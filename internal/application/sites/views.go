package sites

import (
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/dto"
)

type Plugin struct {
	Installed    bool     `json:"installed"`
	Version      string   `json:"version"`
	Capabilities []string `json:"capabilities"`
	SEOPlugin    string   `json:"seoPlugin"`
}

type Defaults struct {
	TemplateID    *string                 `json:"templateId"`
	LinkPolicyID  *string                 `json:"linkPolicyId"`
	ModelProfiles map[string]llm.ModelRef `json:"modelProfiles"`
}

type Site struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	BaseURL       string   `json:"baseUrl"`
	Username      string   `json:"username"`
	Status        string   `json:"status"`
	AllowInsecure bool     `json:"allowInsecure"`
	Plugin        Plugin   `json:"plugin"`
	Defaults      Defaults `json:"defaults"`
	CreatedAt     dto.Time `json:"createdAt"`
	UpdatedAt     dto.Time `json:"updatedAt"`
}

type Reachability struct {
	Reach            string `json:"reach"`
	Message          string `json:"message"`
	SiteName         string `json:"siteName"`
	HomeURL          string `json:"homeUrl"`
	SuggestedBaseURL string `json:"suggestedBaseUrl"`
	HasPlugin        bool   `json:"hasPlugin"`
	HasWoo           bool   `json:"hasWoo"`
}

func reachabilityView(r site.Reachability) Reachability {
	return Reachability{
		Reach:            string(r.Reach),
		Message:          r.Message,
		SiteName:         r.SiteName,
		HomeURL:          r.HomeURL,
		SuggestedBaseURL: r.SuggestedBaseURL,
		HasPlugin:        r.HasPlugin,
		HasWoo:           r.HasWoo,
	}
}

func view(s site.Site) Site {
	capabilities := s.Plugin.Capabilities
	if capabilities == nil {
		capabilities = []string{}
	}
	profiles := make(map[string]llm.ModelRef, len(s.Defaults.ModelProfiles))
	for role, ref := range s.Defaults.ModelProfiles {
		profiles[string(role)] = ref
	}
	return Site{
		ID:            s.ID,
		Name:          s.Name,
		BaseURL:       s.BaseURL,
		Username:      s.Username,
		Status:        string(s.Status),
		AllowInsecure: s.AllowInsecure,
		Plugin:        Plugin{Installed: s.Plugin.Installed, Version: s.Plugin.Version, Capabilities: capabilities, SEOPlugin: s.Plugin.SEOPlugin},
		Defaults:      Defaults{TemplateID: s.Defaults.TemplateID, LinkPolicyID: s.Defaults.LinkPolicyID, ModelProfiles: profiles},
		CreatedAt:     dto.NewTime(s.CreatedAt),
		UpdatedAt:     dto.NewTime(s.UpdatedAt),
	}
}

func profilesOf(profiles map[string]llm.ModelRef) map[llm.Role]llm.ModelRef {
	out := make(map[llm.Role]llm.ModelRef, len(profiles))
	for role, ref := range profiles {
		out[llm.Role(role)] = ref
	}
	return out
}
