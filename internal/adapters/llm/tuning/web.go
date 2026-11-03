package tuning

import (
	"strconv"
	"strings"

	port "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/settings"
)

const (
	defaultSearchCalls = 5
	mostSearchCalls    = 20
)

var (
	searchDepths = []string{string(llm.DepthLow), string(llm.DepthMedium), string(llm.DepthHigh)}

	searchDepthSetting  = settings.Enum("llm.webSearch.depth", string(llm.DepthMedium), searchDepths)
	searchCallsSetting  = settings.Int("llm.webSearch.maxCalls", defaultSearchCalls, settings.IntRange(1, mostSearchCalls))
	allowedSitesSetting = settings.String("llm.webSearch.allowedDomains", "", domainsValidator())
	blockedSitesSetting = settings.String("llm.webSearch.blockedDomains", "", domainsValidator())
)

func domainsValidator() settings.Validator[string] {
	return settings.Check(checkDomains)
}

func (p *Policy) WebSearch() port.WebSearch {
	return port.WebSearch{
		Depth:    llm.SearchDepth(searchDepthSetting.Get(p.values)),
		MaxCalls: searchCallsSetting.Get(p.values),
		Allowed:  domainList(allowedSitesSetting.Get(p.values)),
		Blocked:  domainList(blockedSitesSetting.Get(p.values)),
	}
}

func fieldsOf(raw string) []string {
	return strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || r == ' ' || r == '\n' || r == '\r' || r == '\t'
	})
}

func checkDomains(raw string) error {
	for _, field := range fieldsOf(raw) {
		if _, err := hostOf(field); err != nil {
			return err
		}
	}
	if len(domainList(raw)) > port.MaxSearchDomains {
		return errors.New(errors.Invalid, "a web search takes at most "+strconv.Itoa(port.MaxSearchDomains)+" domains")
	}
	return nil
}

func domainList(raw string) []string {
	var out []string
	seen := map[string]bool{}
	for _, field := range fieldsOf(raw) {
		host, err := hostOf(field)
		if err != nil || seen[host] {
			continue
		}
		seen[host] = true
		out = append(out, host)
	}
	return out
}

func hostOf(field string) (string, error) {
	host := strings.ToLower(strings.TrimSpace(field))
	host = strings.TrimPrefix(strings.TrimPrefix(host, "https://"), "http://")
	if cut := strings.IndexAny(host, "/?#"); cut >= 0 {
		host = host[:cut]
	}
	if !domainLike(host) {
		return "", errors.New(errors.Invalid, "“"+field+"” is not a domain such as example.com").WithDetail("value", field)
	}
	return host, nil
}

func domainLike(host string) bool {
	labels := strings.Split(host, ".")
	if len(labels) < 2 {
		return false
	}
	for _, label := range labels {
		if label == "" || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return false
		}
		for _, r := range label {
			if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
				return false
			}
		}
	}
	return true
}
