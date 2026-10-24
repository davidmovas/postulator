package field

type Change struct {
	Key        Key    `json:"key"`
	Before     string `json:"before"`
	BeforeHash string `json:"beforeHash"`
	After      string `json:"after"`
	AfterHash  string `json:"afterHash"`
}

func NewChange(key Key, before, after string) Change {
	return Change{Key: key, Before: before, BeforeHash: Hash(before), After: after, AfterHash: Hash(after)}
}

func (c Change) Moves() bool {
	return c.BeforeHash != c.AfterHash
}
