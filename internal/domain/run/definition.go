package run

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"slices"
	"strconv"
	"time"

	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	MaxRetryAttempts = 10
	CheckpointAccept = "accept"
)

type RetryPolicy struct {
	Backoff func(attempt int) time.Duration
	Max     int
}

type StepContext struct {
	Run       Run
	Item      Item
	Page      pagemap.Page
	Spec      template.TemplateSpec
	Params    map[string]any
	Artifacts map[ArtifactKind]Artifact
	Check     Checkpoint
}

func (c *StepContext) Artifact(kind ArtifactKind) (Artifact, error) {
	artifact, ok := c.Artifacts[kind]
	if !ok {
		return Artifact{}, errors.New(errors.Invalid, "the step ran without the artifact it requires").
			WithDetail("kind", string(kind))
	}
	if artifact.Purged {
		return Artifact{}, errors.New(errors.NotFound, "the artifact the step requires was purged").
			WithDetail("kind", string(kind))
	}
	return artifact, nil
}

func (c *StepContext) Accepted() bool {
	step, found, err := Get[string](c.Check, CheckpointAccept)
	return err == nil && found && step == c.Item.CurrentStep
}

func (c *StepContext) Param(name string) (any, bool) {
	value, ok := c.Params[name]
	return value, ok
}

func (c *StepContext) BoolParam(name string) bool {
	value, ok := c.Params[name].(bool)
	return ok && value
}

type Result struct {
	WakeAt     time.Time
	Checkpoint Checkpoint
	Artifacts  []Artifact
	Next       Transition
	Reason     PauseReason
	Message    string
	Notice     string
	Tokens     int
	USD        float64
}

type Price struct {
	Calls        func(spec template.TemplateSpec, params map[string]any) int
	Ref          *llm.ModelRef
	OutputTokens int
	Unpriced     bool
}

type Target struct {
	Page pagemap.Page
	Spec template.TemplateSpec
}

type Preflight func(ctx context.Context, record Run, targets map[string]Target) ([]EstimateFinding, error)

type StepDef struct {
	Run       func(ctx context.Context, sc *StepContext) (Result, error)
	Preflight Preflight
	Name      string
	Role      llm.Role
	Requires  []ArtifactKind
	Produces  []ArtifactKind
	Retry     RetryPolicy
	Price     Price
	Timeout   time.Duration
}

type Definition struct {
	Kind     Kind
	Steps    []StepDef
	Deadline time.Duration
}

func (d Definition) StepByName(name string) (index int, step StepDef, found bool) {
	for i := range d.Steps {
		if d.Steps[i].Name == name {
			return i, d.Steps[i], true
		}
	}
	return 0, StepDef{}, false
}

type Registry struct {
	byName map[string]StepDef
	order  []string
}

func NewRegistry() *Registry {
	return &Registry{byName: make(map[string]StepDef)}
}

func (r *Registry) Register(def StepDef) error {
	switch {
	case def.Name == "":
		return invalid("a step must be registered under a name", "name")
	case def.Run == nil:
		return invalid("step "+def.Name+" has nothing to run", "run")
	case def.Retry.Max < 0 || def.Retry.Max > MaxRetryAttempts:
		return invalid("step "+def.Name+" declares a retry ceiling outside 0.."+strconv.Itoa(MaxRetryAttempts), "retry.max")
	case def.Timeout < 0:
		return invalid("step "+def.Name+" declares a negative timeout", "timeout")
	case def.Role != "" && !def.Role.Valid():
		return invalid("step "+def.Name+" names the unknown model role "+string(def.Role), "role")
	}
	if _, exists := r.byName[def.Name]; exists {
		return errors.New(errors.Conflict, "step "+def.Name+" is already registered").WithDetail("name", def.Name)
	}
	for _, kind := range slices.Concat(def.Requires, def.Produces) {
		if !kind.Valid() {
			return invalid("step "+def.Name+" names the unknown artifact kind "+string(kind), "artifactKind")
		}
	}

	r.byName[def.Name] = def
	r.order = append(r.order, def.Name)
	return nil
}

func (r *Registry) Lookup(name string) (StepDef, bool) {
	def, ok := r.byName[name]
	return def, ok
}

func (r *Registry) Names() []string {
	return slices.Clone(r.order)
}

func Enabled(recipe []template.StepSpec) []template.StepSpec {
	out := make([]template.StepSpec, 0, len(recipe))
	for _, spec := range recipe {
		if spec.Enabled {
			out = append(out, spec)
		}
	}
	return out
}

type StepCatalog interface {
	Lookup(name string) (StepDef, bool)
	Names() []string
}

func ValidateRecipe(registry StepCatalog, recipe []template.StepSpec) error {
	enabled := Enabled(recipe)
	if len(enabled) == 0 {
		return invalid("the recipe enables no step", "recipe")
	}

	seen := make(map[string]struct{}, len(enabled))
	produced := make(map[ArtifactKind]struct{})
	for _, spec := range enabled {
		def, known := registry.Lookup(spec.Name)
		if !known {
			return errors.New(errors.NotFound, "the recipe names the unknown step "+spec.Name).
				WithDetail("step", spec.Name).WithDetail("known", registry.Names())
		}
		if _, duplicate := seen[spec.Name]; duplicate {
			return errors.New(errors.Conflict, "the recipe enables step "+spec.Name+" twice").
				WithDetail("step", spec.Name)
		}
		seen[spec.Name] = struct{}{}

		for _, required := range def.Requires {
			if _, ok := produced[required]; !ok {
				return invalid("step "+spec.Name+" requires "+string(required)+", which no earlier step produces", "recipe")
			}
		}
		for _, kind := range def.Produces {
			produced[kind] = struct{}{}
		}
	}
	return nil
}

func Plan(registry *Registry, kind Kind, recipe []template.StepSpec, deadline time.Duration) (Definition, error) {
	if err := ValidateRecipe(registry, recipe); err != nil {
		return Definition{}, err
	}
	if deadline <= 0 {
		return Definition{}, invalid("a definition needs a positive deadline", "deadline")
	}

	enabled := Enabled(recipe)
	steps := make([]StepDef, 0, len(enabled))
	for _, spec := range enabled {
		def, _ := registry.Lookup(spec.Name)
		steps = append(steps, def)
	}
	return Definition{Kind: kind, Steps: steps, Deadline: deadline}, nil
}

func ParamsFor(recipe []template.StepSpec, step string) map[string]any {
	for _, spec := range recipe {
		if spec.Name == step {
			return spec.Params
		}
	}
	return nil
}

func InputHash(step string, params map[string]any, required []Artifact, templateVersion int,
	check Checkpoint) (string, error) {
	encodedParams, err := json.Marshal(params)
	if err != nil {
		return "", errors.Wrap(err, errors.Internal, "encode the step parameters for the input hash")
	}
	encodedCheck, err := check.Encode()
	if err != nil {
		return "", err
	}

	hashes := make([]string, 0, len(required))
	for i := range required {
		hashes = append(hashes, string(required[i].Kind)+":"+required[i].Hash)
	}
	slices.Sort(hashes)

	digest := sha256.New()
	prefix := []string{step, string(encodedParams), strconv.Itoa(templateVersion), encodedCheck}
	for _, part := range append(prefix, hashes...) {
		digest.Write([]byte(part))
		digest.Write([]byte{0})
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}
