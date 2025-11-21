package pipeline

type Middleware func(Command) Command
