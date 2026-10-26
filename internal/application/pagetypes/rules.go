package pagetypes

import (
	"context"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

const (
	reasonFallbackFixed = "fallback_fixed"
	reasonRuleLimit     = "rule_limit"
	reasonOrderMismatch = "order_mismatch"
	reasonFallbackRole  = "fallback_role"
)

type ruleStore interface {
	Insert(ctx context.Context, rule pagetype.Rule) error
	Update(ctx context.Context, rule pagetype.Rule) error
	Delete(ctx context.Context, id string) error
	Get(ctx context.Context, id string) (pagetype.Rule, error)
	ListBySite(ctx context.Context, siteID string) ([]pagetype.Rule, error)
}

func (s *Service) ListRules(ctx context.Context, req ListRulesRequest) (RulesResponse, error) {
	if strings.TrimSpace(req.SiteID) == "" {
		return RulesResponse{}, errors.New(errors.Invalid, "a site is required").WithDetail("field", "siteId")
	}
	rules, err := s.deps.Rules.ListBySite(ctx, req.SiteID)
	if err != nil {
		return RulesResponse{}, err
	}
	out := RulesResponse{Rules: make([]Rule, 0, len(rules))}
	for i := range rules {
		if rules[i].Kind == pagetype.KindFallback {
			view := ruleView(rules[i], 0)
			out.Fallback = &view
			continue
		}
		out.Rules = append(out.Rules, ruleView(rules[i], len(out.Rules)+1))
	}
	return out, nil
}

func (s *Service) SaveRule(ctx context.Context, req SaveRuleRequest) (Rule, error) {
	var saved pagetype.Rule
	err := s.deps.UnitOfWork.Do(ctx, func(c context.Context) error {
		var err error
		saved, err = s.saveRule(c, req)
		return err
	})
	if err != nil {
		return Rule{}, err
	}
	if publishErr := s.changed(saved.SiteID); publishErr != nil {
		return Rule{}, publishErr
	}
	return s.ruleOf(ctx, saved)
}

func (s *Service) saveRule(ctx context.Context, req SaveRuleRequest) (pagetype.Rule, error) {
	now := s.now()
	var rule pagetype.Rule
	if strings.TrimSpace(req.ID) == "" {
		siblings, err := s.deps.Rules.ListBySite(ctx, req.SiteID)
		if err != nil {
			return pagetype.Rule{}, err
		}
		if len(siblings) >= pagetype.RuleLimit {
			return pagetype.Rule{}, errors.New(errors.Invalid, "a site holds at most "+strconv.Itoa(pagetype.RuleLimit)+" rules").
				WithDetail("field", "siteId").WithDetail("reason", reasonRuleLimit)
		}
		rule = pagetype.Rule{ID: id.New(), SiteID: req.SiteID, Kind: pagetype.KindRule, Position: lastRule(siblings) + 1, CreatedAt: now}
		if roomErr := s.makeRoom(ctx, siblings, rule.Position, now); roomErr != nil {
			return pagetype.Rule{}, roomErr
		}
	} else {
		held, err := s.deps.Rules.Get(ctx, req.ID)
		if err != nil {
			return pagetype.Rule{}, err
		}
		rule = held
		if rule.Kind == pagetype.KindFallback && (!req.Conditions.Empty() || !req.Enabled) {
			return pagetype.Rule{}, errors.New(errors.Invalid, "the rule for anything no rule matched keeps no conditions and stays on").
				WithDetail("field", "conditions").WithDetail("reason", reasonFallbackFixed)
		}
	}
	rule.Name, rule.Enabled, rule.TypeID = strings.TrimSpace(req.Name), req.Enabled, strings.TrimSpace(req.TypeID)
	rule.Conditions, rule.UpdatedAt = req.Conditions.Normalized(), now
	set, err := s.Set(ctx, rule.SiteID)
	if err != nil {
		return pagetype.Rule{}, err
	}
	if invalidErr := rule.Validate(set); invalidErr != nil {
		return pagetype.Rule{}, invalidErr
	}
	if pageErr := s.pageOfSite(ctx, rule); pageErr != nil {
		return pagetype.Rule{}, pageErr
	}
	if strings.TrimSpace(req.ID) == "" {
		return rule, s.deps.Rules.Insert(ctx, rule)
	}
	return rule, s.deps.Rules.Update(ctx, rule)
}

func (s *Service) pageOfSite(ctx context.Context, rule pagetype.Rule) error {
	if rule.Conditions.Ancestor == nil || rule.Conditions.Ancestor.PageID == "" {
		return nil
	}
	page, err := s.deps.Pages.Get(ctx, rule.Conditions.Ancestor.PageID)
	if errors.IsCode(err, errors.NotFound) || err == nil && page.SiteID != rule.SiteID {
		return errors.New(errors.Invalid, "the page above must be one of the site's pages").WithDetail("field", "conditions.ancestor.pageId")
	}
	return err
}

func lastRule(rules []pagetype.Rule) int {
	last := 0
	for i := range rules {
		if rules[i].Kind == pagetype.KindRule && rules[i].Position > last {
			last = rules[i].Position
		}
	}
	return last
}

func (s *Service) makeRoom(ctx context.Context, rules []pagetype.Rule, at int, now time.Time) error {
	for i := range rules {
		if rules[i].Kind == pagetype.KindFallback && rules[i].Position <= at {
			rules[i].Position, rules[i].UpdatedAt = at+1, now
			if err := s.deps.Rules.Update(ctx, rules[i]); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) DeleteRule(ctx context.Context, req DeleteRuleRequest) error {
	var siteID string
	err := s.deps.UnitOfWork.Do(ctx, func(c context.Context) error {
		rule, err := s.deps.Rules.Get(c, req.ID)
		if err != nil {
			return err
		}
		if rule.Kind == pagetype.KindFallback {
			return errors.New(errors.Invalid, "the rule for anything no rule matched stays; give it another type instead").
				WithDetail("field", "id").WithDetail("reason", reasonFallbackFixed)
		}
		siteID = rule.SiteID
		return s.deps.Rules.Delete(c, rule.ID)
	})
	if err != nil {
		return err
	}
	return s.changed(siteID)
}

func (s *Service) ReorderRules(ctx context.Context, req ReorderRulesRequest) (RulesResponse, error) {
	err := s.deps.UnitOfWork.Do(ctx, func(c context.Context) error {
		rules, err := s.deps.Rules.ListBySite(c, req.SiteID)
		if err != nil {
			return err
		}
		held := make(map[string]pagetype.Rule, len(rules))
		var fallback *pagetype.Rule
		for i := range rules {
			if rules[i].Kind == pagetype.KindFallback {
				fallback = &rules[i]
				continue
			}
			held[rules[i].ID] = rules[i]
		}
		if len(req.RuleIDs) != len(held) || !everyHeld(req.RuleIDs, held) {
			return errors.New(errors.Invalid, "the new order names every rule of the site once; look at the rules again").
				WithDetail("field", "ruleIds").WithDetail("reason", reasonOrderMismatch)
		}
		now := s.now()
		for i, ruleID := range req.RuleIDs {
			rule := held[ruleID]
			if rule.Position == i+1 {
				continue
			}
			rule.Position, rule.UpdatedAt = i+1, now
			if updateErr := s.deps.Rules.Update(c, rule); updateErr != nil {
				return updateErr
			}
		}
		if fallback != nil && fallback.Position != len(req.RuleIDs)+1 {
			fallback.Position, fallback.UpdatedAt = len(req.RuleIDs)+1, now
			return s.deps.Rules.Update(c, *fallback)
		}
		return nil
	})
	if err != nil {
		return RulesResponse{}, err
	}
	if publishErr := s.changed(req.SiteID); publishErr != nil {
		return RulesResponse{}, publishErr
	}
	return s.ListRules(ctx, ListRulesRequest{SiteID: req.SiteID})
}

func everyHeld(ids []string, held map[string]pagetype.Rule) bool {
	seen := make(map[string]bool, len(ids))
	for _, ruleID := range ids {
		if _, known := held[ruleID]; !known || seen[ruleID] {
			return false
		}
		seen[ruleID] = true
	}
	return true
}

func (s *Service) repointRules(ctx context.Context, siteID, fromTypeID string, to pagetype.Type) error {
	rules, err := s.deps.Rules.ListBySite(ctx, siteID)
	if err != nil {
		return err
	}
	for i := range rules {
		rule := rules[i]
		moved := false
		if rule.TypeID == fromTypeID {
			rule.TypeID, moved = to.ID, true
		}
		if rule.Conditions.Ancestor != nil && rule.Conditions.Ancestor.TypeID == fromTypeID {
			rule.Conditions.Ancestor = &pagetype.Ancestor{TypeID: to.ID}
			moved = true
		}
		if !moved {
			continue
		}
		if rule.Kind == pagetype.KindFallback && !pagetype.Fits(to.Preset.Role(), pagemap.WPPage) {
			return errors.New(errors.Invalid, "the pages no rule matches would move to "+to.Name+", which cannot hold an ordinary page; choose another type").
				WithDetail("field", "reassignTo").WithDetail("reason", reasonFallbackRole)
		}
		rule.UpdatedAt = s.now()
		if updateErr := s.deps.Rules.Update(ctx, rule); updateErr != nil {
			return updateErr
		}
	}
	return nil
}

func (s *Service) seedRules(ctx context.Context, siteID string) error {
	set, err := s.Set(ctx, siteID)
	if err != nil {
		return err
	}
	rules := pagetype.DefaultRules(set, siteID, s.now(), id.New)
	for i := range rules {
		if insertErr := s.deps.Rules.Insert(ctx, rules[i]); insertErr != nil {
			return insertErr
		}
	}
	return nil
}

func (s *Service) ruleOf(ctx context.Context, rule pagetype.Rule) (Rule, error) {
	listed, err := s.ListRules(ctx, ListRulesRequest{SiteID: rule.SiteID})
	if err != nil {
		return Rule{}, err
	}
	if listed.Fallback != nil && listed.Fallback.ID == rule.ID {
		return *listed.Fallback, nil
	}
	at := slices.IndexFunc(listed.Rules, func(view Rule) bool { return view.ID == rule.ID })
	if at < 0 {
		return Rule{}, errors.New(errors.NotFound, "classification rule not found").WithDetail("ruleId", rule.ID)
	}
	return listed.Rules[at], nil
}
