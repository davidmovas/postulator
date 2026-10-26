package tools

import (
	"context"
	"strconv"
	"strings"

	"github.com/davidmovas/postulator/internal/application/pagetypes"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	pagetypesClassifyName = "pagetypes_classify"
	opPreview             = "preview"
	opApply               = "apply"
	groupsNamed           = 3
	untypedName           = "untyped"
)

type pagetypesClassifyArgs struct {
	Op     string `json:"op" enum:"preview,apply" description:"apply writes the preview"`
	Hash   string `json:"hash,omitempty" description:"apply: hash from preview"`
	Groups []int  `json:"groups,omitempty" description:"apply: only these group numbers, each with those it dependsOn; default all"`
}

type ClassifyRule struct {
	Name  string `json:"name"`
	Pages int    `json:"pages"`
}

type ClassifyGroup struct {
	Group     int            `json:"group"`
	From      string         `json:"from"`
	To        string         `json:"to"`
	Pages     int            `json:"pages"`
	Rules     []ClassifyRule `json:"rules"`
	DependsOn []int          `json:"dependsOn,omitempty"`
	Samples   []string       `json:"samples"`
}

type PagetypesClassifyResponse struct {
	Hash      string                   `json:"hash,omitempty"`
	Groups    []ClassifyGroup          `json:"groups,omitempty"`
	Moving    int                      `json:"moving"`
	Kept      int                      `json:"kept"`
	Waiting   int                      `json:"waiting"`
	ByDefault int                      `json:"byDefault"`
	Applied   *pagetypes.ApplyResponse `json:"applied,omitempty"`
}

type classifyNames struct {
	types map[string]string
	rules map[string]string
}

func pagetypesClassify(deps Deps) Tool {
	tool := confirming(newSiteTool(Def{
		Name: pagetypesClassifyName,
		Description: "Preview how the site's rules would retype its pages, by group, or apply the preview as the person " +
			"confirms; a type a person set is kept.",
		Risk: RiskWrite,
	}, func(ctx context.Context, b Binding, in pagetypesClassifyArgs) (PagetypesClassifyResponse, error) {
		switch in.Op {
		case opPreview:
			return deps.classifyPreview(ctx, b.SiteID)
		case opApply:
			keys, err := deps.classifyKeys(ctx, b.SiteID, in)
			if err != nil {
				return PagetypesClassifyResponse{}, err
			}
			applied, err := deps.Classify.Apply(ctx, pagetypes.ApplyRequest{SiteID: b.SiteID, Hash: in.Hash, Groups: keys})
			if err != nil {
				return PagetypesClassifyResponse{}, err
			}
			return PagetypesClassifyResponse{Applied: &applied}, nil
		default:
			return PagetypesClassifyResponse{}, errors.New(errors.Invalid, "pagetypes_classify takes the op preview or apply").
				WithDetail("field", "op")
		}
	}), func(ctx context.Context, b Binding, in pagetypesClassifyArgs) (string, error) {
		if in.Op != opApply {
			return "", nil
		}
		return deps.classifySummary(ctx, b.SiteID, in)
	})
	return readsOn(tool, opPreview)
}

func (deps Deps) classifyNames(ctx context.Context, siteID string) (classifyNames, error) {
	set, err := deps.PageTypes.Set(ctx, siteID)
	if err != nil {
		return classifyNames{}, err
	}
	rules, err := deps.PageTypes.Rules(ctx, siteID)
	if err != nil {
		return classifyNames{}, err
	}
	names := classifyNames{types: map[string]string{"": untypedName}, rules: make(map[string]string, len(rules))}
	for _, t := range set.Types() {
		names.types[t.ID] = t.Name
	}
	for i := range rules {
		names.rules[rules[i].ID] = rules[i].Name
	}
	return names, nil
}

func (names classifyNames) rule(id string) string {
	if name, known := names.rules[id]; known {
		return name
	}
	return "a rule since removed"
}

