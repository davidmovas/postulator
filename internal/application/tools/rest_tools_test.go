package tools_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/application/schedules"
	"github.com/davidmovas/postulator/internal/application/tools"
	domainagent "github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestTheSiteReadsAnswerTheListOneSiteAndItsLiveOverview(t *testing.T) {
	t.Parallel()

	registry, binding, seeded := wired(t)
	binding.Mode = domainagent.ModeConfirm

	listed := called[tools.SitesResponse](t, registry, binding, "sites", `{"view":"list"}`)
	if listed.Sites == nil || len(listed.Sites.Items) != 1 {
		t.Errorf("sites list = %+v, want the one seeded site", listed.Sites)
	}
	overview := called[tools.SitesResponse](t, registry, binding, "sites", `{"view":"overview"}`)
	if overview.Overview == nil || overview.Overview.Site.ID != seeded.site || overview.Overview.Brief.Overview.Pages != 1 {
		t.Errorf("sites overview = %+v, want the site with its totals and its brief", overview.Overview)
	}
	if _, err := registry.Call(t.Context(), tools.Binding{Mode: domainagent.ModeConfirm}, "sites", json.RawMessage(`{"view":"overview"}`)); !errors.IsCode(err, errors.Unauthorized) {
		t.Errorf("an overview without a site = %v, want Unauthorized", err)
	}
}

func TestTheModelReadsAnswerTheListTheProfilesAndTheSpend(t *testing.T) {
	t.Parallel()

	registry, binding, _ := wired(t)
	listed := called[tools.ModelsResponse](t, registry, binding, "models", `{"view":"list"}`)
	if listed.Models == nil || len(listed.Models.Models) == 0 {
		t.Errorf("models list = %+v, want the catalog", listed.Models)
	}
	profiles := called[tools.ModelsResponse](t, registry, binding, "models", `{"view":"profiles"}`)
	if profiles.Profiles == nil || len(profiles.Profiles.Profiles) == 0 {
		t.Errorf("models profiles = %+v, want a model per job", profiles.Profiles)
	}
	spent := called[tools.ModelsResponse](t, registry, binding, "models", `{"view":"usage","conversationId":"`+binding.ConversationID+`"}`)
	if spent.Usage == nil {
		t.Error("models usage answered no spend")
	}
	for _, gone := range []string{"models_upsert", "models_disable", "models_set_profile", "models_test_provider", "models_set_provider_key"} {
		if _, known := registry.Lookup(gone); known {
			t.Errorf("%s is still offered; the catalog is the person's, in Settings", gone)
		}
	}
}

func TestEverySiteImportAndScheduleChangeIsSaidInOneSentence(t *testing.T) {
	t.Parallel()

	registry, binding, _ := wired(t)
	created := called[schedules.CreateResponse](t, registry, tools.Binding{SiteID: binding.SiteID, Mode: domainagent.ModeAutonomous},
		"schedules", `{"op":"create","name":"Nightly meta","cron":"@daily","steps":["generate_meta"]}`)
	binding.Mode = domainagent.ModeConfirm
	cases := map[string]struct {
		tool string
		args string
		want string
	}{
		"register":       {"sites_manage", `{"op":"create","name":"Second","baseUrl":"https://second.example.com","username":"admin","password":"p"}`, "Register the site Second at https://second.example.com"},
		"settings":       {"sites_manage", `{"op":"update","userAgent":"Agent/2"}`, "settings"},
		"sync":           {"sites_manage", `{"op":"sync","full":true}`, "Walk the whole of"},
		"delete site":    {"sites_manage", `{"op":"delete"}`, "the WordPress site is not touched"},
		"export":         {"imports", `{"op":"export","path":"C:/out/shop.xlsx"}`, "Write the page map and the graph to C:/out/shop.xlsx"},
		"schedule":       {"schedules", `{"op":"create","name":"Weekly links","intervalMinutes":10080}`, "every 10080 minutes"},
		"disarm":         {"schedules", `{"op":"disable","scheduleId":"` + created.Schedule.ID + `"}`, "Disarm the schedule “Nightly meta”"},
		"run a schedule": {"schedules", `{"op":"run_now","scheduleId":"` + created.Schedule.ID + `"}`, "Start the run of the schedule “Nightly meta” now"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			out, err := registry.Call(t.Context(), binding, tc.tool, json.RawMessage(tc.args))
			if err != nil {
				t.Fatalf("%s %s: %v", tc.tool, name, err)
			}
			if asked := waitsForAPerson(t, out); !strings.Contains(asked.Summary, tc.want) {
				t.Errorf("the card says %q, want %q", asked.Summary, tc.want)
			}
		})
	}
}

func TestManyFindingsOfOneCodeAreSettledInOneCall(t *testing.T) {
	t.Parallel()

	registry, binding, seeded := wired(t)
	binding.Mode = domainagent.ModeAutonomous
	first := seedFinding(t, seeded, seeded.site, finding.TechnicalDetected, `{"reason":"shortcode"}`)

	out := callFindings(t, registry, binding, `{"op":"resolve_many","code":"technical_detected","option":"ignore","findingIds":["`+first.ID+`"]}`)
	settled, ok := out.(tools.FindingsResponse)
	if !ok || settled.ResolveMany == nil || len(settled.ResolveMany.Findings) != 1 {
		t.Fatalf("findings resolve_many = %+v, want the one finding settled", out)
	}
	listed, ok := callFindings(t, registry, binding, `{"op":"list"}`).(tools.FindingsResponse)
	if !ok || listed.List == nil || len(listed.List.Items) != 0 {
		t.Errorf("open findings after the bulk settle = %+v, want none", listed.List)
	}
}
