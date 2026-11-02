package site

type Clearing struct {
	Model   bool
	Graph   bool
	History bool
}

func (c Clearing) Normalized() Clearing {
	if c.Model {
		c.History = true
	}
	return c
}

func (c Clearing) Empty() bool {
	return !c.Model && !c.Graph && !c.History
}

type Held struct {
	Pages      int
	Findings   int
	Entities   int
	Edges      int
	Runs       int
	Unfinished int
	Chats      int
}
