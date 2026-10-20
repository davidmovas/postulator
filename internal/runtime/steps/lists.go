package steps

import (
	"regexp"
	"slices"
	"strconv"

	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/field"
)

const (
	reasonNoCandidates = "the site model offers no other published product beside it"
	reasonNoneChosen   = "the model chose no product among the candidates"
)

var (
	listFields = []field.Key{field.Upsells, field.CrossSells}
	listIDs    = regexp.MustCompile(`\d+`)
)

func listField(key field.Key) bool {
	return slices.Contains(listFields, key)
}

func listAnswer(answer string, candidates []contextpack.Candidate) string {
	offered := make(map[int64]struct{}, len(candidates))
	for i := range candidates {
		offered[candidates[i].WPID] = struct{}{}
	}
	chosen := make([]int64, 0)
	for _, match := range listIDs.FindAllString(answer, -1) {
		id, err := strconv.ParseInt(match, 10, 64)
		if _, known := offered[id]; err == nil && known {
			chosen = append(chosen, id)
		}
	}
	return field.IDList(chosen)
}

func withCandidates(candidates []contextpack.Candidate, asked []askedScope, skipped []FieldSkip) ([]askedScope, []FieldSkip) {
	if len(candidates) > 0 {
		return asked, skipped
	}
	kept := make([]askedScope, 0, len(asked))
	for i := range asked {
		if listField(asked[i].Key) {
			skipped = append(skipped, FieldSkip{Key: string(asked[i].Key), Reason: reasonNoCandidates})
			continue
		}
		kept = append(kept, asked[i])
	}
	return kept, skipped
}

func candidateRules(candidates []contextpack.Candidate) []string {
	rules := make([]string, 0, 1+len(candidates))
	rules = append(rules, "Answer the ids of the products it names, separated by commas, chosen only from these:")
	for i := range candidates {
		rules = append(rules, strconv.FormatInt(candidates[i].WPID, 10)+": "+candidates[i].Name+" ("+candidates[i].Path+")")
	}
	return rules
}
