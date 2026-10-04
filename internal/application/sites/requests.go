package sites

import "github.com/davidmovas/postulator/internal/kernel/dto"

type CreateRequest struct {
	Name          string `json:"name" description:"Name inside Postulator"`
	BaseURL       string `json:"baseUrl" description:"Site URL, e.g. https://example.com"`
	Username      string `json:"username" description:"WordPress administrator login"`
	Password      string `json:"password" description:"Its application password"`
	AllowInsecure bool   `json:"allowInsecure" description:"Accept an untrusted certificate, local sites only"`
}

type CreateResponse struct {
	Site Site `json:"site"`
}

type UpdateRequest struct {
	ID            string    `json:"id" description:"Site id"`
	Name          *string   `json:"name,omitempty" description:"New name"`
	BaseURL       *string   `json:"baseUrl,omitempty" description:"New site URL"`
	Username      *string   `json:"username,omitempty" description:"New administrator login"`
	Password      *string   `json:"password,omitempty" description:"New application password"`
	AllowInsecure *bool     `json:"allowInsecure,omitempty" description:"Accept an untrusted certificate, local sites only"`
	Status        *string   `json:"status,omitempty" enum:"active,paused,error" description:"New status"`
	Defaults      *Defaults `json:"defaults,omitempty" description:"New fallback template and policy"`
}

type UpdateResponse struct {
	Site Site `json:"site"`
}

type DeleteRequest struct {
	ID string `json:"id" description:"Site id"`
}

type DeleteResponse struct{}

type GetRequest struct {
	ID string `json:"id" description:"Site id"`
}

type GetResponse struct {
	Site Site `json:"site"`
}

type TestConnectionRequest struct {
	SiteID        string `json:"siteId,omitempty"`
	BaseURL       string `json:"baseUrl,omitempty"`
	Username      string `json:"username,omitempty"`
	Password      string `json:"password,omitempty"`
	AllowInsecure *bool  `json:"allowInsecure,omitempty"`
}

type TestConnectionResponse struct {
	Reachability Reachability `json:"reachability"`
}

type ListRequest struct {
	dto.ListRequest
	Status string `json:"status,omitempty" enum:"active,paused,error" description:"Only this state"`
}
