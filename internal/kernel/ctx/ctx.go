package ctx

import "context"

type Actor string

const (
	ActorUser     Actor = "user"
	ActorAgent    Actor = "agent"
	ActorSchedule Actor = "schedule"
)

func (a Actor) String() string {
	return string(a)
}

type key uint8

const (
	runIDKey key = iota
	conversationIDKey
	actorKey
)

func WithRunID(parent context.Context, runID string) context.Context {
	return context.WithValue(parent, runIDKey, runID)
}

func RunID(c context.Context) (string, bool) {
	value, ok := c.Value(runIDKey).(string)
	return value, ok
}

func WithConversationID(parent context.Context, conversationID string) context.Context {
	return context.WithValue(parent, conversationIDKey, conversationID)
}

func ConversationID(c context.Context) (string, bool) {
	value, ok := c.Value(conversationIDKey).(string)
	return value, ok
}

func WithActor(parent context.Context, actor Actor) context.Context {
	return context.WithValue(parent, actorKey, actor)
}

func ActorFrom(c context.Context) (Actor, bool) {
	value, ok := c.Value(actorKey).(Actor)
	return value, ok
}
