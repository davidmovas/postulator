package content

import (
	"github.com/davidmovas/postulator/internal/application/llm"
	contentdomain "github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/template"
)

type AssessRequest struct {
	SiteID     string
	Page       pagemap.Page
	Entity     graph.Entity
	Spec       template.TemplateSpec
	Body       string
	Snippet    Snippet
	Targets    []contentdomain.LinkTarget
	Call       llm.CallMeta
	HasSnippet bool
}

type AssessResponse struct {
	Report JudgeReport
	Tokens int
}

type JudgeRequest struct {
	PageID string `json:"pageId"`
}

type JudgeResponse struct {
	PageID string      `json:"pageId"`
	Path   string      `json:"path"`
	Report JudgeReport `json:"report"`
	Tokens int         `json:"tokens"`
}
