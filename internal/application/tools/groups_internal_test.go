package tools

import (
	"slices"
	"strings"
	"testing"
)

func registeredDefs() []Def {
	defs := make([]Def, 0, len(factories()))
	for _, build := range factories() {
		defs = append(defs, build(Deps{}).Def)
	}
	return defs
}

func TestEveryToolBelongsToADeclaredGroupAndEveryGroupHoldsOne(t *testing.T) {
	t.Parallel()

	held := map[string]int{}
	for _, def := range registeredDefs() {
		group, found := groupOf(def.Name)
		if !found {
			t.Errorf("%s belongs to no declared group, so deferred loading could not offer it", def.Name)
			continue
		}
		if !strings.HasPrefix(def.Name, group.Name+"_") {
			t.Errorf("%s was put in the group %s", def.Name, group.Name)
		}
		held[group.Name]++
	}

	seen := map[string]bool{}
	for _, group := range groups {
		if seen[group.Name] {
			t.Errorf("the group %s is declared twice", group.Name)
		}
		seen[group.Name] = true
		if strings.TrimSpace(group.Description) == "" || !strings.HasSuffix(group.Description, ".") {
			t.Errorf("the group %s says %q, want a sentence the model can search by", group.Name, group.Description)
		}
		if held[group.Name] == 0 {
			t.Errorf("the group %s holds no tool", group.Name)
		}
	}
}

func TestTheToolsSentWholeAreReadsTheModelOrientsBy(t *testing.T) {
	t.Parallel()

	defs := registeredDefs()
	for _, name := range orienting {
		at := slices.IndexFunc(defs, func(def Def) bool { return def.Name == name })
		if at < 0 {
			t.Errorf("%s is sent whole but is not registered", name)
			continue
		}
		if defs[at].Risk != RiskRead {
			t.Errorf("%s is sent whole but its risk is %s; only a read orients the model", name, defs[at].Risk)
		}
	}
}

func TestOnDemandNamesTheGroupOfEveryToolButTheOrientingReads(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		tool  string
		group string
		found bool
	}{
		{name: "an orienting read is sent whole", tool: "pages_list"},
		{name: "the overview is sent whole although its group is not", tool: "reports_site_overview"},
		{name: "a write loads on demand", tool: "pages_create", group: "pages", found: true},
		{name: "a read outside the orienting set loads on demand", tool: "reports_link_audit", group: "reports", found: true},
		{name: "a name of an unknown group", tool: "weather_today"},
		{name: "a name without a group", tool: "pages"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			group, found := OnDemand(tc.tool)
			if found != tc.found || group.Name != tc.group {
				t.Fatalf("OnDemand(%s) = %+v, %t, want %q, %t", tc.tool, group, found, tc.group, tc.found)
			}
			if found && group.Description == "" {
				t.Errorf("the group %s carries no description", group.Name)
			}
		})
	}
}
