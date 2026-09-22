package events

import (
	"slices"
	"strings"
)

type Entry struct {
	Type    Type
	Payload any
	Run     bool
}

type Registry struct {
	entries []Entry
	byType  map[Type]Entry
}

func NewRegistry() Registry {
	entries := []Entry{
		{Type: GraphChanged, Payload: GraphChangedPayload{}},
		{Type: PagesChanged, Payload: PagesChangedPayload{}},
		{Type: TemplatesChanged, Payload: TemplatesChangedPayload{}},
		{Type: SitesChanged, Payload: SitesChangedPayload{}},
		{Type: SchedulesChanged, Payload: SchedulesChangedPayload{}},
		{Type: SettingsChanged, Payload: SettingsChangedPayload{}},
		{Type: AgentDelta, Payload: AgentDeltaPayload{}},
		{Type: AgentToolStarted, Payload: AgentToolStartedPayload{}},
		{Type: AgentToolFinished, Payload: AgentToolFinishedPayload{}},
		{Type: AgentConfirmRequested, Payload: AgentConfirmRequestedPayload{}},
		{Type: AgentConfirmResolved, Payload: AgentConfirmResolvedPayload{}},
		{Type: AgentUsage, Payload: AgentUsagePayload{}},
		{Type: AgentWaiting, Payload: AgentWaitingPayload{}},
		{Type: AgentDone, Payload: AgentDonePayload{}},
		{Type: AgentTitled, Payload: AgentTitledPayload{}},
		{Type: AppLocked, Payload: AppLockedPayload{}},
		{Type: AppUnlocked, Payload: AppUnlockedPayload{}},
		{Type: FilesDropped, Payload: FilesDroppedPayload{}},
		{Type: RunQueued, Payload: RunQueuedPayload{}, Run: true},
		{Type: RunStarted, Payload: RunStartedPayload{}, Run: true},
		{Type: RunPaused, Payload: RunPausedPayload{}, Run: true},
		{Type: RunResumed, Payload: RunResumedPayload{}, Run: true},
		{Type: RunCancelled, Payload: RunCancelledPayload{}, Run: true},
		{Type: RunCompleted, Payload: RunCompletedPayload{}, Run: true},
		{Type: RunFailed, Payload: RunFailedPayload{}, Run: true},
		{Type: RunBudgetExceeded, Payload: RunBudgetExceededPayload{}, Run: true},
		{Type: ItemStarted, Payload: ItemStartedPayload{}, Run: true},
		{Type: ItemDone, Payload: ItemDonePayload{}, Run: true},
		{Type: ItemFailed, Payload: ItemFailedPayload{}, Run: true},
		{Type: ItemNeedsHuman, Payload: ItemNeedsHumanPayload{}, Run: true},
		{Type: StepStarted, Payload: StepStartedPayload{}, Run: true},
		{Type: StepDone, Payload: StepDonePayload{}, Run: true},
		{Type: StepFailed, Payload: StepFailedPayload{}, Run: true},
		{Type: StepRetrying, Payload: StepRetryingPayload{}, Run: true},
		{Type: LLMUsage, Payload: LLMUsagePayload{}},
	}

	slices.SortFunc(entries, func(a, b Entry) int {
		return strings.Compare(string(a.Type), string(b.Type))
	})

	byType := make(map[Type]Entry, len(entries))
	for _, entry := range entries {
		byType[entry.Type] = entry
	}
	return Registry{entries: entries, byType: byType}
}

func (r Registry) Entries() []Entry {
	return slices.Clone(r.entries)
}

func (r Registry) Lookup(eventType Type) (Entry, bool) {
	entry, ok := r.byType[eventType]
	return entry, ok
}
