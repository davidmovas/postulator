package sites

import "github.com/davidmovas/postulator/internal/kernel/dto"

type CreateRequest struct {
	Name          string `json:"name"`
	BaseURL       string `json:"baseUrl"`
	Username      string `json:"username"`
	Password      string `json:"password"`
	AllowInsecure bool   `json:"allowInsecure"`
}

type CreateResponse struct {
	Site Site `json:"site"`
}

type UpdateRequest struct {
	ID            string    `json:"id"`
	Name          *string   `json:"name,omitempty"`
	BaseURL       *string   `json:"baseUrl,omitempty"`
	Username      *string   `json:"username,omitempty"`
	Password      *string   `json:"password,omitempty"`
	AllowInsecure *bool     `json:"allowInsecure,omitempty"`
	Status        *string   `json:"status,omitempty"`
	Defaults      *Defaults `json:"defaults,omitempty"`
}

type UpdateResponse struct {
	Site Site `json:"site"`
}

type DeleteRequest struct {
	ID string `json:"id"`
}

type DeleteResponse struct{}

type GetRequest struct {
	ID string `json:"id"`
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
	Status string `json:"status,omitempty"`
}
