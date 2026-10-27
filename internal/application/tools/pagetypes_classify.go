package tools

import (
	"context"
	"strconv"
	"strings"

	"github.com/davidmovas/postulator/internal/application/pagetypes"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	pagetypesClassifyName = "pagetypes_classify"
	opPreview             = "preview"
	opApply               = "apply"
	opRules               = "rules"
	opSaveRule            = "save_rule"
	opDeleteRule          = "delete_rule"
	opReorderRules        = "reorder_rules"
	opTestRule            = "test_rule"
	groupsNamed           = 3
	untypedName           = "untyped"
)

type urlMatchArgs struct {
	Mode  string `json:"mode" enum:"prefix,segment,glob" description:"A path's prefix, one of its segments, or a glob"`
	Value string `json:"value" description:"What the path is matched against"`
}

type rangeArgs struct {
	Min *int `json:"min,omitempty" minimum:"0" description:"Least"`
	Max *int `json:"max,omitempty" minimum:"0" description:"Most"`
}

type ancestorArgs struct {
	PageID string `json:"pageId,omitempty" description:"Below this page"`
	TypeID string `json:"typeId,omitempty" description:"Below a page of this type"`
}

type listingArgs struct {
	Present     bool `json:"present" description:"It lists products, or with false lists none"`
	MinProducts int  `json:"minProducts,omitempty" minimum:"0" description:"At least this many"`
}

type conditionsArgs struct {
	PostTypes []string       `json:"postTypes,omitempty" description:"WordPress post types, any of them"`
	URLs      []urlMatchArgs `json:"urls,omitempty" description:"Paths, any of them"`
	Slugs     []string       `json:"slugs,omitempty" description:"Slugs, any of them"`
	Ancestor  *ancestorArgs  `json:"ancestor,omitempty" description:"Where in the tree"`
	Level     *rangeArgs     `json:"level,omitempty" description:"Depth in the tree, the front page 0"`
	Listing   *listingArgs   `json:"listing,omitempty" description:"Whether it lists products"`
	Schema    []string       `json:"schema,omitempty" enum:"Article,BlogPosting,HowTo,FAQPage,AboutPage,ContactPage,CollectionPage,ItemList" description:"JSON-LD types, any of them"`
	Words     *rangeArgs     `json:"words,omitempty" description:"Words of its body"`
	MinH2     int            `json:"minH2,omitempty" minimum:"0" description:"At least this many H2 headings"`
	Technical *bool          `json:"technical,omitempty" description:"A cart, checkout, account or legal page"`
	FrontPage *bool          `json:"frontPage,omitempty" description:"The front page"`
	Shop      *bool          `json:"shop,omitempty" description:"The shop page"`
}

func (in *conditionsArgs) conditions() pagetype.Conditions {
	if in == nil {
		return pagetype.Conditions{}
	}
	out := pagetype.Conditions{PostTypes: in.PostTypes, Slugs: in.Slugs, MinH2: in.MinH2, Technical: in.Technical, FrontPage: in.FrontPage, Shop: in.Shop}
	for _, url := range in.URLs {
		out.URLs = append(out.URLs, pagetype.URLMatch{Mode: pagetype.URLMode(url.Mode), Value: url.Value})
	}
	if in.Ancestor != nil {
		out.Ancestor = &pagetype.Ancestor{PageID: in.Ancestor.PageID, TypeID: in.Ancestor.TypeID}
	}
	if in.Level != nil {
		out.Level = &pagetype.Range{Min: in.Level.Min, Max: in.Level.Max}
	}
	if in.Listing != nil {
		out.Listing = &pagetype.Listing{Present: in.Listing.Present, MinProducts: in.Listing.MinProducts}
	}
	for _, schema := range in.Schema {
		out.Schema = append(out.Schema, pagetype.SchemaType(schema))
	}
	if in.Words != nil {
		out.Words = &pagetype.Range{Min: in.Words.Min, Max: in.Words.Max}
	}
	return out
}

