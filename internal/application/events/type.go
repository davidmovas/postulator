package events

type Type string

const (
	GraphChanged          Type = "graph.changed"
	PagesChanged          Type = "pages.changed"
	TemplatesChanged      Type = "templates.changed"
	SitesChanged          Type = "sites.changed"
	SchedulesChanged      Type = "schedules.changed"
	SettingsChanged       Type = "settings.changed"
	AgentDelta            Type = "agent.delta"
	AgentToolStarted      Type = "agent.tool.started"
	AgentToolFinished     Type = "agent.tool.finished"
	AgentConfirmRequested Type = "agent.confirm.requested"
	AgentConfirmResolved  Type = "agent.confirm.resolved"
	AgentDone             Type = "agent.done"
	AgentTitled           Type = "agent.titled"
	AppLocked             Type = "app.locked"
	AppUnlocked           Type = "app.unlocked"

	RunQueued         Type = "run.queued"
	RunStarted        Type = "run.started"
	RunPaused         Type = "run.paused"
	RunResumed        Type = "run.resumed"
	RunCancelled      Type = "run.cancelled"
	RunCompleted      Type = "run.completed"
	RunFailed         Type = "run.failed"
	RunBudgetExceeded Type = "run.budget_exceeded"
	ItemStarted       Type = "item.started"
	ItemDone          Type = "item.done"
	ItemFailed        Type = "item.failed"
	ItemNeedsHuman    Type = "item.needs_human"
	StepStarted       Type = "step.started"
	StepDone          Type = "step.done"
	StepFailed        Type = "step.failed"
	StepRetrying      Type = "step.retrying"
	LLMUsage          Type = "llm.usage"
)
