package agent_test

import (
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	agentapp "github.com/davidmovas/postulator/internal/application/agent"
	"github.com/davidmovas/postulator/internal/application/applicationtest"
	domainagent "github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestAConfirmedActionRunsThroughTheGuardChain(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		allowed    []string
		wantAction domainagent.ActionStatus
		wantCall   domainagent.CallStatus
		wantGone   bool
	}{
		{
			name:       "an open tool runs and the ledger records it",
			wantAction: domainagent.ActionExecuted,
			wantCall:   domainagent.CallOK,
			wantGone:   true,
		},
		{
			name:       "a tool outside the allow list is refused and still audited",
			allowed:    []string{"sites_list"},
			wantAction: domainagent.ActionFailed,
			wantCall:   domainagent.CallDenied,
			wantGone:   false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			proposer := newHarness(t)
			doomed := sqlitetest.Page(t, proposer.store, proposer.siteID, "/coffee/doomed/")
			conversation := proposer.conversation(t, domainagent.ModeConfirm)
			proposer.send(t, conversation, `TOOL:pages_delete{"id":"`+doomed.ID+`"}`+"\nFAKE: waiting for you")

			pending, err := proposer.service.ListPendingActions(t.Context(), agentapp.ListPendingActionsRequest{
				ConversationID: conversation, Status: string(domainagent.ActionPending),
			})
			if err != nil || len(pending.Items) != 1 {
				t.Fatalf("the pending actions are %+v, %v", pending, err)
			}

			settler := build(t, proposer.store, proposer.model, &applicationtest.Recorder{}, tc.allowed...)
			settler.siteID = proposer.siteID

			confirmed, err := settler.service.Confirm(t.Context(), agentapp.ConfirmRequest{
				ActionID: pending.Items[0].ID, Approve: true,
			})
			if err != nil {
				t.Fatalf("Confirm: %v", err)
			}
			if confirmed.Action.Status != string(tc.wantAction) {
				t.Fatalf("the action settled as %+v, want %s", confirmed.Action, tc.wantAction)
			}

			_, getErr := proposer.pages.Get(t.Context(), doomed.ID)
			if gone := errors.IsCode(getErr, errors.NotFound); gone != tc.wantGone {
				t.Fatalf("the page gone = %v, want %v (%v)", gone, tc.wantGone, getErr)
			}

			recorded, err := sqlite.NewToolCallRepo(proposer.store).ByConversation(t.Context(), conversation)
			if err != nil {
				t.Fatalf("read the tool call ledger: %v", err)
			}

			replayed := ledgerRow(recorded, pending.Items[0].ID)
			if replayed == nil {
				t.Fatalf("the replay wrote no audit row; the ledger holds %+v", recorded)
			}
			if replayed.Tool != "pages_delete" || replayed.Status != tc.wantCall {
				t.Fatalf("the audit row is %+v, want pages_delete as %s", replayed, tc.wantCall)
			}
			if !strings.Contains(string(replayed.Args), doomed.ID) {
				t.Fatalf("the audit row lost the arguments it replayed: %s", replayed.Args)
			}
			if tc.wantCall == domainagent.CallDenied &&
				!strings.Contains(replayed.Error, "not open to this conversation") {
				t.Fatalf("the denial reads %q", replayed.Error)
			}
		})
	}
}

func TestAnUndeliverableResumeIsWrittenOnTheAction(t *testing.T) {
	t.Parallel()

	proposer := newHarness(t)
	doomed := sqlitetest.Page(t, proposer.store, proposer.siteID, "/coffee/stranded/")
	conversation := proposer.conversation(t, domainagent.ModeConfirm)
	proposer.send(t, conversation, `TOOL:pages_delete{"id":"`+doomed.ID+`"}`+"\nFAKE: waiting for you")

	pending, err := proposer.service.ListPendingActions(t.Context(), agentapp.ListPendingActionsRequest{
		ConversationID: conversation, Status: string(domainagent.ActionPending),
	})
	if err != nil || len(pending.Items) != 1 {
		t.Fatalf("the pending actions are %+v, %v", pending, err)
	}

	settler := build(t, proposer.store, proposer.model, &applicationtest.Recorder{})
	settler.siteID = proposer.siteID
	settler.service.Close()

	confirmed, err := settler.service.Confirm(t.Context(), agentapp.ConfirmRequest{
		ActionID: pending.Items[0].ID, Approve: true,
	})
	if err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if confirmed.Action.Status != string(domainagent.ActionExecuted) {
		t.Fatalf("the action settled as %+v, and the tool did run", confirmed.Action)
	}
	if !strings.Contains(confirmed.Action.Error, "could not be told") {
		t.Fatalf("the action says nothing about the undelivered result: %+v", confirmed.Action)
	}

	if _, getErr := proposer.pages.Get(t.Context(), doomed.ID); !errors.IsCode(getErr, errors.NotFound) {
		t.Fatalf("the approved tool did not run: %v", getErr)
	}
}

func ledgerRow(recorded []domainagent.ToolCall, callID string) *domainagent.ToolCall {
	for i := range recorded {
		if recorded[i].CallID == callID {
			return &recorded[i]
		}
	}
	return nil
}
