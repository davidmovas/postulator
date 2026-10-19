package graph

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/domain/finding"
	graphdomain "github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const (
	ResolutionAccepted = "accepted"
	ResolutionIgnored  = "ignored"
	ResolvedByPerson   = "person"

	ReasonCardUnchanged = "unchanged"
	ReasonCardDismissed = "dismissed"
	ReasonCardChanged   = "card_changed"

	maxCardsPerCall = 50
)

type CardInput struct {
	EntityID   string `json:"entityId"`
	Definition string `json:"definition"`
	Facts      []Fact `json:"facts"`
	Reason     string `json:"reason"`
}

type Card struct {
	Definition string `json:"definition"`
	Facts      []Fact `json:"facts"`
}

type CardProposal struct {
	FindingID string   `json:"findingId"`
	EntityID  string   `json:"entityId"`
	Name      string   `json:"name"`
	Current   Card     `json:"current"`
	Proposed  Card     `json:"proposed"`
	Reason    string   `json:"reason"`
	Changed   bool     `json:"changed"`
	CreatedAt dto.Time `json:"createdAt"`
}

type CardSkip struct {
	EntityID string `json:"entityId"`
	Reason   string `json:"reason"`
}

type ProposeCardsRequest struct {
	SiteID string      `json:"siteId"`
	Cards  []CardInput `json:"cards"`
}

type ProposeCardsResponse struct {
	Proposals []CardProposal `json:"proposals"`
	Skipped   []CardSkip     `json:"skipped"`
}

type ApplyCardsRequest struct {
	SiteID string      `json:"siteId"`
	Cards  []CardInput `json:"cards"`
}

type ApplyCardsResponse struct {
	Entities []Entity   `json:"entities"`
	Skipped  []CardSkip `json:"skipped"`
}

type AcceptCardRequest struct {
	FindingID string `json:"findingId"`
}

type AcceptCardResponse struct {
	Entity Entity `json:"entity"`
}

type DismissCardRequest struct {
	FindingID string `json:"findingId"`
}

type DismissCardResponse struct {
	FindingID string `json:"findingId"`
}

type CardProposalsRequest struct {
	SiteID string `json:"siteId"`
	Cursor string `json:"cursor"`
	Limit  int    `json:"limit"`
}

type cardSubject struct {
	EntityID   string `json:"entityId"`
	Name       string `json:"name"`
	Base       string `json:"base"`
	Definition string `json:"definition"`
	Facts      []Fact `json:"facts"`
	Reason     string `json:"reason"`
}

type claimedCard struct {
	current  graphdomain.Entity
	proposed graphdomain.Entity
	reason   string
}

func (s *Service) ProposeCards(ctx context.Context, req ProposeCardsRequest) (ProposeCardsResponse, error) {
	out := ProposeCardsResponse{Proposals: []CardProposal{}, Skipped: []CardSkip{}}
	now := s.now()
	err := s.uow.Do(ctx, func(c context.Context) error {
		claimed, skipped, err := s.claimCards(c, req.SiteID, req.Cards)
		if err != nil {
			return err
		}
		out.Skipped = skipped
		for i := range claimed {
			if raiseErr := s.raiseCard(c, req.SiteID, claimed[i], now); raiseErr != nil {
				return raiseErr
			}
		}
		open, err := s.openCards(c, req.SiteID)
		if err != nil {
			return err
		}
		for i := range claimed {
			proposal, held := open[claimed[i].current.ID]
			if !held {
				out.Skipped = append(out.Skipped, CardSkip{EntityID: claimed[i].current.ID, Reason: ReasonCardDismissed})
				continue
			}
			out.Proposals = append(out.Proposals, proposal)
		}
		return nil
	})
	if err != nil {
		return ProposeCardsResponse{}, err
	}
	if len(out.Proposals) > 0 {
		if publishErr := s.changed(req.SiteID); publishErr != nil {
			return ProposeCardsResponse{}, publishErr
		}
	}
	return out, nil
}

func (s *Service) ApplyCards(ctx context.Context, req ApplyCardsRequest) (ApplyCardsResponse, error) {
	out := ApplyCardsResponse{Entities: []Entity{}, Skipped: []CardSkip{}}
	now := s.now()
	err := s.uow.Do(ctx, func(c context.Context) error {
		claimed, skipped, err := s.claimCards(c, req.SiteID, req.Cards)
		if err != nil {
			return err
		}
		out.Skipped = skipped
		for i := range claimed {
			written, writeErr := s.writeCard(c, claimed[i].current, claimed[i].proposed.Definition, claimed[i].proposed.Facts, now)
			if writeErr != nil {
				return writeErr
			}
			out.Entities = append(out.Entities, entityView(written))
		}
		return nil
	})
	if err != nil {
		return ApplyCardsResponse{}, err
	}
	if len(out.Entities) > 0 {
		if publishErr := s.changed(req.SiteID); publishErr != nil {
			return ApplyCardsResponse{}, publishErr
		}
	}
	return out, nil
}

