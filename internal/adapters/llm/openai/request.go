package openai

import (
	"crypto/sha256"
	"encoding/hex"

	port "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/domain/llm"
)

const (
	tierDefault = "default"
	tierFlex    = "flex"

	cacheExplicit   = "explicit"
	toolFunction    = "function"
	toolNamespace   = "namespace"
	toolSearch      = "tool_search"
	minOutputTokens = 16
	maxCacheKey     = 64

	effortWhenUnasked = llm.EffortLow
)

type wireRequest struct {
	Model         string             `json:"model"`
	Instructions  string             `json:"instructions,omitempty"`
	Input         []inputItem        `json:"input"`
	Store         bool               `json:"store"`
	MaxOutput     int                `json:"max_output_tokens,omitempty"`
	Reasoning     *wireReasoning     `json:"reasoning,omitempty"`
	Text          *wireText          `json:"text,omitempty"`
	Tools         []wireTool         `json:"tools,omitempty"`
	ServiceTier   string             `json:"service_tier"`
	CacheKey      string             `json:"prompt_cache_key,omitempty"`
	CacheOptions  *wireCacheOptions  `json:"prompt_cache_options,omitempty"`
	Stream        bool               `json:"stream,omitempty"`
	StreamOptions *wireStreamOptions `json:"stream_options,omitempty"`
}

type wireReasoning struct {
	Effort string `json:"effort"`
}

type wireText struct {
	Format wireFormat `json:"format"`
}

type wireFormat struct {
	Type   string         `json:"type"`
	Name   string         `json:"name"`
	Schema map[string]any `json:"schema"`
	Strict bool           `json:"strict"`
}

type wireTool struct {
	Type         string         `json:"type"`
	Name         string         `json:"name,omitempty"`
	Description  string         `json:"description,omitempty"`
	Parameters   map[string]any `json:"parameters,omitempty"`
	Strict       *bool          `json:"strict,omitempty"`
	DeferLoading bool           `json:"defer_loading,omitempty"`
	Tools        []wireTool     `json:"tools,omitempty"`
}

type wireCacheOptions struct {
	Mode string `json:"mode"`
}

type wireStreamOptions struct {
	IncludeObfuscation bool `json:"include_obfuscation"`
}

type catalogRow struct {
	info  llm.ModelInfo
	known bool
}

func listed(info llm.ModelInfo) catalogRow {
	return catalogRow{info: info, known: true}
}

func requestOf(req port.Request, model catalogRow, stream bool) (wireRequest, error) {
	body := wireRequest{
		Model:        req.Ref.Model,
		Instructions: req.System,
		Input:        itemsOf(req.Messages),
		ServiceTier:  tierFor(req.Tier, model),
		Tools:        toolsOf(req.Tools),
		CacheKey:     cacheKeyOf(req.CacheKey),
	}

	effort, reasons := effortFor(req.Effort, model)
	if reasons {
		body.Reasoning = &wireReasoning{Effort: string(effort)}
	}
	body.MaxOutput = ceilingOf(req.MaxTokens, effort, reasons, model)

	if req.Schema != nil {
		format, err := formatOf(req.Schema, req.Meta.Step)
		if err != nil {
			return wireRequest{}, err
		}
		body.Text = &wireText{Format: format}
	}
	if len(req.Tools) == 0 && req.CacheKey == "" {
		body.CacheOptions = &wireCacheOptions{Mode: cacheExplicit}
	}
	if stream {
		body.Stream = true
		body.StreamOptions = &wireStreamOptions{IncludeObfuscation: false}
	}
	return body, nil
}

func effortFor(asked llm.ReasoningEffort, model catalogRow) (llm.ReasoningEffort, bool) {
	switch {
	case model.known && !model.info.Reasoning:
		return "", false
	case asked.Valid():
		return asked, true
	case model.known:
		return effortWhenUnasked, true
	default:
		return "", false
	}
}

func ceilingOf(asked int, effort llm.ReasoningEffort, reasons bool, model catalogRow) int {
	if asked <= 0 {
		return 0
	}

	ceiling := asked
	if reasons {
		ceiling += llm.Allowance(effort)
	}
	if model.known && model.info.MaxOutputTokens > 0 {
		ceiling = min(ceiling, model.info.MaxOutputTokens)
	}
	return max(ceiling, minOutputTokens)
}

func tierFor(asked llm.ServiceTier, model catalogRow) string {
	if asked == llm.TierFlex && model.known && model.info.OffersFlex() {
		return tierFlex
	}
	return tierDefault
}

func cacheKeyOf(key string) string {
	if len(key) <= maxCacheKey {
		return key
	}
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

func toolsOf(tools []port.Tool) []wireTool {
	if len(tools) == 0 {
		return nil
	}

	wired := make([]wireTool, 0, len(tools)+1)
	namespaces := map[string]int{}
	for _, tool := range tools {
		function := functionOf(tool)
		if tool.Deferred == nil {
			wired = append(wired, function)
			continue
		}

		at, opened := namespaces[tool.Deferred.Name]
		if !opened {
			at = len(wired)
			namespaces[tool.Deferred.Name] = at
			wired = append(wired, wireTool{Type: toolNamespace, Name: tool.Deferred.Name, Description: tool.Deferred.Description})
		}
		function.DeferLoading = true
		wired[at].Tools = append(wired[at].Tools, function)
	}
	if len(namespaces) > 0 {
		wired = append(wired, wireTool{Type: toolSearch})
	}
	return wired
}

func functionOf(tool port.Tool) wireTool {
	strict := false
	return wireTool{
		Type:        toolFunction,
		Name:        tool.Name,
		Description: tool.Description,
		Parameters:  parametersOf(tool.Schema),
		Strict:      &strict,
	}
}
