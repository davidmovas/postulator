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

const actorKey key = iota

func WithActor(parent context.Context, actor Actor) context.Context {
	return context.WithValue(parent, actorKey, actor)
}

func ActorFrom(c context.Context) (Actor, bool) {
	value, ok := c.Value(actorKey).(Actor)
	return value, ok
}