type pagetypesClassifyArgs struct {
	Op         string          `json:"op" enum:"preview,apply,rules,save_rule,delete_rule,reorder_rules,test_rule" description:"preview how the rules would retype the pages, by group; apply writes it; the rule ops read and change the ordered rules, the first that matches a page wins"`
	Hash       string          `json:"hash,omitempty" description:"apply: hash from preview"`
	Groups     []int           `json:"groups,omitempty" description:"apply: only these group numbers, each with those it dependsOn; default all"`
	RuleID     string          `json:"ruleId,omitempty" description:"save_rule: the rule to change, none to add one; delete_rule, test_rule: the rule"`
	Name       string          `json:"name,omitempty" description:"save_rule: the rule's name"`
	Enabled    *bool           `json:"enabled,omitempty" description:"save_rule: default true"`
	TypeID     string          `json:"typeId,omitempty" description:"save_rule: the page type it gives"`
	Conditions *conditionsArgs `json:"conditions,omitempty" description:"save_rule, test_rule: what a page must meet, every condition at once"`
	RuleIDs    []string        `json:"ruleIds,omitempty" description:"reorder_rules: every rule id in its new order"`
	PageID     string          `json:"pageId,omitempty" description:"test_rule: the page to test, against ruleId or conditions"`
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
	Hash      string                      `json:"hash,omitempty"`
	Groups    []ClassifyGroup             `json:"groups,omitempty"`
	Moving    int                         `json:"moving"`
	Kept      int                         `json:"kept"`
	Waiting   int                         `json:"waiting"`
	ByDefault int                         `json:"byDefault"`
	Applied   *pagetypes.ApplyResponse    `json:"applied,omitempty"`
	Rules     *pagetypes.RulesResponse    `json:"rules,omitempty"`
	Rule      *pagetypes.Rule             `json:"rule,omitempty"`
	Tested    *pagetypes.TestRuleResponse `json:"tested,omitempty"`
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
			return deps.classifyRules(ctx, b, &in)
		}
	}), func(ctx context.Context, b Binding, in pagetypesClassifyArgs) (string, error) {
		if in.Op != opApply {
			return "", nil
		}
		return deps.classifySummary(ctx, b.SiteID, in)
	})
	tool = describing(tool, func(ctx context.Context, b Binding, in pagetypesClassifyArgs) (string, error) {
		return deps.ruleChange(ctx, b, &in)
	})
	return readsOn(tool, opPreview, opRules, opTestRule)
}

func (deps Deps) classifyRules(ctx context.Context, b Binding, in *pagetypesClassifyArgs) (PagetypesClassifyResponse, error) {
	var out PagetypesClassifyResponse
	switch in.Op {
	case opRules:
		listed, err := deps.PageTypes.ListRules(ctx, pagetypes.ListRulesRequest{SiteID: b.SiteID})
		out.Rules = &listed
		return out, err
	case opSaveRule:
		enabled := in.Enabled == nil || *in.Enabled
		saved, err := deps.PageTypes.SaveRule(ctx, pagetypes.SaveRuleRequest{
			ID: in.RuleID, SiteID: b.SiteID, Name: in.Name, Enabled: enabled, TypeID: in.TypeID, Conditions: in.Conditions.conditions(),
		})
		out.Rule = &saved
		return out, err
	case opDeleteRule:
		_, err := deps.PageTypes.DeleteRule(ctx, pagetypes.DeleteRuleRequest{ID: in.RuleID})
		return out, err
	case opReorderRules:
		listed, err := deps.PageTypes.ReorderRules(ctx, pagetypes.ReorderRulesRequest{SiteID: b.SiteID, RuleIDs: in.RuleIDs})
		out.Rules = &listed
		return out, err
	case opTestRule:
		req := pagetypes.TestRuleRequest{SiteID: b.SiteID, PageID: in.PageID, RuleID: in.RuleID}
		if in.RuleID == "" {
			req.Rule = &pagetypes.SaveRuleRequest{SiteID: b.SiteID, Name: in.Name, Enabled: true, TypeID: in.TypeID, Conditions: in.Conditions.conditions()}
		}
		tested, err := deps.Classify.TestRule(ctx, req)
		out.Tested = &tested
		return out, err
	default:
		return out, errors.New(errors.Invalid, "pagetypes_classify takes the op preview, apply, rules, save_rule, delete_rule, "+
			"reorder_rules or test_rule").WithDetail("field", "op")
	}
}

func (deps Deps) ruleChange(ctx context.Context, b Binding, in *pagetypesClassifyArgs) (string, error) {
	switch in.Op {
	case opSaveRule:
		kind, err := deps.ownType(ctx, b, in.TypeID)
		if err != nil {
			return "", err
		}
		if in.RuleID == "" {
			return "Add the rule " + quoted(in.Name) + ", which types the pages it matches " + kind.Name + "; nothing moves until a preview is applied", nil
		}
		return "Change the rule " + quoted(in.Name) + " to type the pages it matches " + kind.Name + "; nothing moves until a preview is applied", nil
	case opDeleteRule:
		return "Delete the rule " + in.RuleID + "; nothing moves until a preview is applied", nil
	default:
		return "Put the site's " + strconv.Itoa(len(in.RuleIDs)) + " rules in a new order; the first that matches a page wins", nil
	}
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
