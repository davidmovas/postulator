package pipeline

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/domain/topics"
)

type MetadataKey string

type Context struct {
	ctx       context.Context
	Job       *entities.Job
	State     *StateMachine
	StartTime time.Time
	Metadata  map[MetadataKey]any

	Validated   *ValidatedPhase
	Selection   *SelectionPhase
	Execution   *ExecutionPhase
	Generation  *GenerationPhase
	Publication *PublicationPhase
}

type ValidatedPhase struct {
	Site     *entities.Site
	Strategy topics.TopicStrategyHandler
}

type SelectionPhase struct {
	OriginalTopic  *entities.Topic
	VariationTopic *entities.Topic
	Category       *entities.Category
}

type ExecutionPhase struct {
	Execution *entities.Execution
	Prompt    *entities.Prompt
	Provider  *entities.Provider
}

type GenerationPhase struct {
	SystemPrompt     string
	UserPrompt       string
	GeneratedTitle   string
	GeneratedExcerpt string
	GeneratedContent string
	TokensUsed       int
	CostUSD          float64
	GenerationTimeMs int
}

type PublicationPhase struct {
	Article *entities.Article
}

func NewContext(job *entities.Job) *Context {
	return &Context{
		Job:       job,
		State:     NewStateMachine(StateInitialized),
		StartTime: time.Now(),
		Metadata:  make(map[MetadataKey]any),
	}
}

func (c *Context) Context() context.Context {
	return c.ctx
}

func (c *Context) WithContext(ctx context.Context) *Context {
	c.ctx = ctx
	return c
}

func (c *Context) SetMetadata(key MetadataKey, value any) {
	c.Metadata[key] = value
}

func (c *Context) GetMetadata(key MetadataKey) (any, bool) {
	val, ok := c.Metadata[key]
	return val, ok
}

func (c *Context) Duration() time.Duration {
	return time.Since(c.StartTime)
}

func (c *Context) InitValidatedPhase(site *entities.Site, strategy topics.TopicStrategyHandler) {
	c.Validated = &ValidatedPhase{
		Site:     site,
		Strategy: strategy,
	}
}

func (c *Context) InitSelectionPhase(original, variation *entities.Topic, category *entities.Category) {
	c.Selection = &SelectionPhase{
		OriginalTopic:  original,
		VariationTopic: variation,
		Category:       category,
	}
}

func (c *Context) InitExecutionPhase(exec *entities.Execution, prompt *entities.Prompt, provider *entities.Provider) {
	c.Execution = &ExecutionPhase{
		Execution: exec,
		Prompt:    prompt,
		Provider:  provider,
	}
}

func (c *Context) InitGenerationPhase() {
	c.Generation = &GenerationPhase{}
}

func (c *Context) InitPublicationPhase(article *entities.Article) {
	c.Publication = &PublicationPhase{
		Article: article,
	}
}

func (c *Context) HasValidated() bool {
	return c.Validated != nil
}

func (c *Context) HasSelection() bool {
	return c.Selection != nil
}

func (c *Context) HasExecution() bool {
	return c.Execution != nil
}

func (c *Context) HasGeneration() bool {
	return c.Generation != nil
}

func (c *Context) HasPublication() bool {
	return c.Publication != nil
}

func (c *Context) GetSite() *entities.Site {
	if c.Validated != nil {
		return c.Validated.Site
	}
	return nil
}

func (c *Context) GetStrategy() topics.TopicStrategyHandler {
	if c.Validated != nil {
		return c.Validated.Strategy
	}
	return nil
}

func (c *Context) GetTopic() *entities.Topic {
	if c.Selection != nil {
		return c.Selection.VariationTopic
	}
	return nil
}

func (c *Context) GetOriginalTopic() *entities.Topic {
	if c.Selection != nil {
		return c.Selection.OriginalTopic
	}
	return nil
}

func (c *Context) GetCategory() *entities.Category {
	if c.Selection != nil {
		return c.Selection.Category
	}
	return nil
}

func (c *Context) GetExecution() *entities.Execution {
	if c.Execution != nil {
		return c.Execution.Execution
	}
	return nil
}

func (c *Context) GetPrompt() *entities.Prompt {
	if c.Execution != nil {
		return c.Execution.Prompt
	}
	return nil
}

func (c *Context) GetProvider() *entities.Provider {
	if c.Execution != nil {
		return c.Execution.Provider
	}
	return nil
}

func (c *Context) GetArticle() *entities.Article {
	if c.Publication != nil {
		return c.Publication.Article
	}
	return nil
}

func (c *Context) Clone() *Context {
	clone := &Context{
		Job:       c.Job,
		State:     c.State,
		StartTime: c.StartTime,
		Metadata:  make(map[MetadataKey]any),
	}

	for k, v := range c.Metadata {
		clone.Metadata[k] = v
	}

	clone.Validated = c.Validated
	clone.Selection = c.Selection
	clone.Execution = c.Execution
	clone.Generation = c.Generation
	clone.Publication = c.Publication

	return clone
}

func GetTypedMetadata[T any](c *Context, key MetadataKey) (T, bool) {
	val, ok := c.GetMetadata(key)
	if !ok {
		var zero T
		return zero, false
	}
	typed, ok := val.(T)
	return typed, ok
}
