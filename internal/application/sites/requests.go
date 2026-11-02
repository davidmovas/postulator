package sites

import (
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/dto"
)

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
	ID            string         `json:"id" description:"Site id"`
	Name          *string        `json:"name,omitempty" description:"New name"`
	BaseURL       *string        `json:"baseUrl,omitempty" description:"New site URL"`
	Username      *string        `json:"username,omitempty" description:"New administrator login"`
	Password      *string        `json:"password,omitempty" description:"New application password"`
	AllowInsecure *bool          `json:"allowInsecure,omitempty" description:"Accept an untrusted certificate, local sites only"`
	Status        *string        `json:"status,omitempty" enum:"active,paused,error" description:"New status"`
	Defaults      *Defaults      `json:"defaults,omitempty" description:"New fallback template and policy"`
	Settings      *site.Settings `json:"settings,omitempty" description:"New per-site settings; the observation's own bookkeeping is kept"`
}

type UpdateResponse struct {
	Site Site `json:"site"`
}

type DeleteRequest struct {
	ID string `json:"id" description:"Site id"`
}

type DeleteResponse struct{}

type ClearDataRequest struct {
	ID      string `json:"id" description:"Site id"`
	Model   bool   `json:"model" description:"Remove the pages and everything a sync read, the findings, the runs and the chats"`
	Graph   bool   `json:"graph" description:"Remove the entities and their edges; the next sync builds them again"`
	History bool   `json:"history" description:"Remove the runs and the chats"`
}

type ClearDataResponse struct {
	Held DataHeld `json:"held"`
}

type DataCountsRequest struct {
	ID string `json:"id" description:"Site id"`
}

type DataCountsResponse struct {
	Held DataHeld `json:"held"`
	Busy []string `json:"busy"`
}

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