func (s *Service) AcceptCard(ctx context.Context, req AcceptCardRequest) (AcceptCardResponse, error) {
	var written graphdomain.Entity
	now := s.now()
	err := s.uow.Do(ctx, func(c context.Context) error {
		held, subject, err := s.openProposal(c, req.FindingID)
		if err != nil {
			return err
		}
		current, err := s.entities.Get(c, subject.EntityID)
		if err != nil {
			return err
		}
		if current.CardHash() != subject.Base {
			return errors.New(errors.Conflict, "the entity's card changed since it was proposed; propose it again").
				WithDetail("reason", ReasonCardChanged).WithDetail("entityId", current.ID)
		}
		facts := make([]graphdomain.Fact, 0, len(subject.Facts))
		for _, fact := range subject.Facts {
			facts = append(facts, graphdomain.Fact(fact))
		}
		if written, err = s.writeCard(c, current, subject.Definition, facts, now); err != nil {
			return err
		}
		return s.findings.Settle(c, held.ID, finding.Resolved, ResolutionAccepted, ResolvedByPerson, now)
	})
	if err != nil {
		return AcceptCardResponse{}, err
	}
	if publishErr := s.changed(written.SiteID); publishErr != nil {
		return AcceptCardResponse{}, publishErr
	}
	return AcceptCardResponse{Entity: entityView(written)}, nil
}

func (s *Service) DismissCard(ctx context.Context, req DismissCardRequest) (DismissCardResponse, error) {
	var siteID string
	err := s.uow.Do(ctx, func(c context.Context) error {
		held, _, err := s.openProposal(c, req.FindingID)
		if err != nil {
			return err
		}
		siteID = held.SiteID
		return s.findings.Settle(c, held.ID, finding.Dismissed, ResolutionIgnored, ResolvedByPerson, s.now())
	})
	if err != nil {
		return DismissCardResponse{}, err
	}
	if publishErr := s.changed(siteID); publishErr != nil {
		return DismissCardResponse{}, publishErr
	}
	return DismissCardResponse(req), nil
}

func (s *Service) CardProposals(ctx context.Context, req CardProposalsRequest) (paging.List[CardProposal], error) {
	if err := requireSite(req.SiteID); err != nil {
		return paging.List[CardProposal]{}, err
	}
	open, err := s.findings.OpenBySite(ctx, req.SiteID, finding.EntityCardProposed)
	if err != nil {
		return paging.List[CardProposal]{}, err
	}
	rows := make([]CardProposal, 0, len(open))
	for i := range open {
		proposal, viewErr := s.proposalView(ctx, open[i])
		if viewErr != nil {
			return paging.List[CardProposal]{}, viewErr
		}
		rows = append(rows, proposal)
	}
	keyset := paging.Keyset[CardProposal]{
		IDColumn: "id",
		ID:       func(p CardProposal) string { return p.FindingID },
		Keys: []paging.SortKey[CardProposal]{
			paging.TimeKey("createdAt", "created_at", func(p CardProposal) any { return p.CreatedAt.Std() }),
		},
	}
	return keyset.Page(rows, paging.Request{After: paging.Cursor(req.Cursor), Limit: dto.PageSize(req.Limit)})
}

func (s *Service) claimCards(ctx context.Context, siteID string, cards []CardInput) ([]claimedCard, []CardSkip, error) {
	if err := requireSite(siteID); err != nil {
		return nil, nil, err
	}
	switch {
	case len(cards) == 0:
		return nil, nil, errors.New(errors.Invalid, "name at least one card").WithDetail("field", "cards")
	case len(cards) > maxCardsPerCall:
		return nil, nil, errors.New(errors.Invalid, "at most "+strconv.Itoa(maxCardsPerCall)+" cards at a time").WithDetail("field", "cards")
	}
	claimed := make([]claimedCard, 0, len(cards))
	skipped := make([]CardSkip, 0)
	seen := make(map[string]struct{}, len(cards))
	for i := range cards {
		field := "cards[" + strconv.Itoa(i) + "]"
		card := &cards[i]
		current, err := s.entities.Get(ctx, card.EntityID)
		if err != nil || current.SiteID != siteID {
			return nil, nil, errors.New(errors.NotFound, "the site holds no such entity").WithDetail("field", field+".entityId")
		}
		if _, twice := seen[current.ID]; twice {
			return nil, nil, errors.New(errors.Invalid, "propose one card per entity").WithDetail("field", field+".entityId")
		}
		seen[current.ID] = struct{}{}
		proposed, err := proposedCard(current, card, field)
		if err != nil {
			return nil, nil, err
		}
		if proposed.CardHash() == current.CardHash() {
			skipped = append(skipped, CardSkip{EntityID: current.ID, Reason: ReasonCardUnchanged})
			continue
		}
		claimed = append(claimed, claimedCard{current: current, proposed: proposed, reason: strings.TrimSpace(card.Reason)})
	}
	return claimed, skipped, nil
}

