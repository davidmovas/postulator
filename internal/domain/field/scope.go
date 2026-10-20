package field

import (
	"bytes"
	"encoding/json"
	"slices"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	ReasonDenied  = "field_denied"
	ReasonUnknown = "field_unknown"
)

type Provider string

const (
	ProviderNone   Provider = ""
	ProviderPlugin Provider = "plugin"
	ProviderStore  Provider = "store"
)

func (p Provider) Valid() bool {
	return p == ProviderPlugin || p == ProviderStore
}

type AnchorStrategy string

const (
	AnchorPreferUser AnchorStrategy = "prefer_user"
	AnchorRotate     AnchorStrategy = "rotate"
)

func (s AnchorStrategy) Valid() bool {
	return s == AnchorPreferUser || s == AnchorRotate
}

type LinkRules struct {
	UpDepth                    int     `json:"upDepth" minimum:"0" description:"Levels up to link; 1 is the parent"`
	DownLinks                  bool    `json:"downLinks" description:"Link to children"`
	SiblingMinWeight           float64 `json:"siblingMinWeight" minimum:"0" maximum:"1" description:"Related weight a sibling link needs"`
	MaxLinks                   int     `json:"maxLinks" minimum:"0" description:"Max links; 0 for no cap"`
	MaxPerTarget               int     `json:"maxPerTarget" minimum:"0" description:"Max links per target"`
	ParentLinkWithinParagraphs int     `json:"parentLinkWithinParagraphs" minimum:"0" description:"Parent link within N first paragraphs"`
	ChildrenSection            bool    `json:"childrenSection" description:"End with a list of children"`
}

type LinkPolicy struct {
	Links          bool           `json:"links" description:"Carries contextual links"`
	Rules          LinkRules      `json:"rules" description:"Links owed and allowed"`
	ForbidExternal bool           `json:"forbidExternal" description:"Refuse external links"`
	ForbidSelf     bool           `json:"forbidSelf" description:"Refuse self links"`
	AnchorStrategy AnchorStrategy `json:"anchorStrategy" enum:"prefer_user,rotate" description:"prefer_user takes human anchors first"`
}

func NoLinks() LinkPolicy {
	return LinkPolicy{AnchorStrategy: AnchorPreferUser}
}

func (p LinkPolicy) Validate() error {
	switch {
	case p.Rules.UpDepth < 0:
		return invalid("up depth must not be negative", "links.rules.upDepth")
	case p.Rules.SiblingMinWeight < 0 || p.Rules.SiblingMinWeight > 1:
		return invalid("sibling minimum weight must be between 0 and 1", "links.rules.siblingMinWeight")
	case p.Rules.MaxLinks < 0:
		return invalid("maximum links must not be negative", "links.rules.maxLinks")
	case p.Rules.MaxPerTarget < 0:
		return invalid("maximum links per target must not be negative", "links.rules.maxPerTarget")
	case p.Rules.ParentLinkWithinParagraphs < 0:
		return invalid("parent link paragraph window must not be negative", "links.rules.parentLinkWithinParagraphs")
	case !p.AnchorStrategy.Valid():
		return invalid("anchor strategy is not recognized", "links.anchorStrategy")
	default:
		return nil
	}
}

func ParsePolicy(raw []byte) (LinkPolicy, error) {
	var stated struct {
		Links *bool `json:"links"`
	}
	if err := json.Unmarshal(raw, &stated); err != nil {
		return LinkPolicy{}, errors.Wrap(err, errors.Internal, "decode a link policy")
	}
	if stated.Links == nil {
		return LinkPolicy{}, errors.New(errors.Internal, "a stored link policy does not say whether it carries links")
	}
	policy := NoLinks()
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&policy); err != nil {
		return LinkPolicy{}, errors.Wrap(err, errors.Internal, "decode a link policy")
	}
	return policy, nil
}

type Scope struct {
	PageTypeID  string
	Key         Key
	Enabled     bool
	AIWrites    bool
	Instruction string
	MinLength   int
	MaxLength   int
	Rewrites    bool
	Links       LinkPolicy
	Label       string
	Provider    Provider
	UpdatedAt   time.Time
}

func (s Scope) Validate() error {
	if strings.TrimSpace(s.PageTypeID) == "" {
		return invalid("a scope belongs to a page type", "pageTypeId")
	}
	if err := CheckScope(s.Key); err != nil {
		return err
	}
	custom := s.Key.Kind() == Meta
	switch {
	case custom && strings.TrimSpace(s.Label) == "":
		return invalid("a custom field needs a label", "label")
	case custom && !s.Provider.Valid():
		return invalid("a custom field is read through the plugin or the store", "provider")
	case !custom && s.Provider != ProviderNone:
		return invalid("only a custom field names its provider", "provider")
	case s.MinLength < 0:
		return invalid("minimum length must not be negative", "minLength")
	case s.MaxLength < 0 || (s.MaxLength > 0 && s.MaxLength < s.MinLength):
		return invalid("maximum length must be at least the minimum", "maxLength")
	}
	return s.Links.Validate()
}

var deniedMetaPrefixes = []string{"_", "rank_math_"}

var deniedMetaKeys = []string{"total_sales"}

func CheckScope(key Key) error {
	spec, known := Lookup(key)
	switch {
	case known && spec.Denied:
		return refused(key, ReasonDenied, "the field is never written")
	case !known || !spec.Text && !scopedWithoutText(spec.Key):
		return refused(key, ReasonUnknown, "no scope may name the field")
	case spec.Key != Meta && key.Instance() != "":
		return refused(key, ReasonUnknown, "a scope names a field for every instance")
	case spec.Key == Meta:
		return checkMetaKey(key)
	default:
		return nil
	}
}

func scopedWithoutText(key Key) bool {
	return key == Upsells || key == CrossSells
}

func checkMetaKey(key Key) error {
	name := key.Instance()
	if name == "" || strings.TrimSpace(name) != name || strings.ContainsAny(name, ", \t\r\n") {
		return refused(key, ReasonUnknown, "a custom field needs one meta key")
	}
	for _, prefix := range deniedMetaPrefixes {
		if strings.HasPrefix(name, prefix) {
			return refused(key, ReasonDenied, "the meta key is protected")
		}
	}
	if slices.Contains(deniedMetaKeys, name) {
		return refused(key, ReasonDenied, "the meta key is the store's")
	}
	return nil
}

func refused(key Key, reason, message string) error {
	return invalid(message, "field").WithDetail("reason", reason).WithDetail("key", string(key))
}

func invalid(message, field string) *errors.Error {
	return errors.New(errors.Invalid, message).WithDetail("field", field)
}
