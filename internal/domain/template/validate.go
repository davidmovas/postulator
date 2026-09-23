package template

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func invalid(message, field string) *errors.Error {
	return errors.New(errors.Invalid, message).WithDetail("field", field)
}

func Validate(spec TemplateSpec) error {
	if len(spec.Sections) == 0 {
		return invalid("template needs at least one section", "sections")
	}
	for i := range spec.Sections {
		field := "sections[" + strconv.Itoa(i) + "]"
		if strings.TrimSpace(spec.Sections[i].Heading) == "" {
			return invalid("section heading must not be empty", field+".heading")
		}
		if spec.Sections[i].TargetWords < 0 {
			return invalid("section target words must not be negative", field+".targetWords")
		}
	}
	if spec.Length.Min < 0 {
		return invalid("length minimum must not be negative", "length.min")
	}
	if spec.Length.Max < spec.Length.Min {
		return invalid("length maximum must not be below the minimum", "length.max")
	}
	if spec.KeywordRules.MaxDensity < 0 || spec.KeywordRules.MaxDensity > 1 {
		return invalid("keyword density must be between 0 and 1", "keywordRules.maxDensity")
	}
	if err := ValidateLinkRules(spec.LinkRules); err != nil {
		return err
	}
	if spec.MetaRules.DescriptionMax < 0 {
		return invalid("description maximum must not be negative", "metaRules.descriptionMax")
	}
	if spec.Images.Inline < 0 {
		return invalid("inline image count must not be negative", "images.inline")
	}
	if spec.Images.Source != "" && !spec.Images.Source.Valid() {
		return invalid("image source is not recognized", "images.source")
	}
	if (spec.Images.Featured || spec.Images.Inline > 0) && spec.Images.Source == "" {
		return invalid("image source is required when images are requested", "images.source")
	}
	for role, ref := range spec.ModelProfiles {
		field := "modelProfiles." + string(role)
		if !role.Valid() {
			return invalid("model profile role is not recognized", field)
		}
		if !ref.Valid() {
			return invalid("model profile needs a provider and a model", field)
		}
	}
	seen := make(map[string]struct{}, len(spec.Recipe))
	for i := range spec.Recipe {
		name := strings.TrimSpace(spec.Recipe[i].Name)
		field := "recipe[" + strconv.Itoa(i) + "].name"
		if name == "" {
			return invalid("recipe step name must not be empty", field)
		}
		if _, dup := seen[name]; dup {
			return invalid("recipe step name is repeated", field)
		}
		seen[name] = struct{}{}
	}
	return nil
}

func ValidateLinkRules(rules LinkRules) error {
	switch {
	case rules.UpDepth < 0:
		return invalid("up depth must not be negative", "linkRules.upDepth")
	case rules.SiblingMinWeight < 0 || rules.SiblingMinWeight > 1:
		return invalid("sibling minimum weight must be between 0 and 1", "linkRules.siblingMinWeight")
	case rules.MaxLinks < 0:
		return invalid("maximum links must not be negative", "linkRules.maxLinks")
	case rules.MaxPerTarget < 0:
		return invalid("maximum links per target must not be negative", "linkRules.maxPerTarget")
	case rules.ParentLinkWithinParagraphs < 0:
		return invalid("parent link paragraph window must not be negative", "linkRules.parentLinkWithinParagraphs")
	default:
		return nil
	}
}

func validateScope(scope Scope, siteID *string) error {
	if !scope.Valid() {
		return invalid("scope is not recognized", "scope")
	}
	if scope == ScopeGlobal && siteID != nil {
		return invalid("a global record carries no site id", "siteId")
	}
	if scope == ScopeSite && (siteID == nil || *siteID == "") {
		return invalid("a site record needs a site id", "siteId")
	}
	return nil
}

func (t Template) Validate() error {
	if t.ID == "" {
		return invalid("template id must not be empty", "id")
	}
	if err := validateScope(t.Scope, t.SiteID); err != nil {
		return err
	}
	if strings.TrimSpace(t.Name) == "" {
		return invalid("template name must not be empty", "name")
	}
	if strings.TrimSpace(t.PageKind) == "" {
		return invalid("template page kind must not be empty", "pageKind")
	}
	if t.Version < 1 {
		return invalid("template version starts at 1", "version")
	}
	return Validate(t.Spec)
}

func (p LinkPolicy) Validate() error {
	if p.ID == "" {
		return invalid("link policy id must not be empty", "id")
	}
	if err := validateScope(p.Scope, p.SiteID); err != nil {
		return err
	}
	if strings.TrimSpace(p.Name) == "" {
		return invalid("link policy name must not be empty", "name")
	}
	if err := ValidateLinkRules(p.Rules); err != nil {
		return err
	}
	if !p.AnchorStrategy.Valid() {
		return invalid("anchor strategy is not recognized", "anchorStrategy")
	}
	return nil
}

func (o Override) Validate() error {
	switch {
	case o.ID == "":
		return invalid("override id must not be empty", "id")
	case o.TemplateID == "":
		return invalid("override template id must not be empty", "templateId")
	case !o.Scope.Valid():
		return invalid("override scope is not recognized", "scope")
	case o.TargetID == "":
		return invalid("override target id must not be empty", "targetId")
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(o.Patch, &object); err != nil || object == nil {
		return invalid("override patch must be a json object", "patch")
	}
	return nil
}
