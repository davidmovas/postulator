package agent

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	domainagent "github.com/davidmovas/postulator/internal/domain/agent"
)

func TestActionViewMasksTheCredentialItWillReplay(t *testing.T) {
	t.Parallel()

	stamp := time.Date(2026, 9, 18, 9, 0, 0, 0, time.UTC)
	view := actionView(domainagent.PendingAction{
		ID: "action-1", ConversationID: "conversation-1", Tool: "models_set_provider_key",
		Args:    json.RawMessage(`{"provider":"openai","apiKey":"sk-live-secret","site":{"password":"hunter2"}}`),
		Summary: "Store the api key of a model provider",
		Status:  domainagent.ActionPending, CreatedAt: stamp, UpdatedAt: stamp,
	})

	if strings.Contains(string(view.Args), "sk-live-secret") || strings.Contains(string(view.Args), "hunter2") {
		t.Fatalf("the pending action view carries the credential: %s", view.Args)
	}
	if !strings.Contains(string(view.Args), `"provider":"openai"`) {
		t.Fatalf("the view dropped what a human needs to read: %s", view.Args)
	}
	if view.ID != "action-1" || view.Tool != "models_set_provider_key" ||
		view.Status != string(domainagent.ActionPending) {
		t.Fatalf("the view is %+v", view)
	}
}
