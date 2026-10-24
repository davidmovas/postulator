package contextpack

import (
	"cmp"
	"slices"
	"strconv"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	ReasonBudgetTooSmall = "budget_too_small"
	keptPriority         = 1
	headroomDivisor      = 20
)

type Drop struct {
	Section SectionName `json:"section"`
	Heading string      `json:"heading"`
	Items   int         `json:"items"`
	Tokens  int         `json:"tokens"`
	Whole   bool        `json:"whole"`
}

type Fitted struct {
	Field    field.Key `json:"field"`
	Sections []Section `json:"sections"`
	Dropped  []Drop    `json:"dropped"`
	Tokens   int       `json:"tokens"`
	Budget   int       `json:"budget"`
}

func Fit(pack Pack, budget int) (Fitted, error) {
	sections := make([]Section, len(pack.Sections))
	for i := range pack.Sections {
		sections[i] = pack.Sections[i]
		sections[i].Lines = slices.Clone(pack.Sections[i].Lines)
		sections[i].Items = slices.Clone(pack.Sections[i].Items)
	}
	fitted := Fitted{Field: pack.Field, Dropped: make([]Drop, 0), Tokens: pack.Tokens, Budget: budget}
	if budget > 0 {
		room := budget - budget/headroomDivisor
		for _, at := range trimOrder(sections) {
			if fitted.Tokens <= room {
				break
			}
			if sections[at].Priority <= keptPriority {
				return Fitted{}, errors.New(errors.Invalid, "the input budget is smaller than what every call must carry").
					WithDetail("reason", ReasonBudgetTooSmall).WithDetail("budget", budget).WithDetail("needed", kept(sections))
			}
			fitted.Tokens -= trim(&sections[at], room, fitted.Tokens, &fitted.Dropped)
		}
		if fitted.Tokens > room {
			return Fitted{}, errors.New(errors.Invalid, "the input budget is smaller than what every call must carry").
				WithDetail("reason", ReasonBudgetTooSmall).WithDetail("budget", budget).WithDetail("needed", kept(sections))
		}
	}
	fitted.Sections = make([]Section, 0, len(sections))
	for i := range sections {
		if sections[i].Tokens > 0 {
			fitted.Sections = append(fitted.Sections, sections[i])
		}
	}
	return fitted, nil
}

func trimOrder(sections []Section) []int {
	order := make([]int, len(sections))
	for i := range order {
		order[i] = i
	}
	slices.SortStableFunc(order, func(a, b int) int {
		if sections[a].Priority != sections[b].Priority {
			return cmp.Compare(sections[b].Priority, sections[a].Priority)
		}
		return cmp.Compare(b, a)
	})
	return order
}

func trim(section *Section, budget, total int, dropped *[]Drop) int {
	removed, items := 0, 0
	for len(section.Items) > 0 && total-removed > budget {
		last := section.Items[len(section.Items)-1]
		removed += llm.Tokens(last)
		section.Items = section.Items[:len(section.Items)-1]
		items++
	}
	whole := total-removed > budget || (len(section.Items) == 0 && len(section.Lines) == 0)
	if whole {
		removed += section.Tokens - removed
		items += len(section.Items)
		section.Items, section.Lines = nil, nil
	}
	if removed == 0 {
		return 0
	}
	*dropped = append(*dropped, Drop{Section: section.Name, Heading: section.Heading, Items: items, Tokens: removed, Whole: whole})
	section.Tokens -= removed
	return removed
}

func kept(sections []Section) int {
	total := 0
	for i := range sections {
		if sections[i].Priority <= keptPriority {
			total += sections[i].Tokens
		}
	}
	return total
}

func (f Fitted) Text() string {
	var builder strings.Builder
	for i, section := range f.Sections {
		if i > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString(strings.ToUpper(section.Heading))
		builder.WriteString("\n")
		for _, line := range section.Lines {
			builder.WriteString(line)
			builder.WriteString("\n")
		}
		for _, item := range section.Items {
			builder.WriteString(item)
			builder.WriteString("\n")
		}
	}
	return builder.String()
}

func (f Fitted) Section(name SectionName) (Section, bool) {
	for _, section := range f.Sections {
		if section.Name == name {
			return section, true
		}
	}
	return Section{}, false
}

func (d Drop) String() string {
	if d.Whole {
		return d.Heading + ": dropped whole, " + strconv.Itoa(d.Tokens) + " tokens"
	}
	return d.Heading + ": " + strconv.Itoa(d.Items) + " item(s) dropped, " + strconv.Itoa(d.Tokens) + " tokens"
}