func proposedCard(current graphdomain.Entity, card *CardInput, field string) (graphdomain.Entity, error) {
	definition := strings.TrimSpace(card.Definition)
	if definition == "" && card.Facts == nil {
		return graphdomain.Entity{}, errors.New(errors.Invalid, "a card proposes a definition, facts or both").WithDetail("field", field)
	}
	proposed := current
	if definition != "" {
		proposed.Definition = definition
	}
	if card.Facts != nil {
		proposed.Facts = make([]graphdomain.Fact, 0, len(card.Facts))
		for j, fact := range card.Facts {
			if strings.TrimSpace(fact.Label) == "" || strings.TrimSpace(fact.Value) == "" {
				return graphdomain.Entity{}, errors.New(errors.Invalid, "a fact needs both what it is about and the fact").
					WithDetail("field", field+".facts["+strconv.Itoa(j)+"]")
			}
			proposed.Facts = append(proposed.Facts, graphdomain.Fact(fact))
		}
	}
	proposed.Facts = graphdomain.CleanFacts(proposed.Facts)
	return proposed, nil
}

func (s *Service) raiseCard(ctx context.Context, siteID string, claimed claimedCard, now time.Time) error {
	facts := make([]Fact, 0, len(claimed.proposed.Facts))
	for _, fact := range claimed.proposed.Facts {
		facts = append(facts, Fact(fact))
	}
	subject, err := json.Marshal(cardSubject{
		EntityID: claimed.current.ID, Name: claimed.current.Name, Base: claimed.current.CardHash(),
		Definition: claimed.proposed.Definition, Facts: facts, Reason: claimed.reason,
	})
	if err != nil {
		return errors.Wrap(err, errors.Internal, "encode the card proposal")
	}
	entityID := claimed.current.ID
	raised, err := finding.New(finding.Finding{
		ID: id.New(), SiteID: siteID, EntityID: &entityID, Code: finding.EntityCardProposed,
		SubjectKey: finding.SubjectEntityCard(entityID), Subject: subject, CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		return err
	}
	return s.findings.Upsert(ctx, raised)
}

func (s *Service) openCards(ctx context.Context, siteID string) (map[string]CardProposal, error) {
	open, err := s.findings.OpenBySite(ctx, siteID, finding.EntityCardProposed)
	if err != nil {
		return nil, err
	}
	out := make(map[string]CardProposal, len(open))
	for i := range open {
		proposal, viewErr := s.proposalView(ctx, open[i])
		if viewErr != nil {
			return nil, viewErr
		}
		out[proposal.EntityID] = proposal
	}
	return out, nil
}

func (s *Service) openProposal(ctx context.Context, findingID string) (finding.Finding, cardSubject, error) {
	if strings.TrimSpace(findingID) == "" {
		return finding.Finding{}, cardSubject{}, errors.New(errors.Invalid, "name the proposal").WithDetail("field", "findingId")
	}
	held, err := s.findings.Get(ctx, findingID)
	if err != nil {
		return finding.Finding{}, cardSubject{}, err
	}
	if held.Code != finding.EntityCardProposed {
		return finding.Finding{}, cardSubject{}, errors.New(errors.NotFound, "no card is proposed under this id").WithDetail("findingId", findingID)
	}
	if held.Status != finding.Open {
		return finding.Finding{}, cardSubject{}, errors.New(errors.Conflict, "the proposal is already settled").WithDetail("findingId", findingID)
	}
	subject, err := subjectOf(held)
	return held, subject, err
}

func subjectOf(held finding.Finding) (cardSubject, error) {
	var subject cardSubject
	if err := json.Unmarshal(held.Subject, &subject); err != nil {
		return cardSubject{}, errors.Wrap(err, errors.Internal, "read the card proposal")
	}
	if subject.Facts == nil {
		subject.Facts = []Fact{}
	}
	return subject, nil
}

func (s *Service) proposalView(ctx context.Context, held finding.Finding) (CardProposal, error) {
	subject, err := subjectOf(held)
	if err != nil {
		return CardProposal{}, err
	}
	current, err := s.entities.Get(ctx, subject.EntityID)
	if err != nil {
		return CardProposal{}, err
	}
	view := entityView(current)
	return CardProposal{
		FindingID: held.ID, EntityID: current.ID, Name: current.Name,
		Current:  Card{Definition: view.Definition, Facts: view.Facts},
		Proposed: Card{Definition: subject.Definition, Facts: subject.Facts},
		Reason:   subject.Reason, Changed: current.CardHash() != subject.Base, CreatedAt: dto.NewTime(held.CreatedAt),
	}, nil
}

func (s *Service) writeCard(ctx context.Context, current graphdomain.Entity, definition string, facts []graphdomain.Fact,
	now time.Time) (graphdomain.Entity, error) {
	next := current
	next.Definition, next.Facts, next.UpdatedAt = definition, facts, now
	valid, err := graphdomain.NewEntity(next)
	if err != nil {
		return graphdomain.Entity{}, err
	}
	if updateErr := s.entities.Update(ctx, valid); updateErr != nil {
		return graphdomain.Entity{}, updateErr
	}
	return valid, nil
}
