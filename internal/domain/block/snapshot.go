package block

import "time"

type Side string

const (
	Current Side = "current"
	Base    Side = "base"
)

type Snapshot struct {
	PageID     string
	Side       Side
	Raw        string
	Hash       string
	Kind       Kind
	ObservedAt time.Time
}

type Written struct {
	PageID    string
	BlockHash string
	RunID     string
	WrittenAt time.Time
}
