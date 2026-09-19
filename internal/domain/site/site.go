package site

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const redactedPassword = "***"

type Status string

const (
	StatusActive Status = "active"
	StatusPaused Status = "paused"
	StatusError  Status = "error"
)

func (s Status) Valid() bool {
	switch s {
	case StatusActive, StatusPaused, StatusError:
		return true
	default:
		return false
	}
}

type PluginState struct {
	Installed    bool
	Version      string
	Capabilities []string
	SEOPlugin    string
}

type Defaults struct {
	TemplateID    *string
	LinkPolicyID  *string
	ModelProfiles map[llm.Role]llm.ModelRef
}

type Site struct {
	ID            string
	Name          string
	BaseURL       string
	Username      string
	SecretRef     string
	Status        Status
	AllowInsecure bool
	Plugin        PluginState
	Defaults      Defaults
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func SecretRef(siteID string) string {
	return "site:" + siteID + ":wp_password"
}

type Reach string

const (
	ReachOK              Reach = "ok"
	ReachUpgradeRequired Reach = "upgradeRequired"
	ReachUnauthorized    Reach = "unauthorized"
	ReachUnreachable     Reach = "unreachable"
)

type Candidate struct {
	SiteID        string
	BaseURL       string
	Username      string
	Password      string
	AllowInsecure bool
}

func (c Candidate) String() string {
	return fmt.Sprintf("site.Candidate{SiteID:%q BaseURL:%q Username:%q Password:%s AllowInsecure:%t}",
		c.SiteID, c.BaseURL, c.Username, redactedPassword, c.AllowInsecure)
}

func (c Candidate) GoString() string {
	return c.String()
}

type Reachability struct {
	Reach            Reach
	Message          string
	SiteName         string
	HomeURL          string
	SuggestedBaseURL string
	HasPlugin        bool
	HasWoo           bool
}

func invalid(message, field string) *errors.Error {
	return errors.New(errors.Invalid, message).WithDetail("field", field)
}

func NormalizeBaseURL(raw string, allowInsecure bool) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", invalid("site base url must not be empty", "baseUrl")
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", invalid("site base url is not a valid url", "baseUrl").WithInternal(err)
	}

	scheme := strings.ToLower(parsed.Scheme)
	switch {
	case scheme == "https":
	case scheme == "http" && allowInsecure:
	case scheme == "http":
		return "", invalid("site base url must use https unless insecure urls are allowed", "baseUrl")
	default:
		return "", invalid("site base url must be an absolute http or https url", "baseUrl")
	}
	if parsed.Host == "" {
		return "", invalid("site base url must name a host", "baseUrl")
	}
	if parsed.User != nil {
		return "", invalid("site base url must not carry credentials", "baseUrl")
	}

	path := strings.TrimRight(parsed.EscapedPath(), "/")
	return scheme + "://" + strings.ToLower(parsed.Host) + path, nil
}

func (s Site) Validate() error {
	if s.ID == "" {
		return invalid("site id must not be empty", "id")
	}
	if strings.TrimSpace(s.Name) == "" {
		return invalid("site name must not be empty", "name")
	}
	if _, err := NormalizeBaseURL(s.BaseURL, s.AllowInsecure); err != nil {
		return err
	}
	if !s.Status.Valid() {
		return invalid("site status is not recognized", "status")
	}
	if s.SecretRef != SecretRef(s.ID) {
		return invalid("site secret reference does not belong to the site", "secretRef")
	}
	for role, ref := range s.Defaults.ModelProfiles {
		field := "defaults.modelProfiles." + string(role)
		if !role.Valid() {
			return invalid("model profile role is not recognized", field)
		}
		if !ref.Valid() {
			return invalid("model profile needs a provider and a model", field)
		}
	}
	return nil
}
