package sites

import "github.com/davidmovas/postulator/internal/kernel/dto"

type CreateRequest struct {
	Name          string `json:"name" description:"What to call the site inside Postulator"`
	BaseURL       string `json:"baseUrl" description:"The address of the WordPress site, for example https://example.com"`
	Username      string `json:"username" description:"A WordPress administrator login"`
	Password      string `json:"password" description:"That administrator's application password"`
	AllowInsecure bool   `json:"allowInsecure" description:"Accept a certificate the machine does not trust, which only a local site should need"`
}

type CreateResponse struct {
	Site Site `json:"site"`
}

type UpdateRequest struct {
	ID            string    `json:"id" description:"The id of the site, exactly as sites_list returned it"`
	Name          *string   `json:"name,omitempty" description:"The new name, left out to keep the current one"`
	BaseURL       *string   `json:"baseUrl,omitempty" description:"The new address, left out to keep the current one"`
	Username      *string   `json:"username,omitempty" description:"The new administrator login, left out to keep the current one"`
	Password      *string   `json:"password,omitempty" description:"The new application password, left out to keep the current one"`
	AllowInsecure *bool     `json:"allowInsecure,omitempty" description:"The new certificate setting, left out to keep the current one"`
	Status        *string   `json:"status,omitempty" enum:"active,paused,error" description:"The new status, left out to keep the current one"`
	Defaults      *Defaults `json:"defaults,omitempty" description:"The template and policy the site falls back to, left out to keep the current ones"`
}

type UpdateResponse struct {
	Site Site `json:"site"`
}

type DeleteRequest struct {
	ID string `json:"id" description:"The id of the site to remove, exactly as sites_list returned it"`
}

type DeleteResponse struct{}

type GetRequest struct {
	ID string `json:"id" description:"The id of the site, exactly as sites_list returned it"`
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
	Status string `json:"status,omitempty" enum:"active,paused,error" description:"Keep only sites in this state"`
}
