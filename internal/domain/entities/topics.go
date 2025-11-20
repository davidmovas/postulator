package entities

import "time"

type Topic struct {
	ID        int64
	Title     string
	CreatedAt time.Time
}

type BatchResult struct {
	Created       int
	Skipped       int
	SkippedTitles []string
	CreatedTopics []*Topic
}

type ImportAssignResult struct {
	TotalProcessed int
	TotalAdded     int
	TotalSkipped   int
	Added          []string
	Skipped        []string
}