func (deps Deps) classifyPreview(ctx context.Context, siteID string) (PagetypesClassifyResponse, error) {
	preview, err := deps.Classify.Preview(ctx, pagetypes.PreviewRequest{SiteID: siteID})
	if err != nil {
		return PagetypesClassifyResponse{}, err
	}
	names, err := deps.classifyNames(ctx, siteID)
	if err != nil {
		return PagetypesClassifyResponse{}, err
	}
	numbers := make(map[pagetypes.GroupKey]int, len(preview.Groups))
	for i := range preview.Groups {
		numbers[pagetypes.GroupKey{From: preview.Groups[i].From, To: preview.Groups[i].To}] = i + 1
	}
	out := PagetypesClassifyResponse{
		Hash: preview.Hash, Groups: make([]ClassifyGroup, 0, len(preview.Groups)), Moving: preview.Moving,
		Kept: preview.Kept, Waiting: preview.Waiting, ByDefault: preview.ByDefault,
	}
	for i := range preview.Groups {
		group := &preview.Groups[i]
		rules := make([]ClassifyRule, 0, len(group.Rules))
		for _, counted := range group.Rules {
			rules = append(rules, ClassifyRule{Name: names.rule(counted.RuleID), Pages: counted.Pages})
		}
		needs := make([]int, 0, len(group.DependsOn))
		for _, need := range group.DependsOn {
			needs = append(needs, numbers[need])
		}
		out.Groups = append(out.Groups, ClassifyGroup{
			Group: i + 1, From: names.types[group.From], To: names.types[group.To], Pages: group.Pages, Rules: rules,
			DependsOn: needs, Samples: group.Samples,
		})
	}
	return out, nil
}

func (deps Deps) classifyKeys(ctx context.Context, siteID string, in pagetypesClassifyArgs) ([]pagetypes.GroupKey, error) {
	if len(in.Groups) == 0 {
		return nil, nil
	}
	all, err := deps.Classify.Chosen(ctx, pagetypes.ApplyRequest{SiteID: siteID, Hash: in.Hash})
	if err != nil {
		return nil, err
	}
	keys := make([]pagetypes.GroupKey, 0, len(in.Groups))
	for _, number := range in.Groups {
		if number < 1 || number > len(all) {
			return nil, errors.New(errors.Invalid, "group "+strconv.Itoa(number)+" is not in the preview, which has "+
				strconv.Itoa(len(all))+" groups").WithDetail("field", "groups")
		}
		keys = append(keys, pagetypes.GroupKey{From: all[number-1].From, To: all[number-1].To})
	}
	return keys, nil
}

func (deps Deps) classifySummary(ctx context.Context, siteID string, in pagetypesClassifyArgs) (string, error) {
	keys, err := deps.classifyKeys(ctx, siteID, in)
	if err != nil {
		return "", err
	}
	chosen, err := deps.Classify.Chosen(ctx, pagetypes.ApplyRequest{SiteID: siteID, Hash: in.Hash, Groups: keys})
	if err != nil {
		return "", err
	}
	if len(chosen) == 0 {
		return "", errors.New(errors.Invalid, "the preview moves no page, so there is nothing to apply").WithDetail("field", "groups")
	}
	names, err := deps.classifyNames(ctx, siteID)
	if err != nil {
		return "", err
	}
	pages, said := 0, make([]string, 0, groupsNamed)
	for i := range chosen {
		pages += chosen[i].Pages
		if len(said) < groupsNamed {
			said = append(said, strconv.Itoa(chosen[i].Pages)+" "+names.types[chosen[i].From]+" → "+names.types[chosen[i].To])
		}
	}
	summary := "Type " + strconv.Itoa(pages) + " pages by the site's rules: " + strings.Join(said, ", ")
	if more := len(chosen) - len(said); more > 0 {
		summary += " and " + strconv.Itoa(more) + " more groups"
	}
	return summary + "; a page a person typed keeps its type, and nothing is written to the site", nil
}
