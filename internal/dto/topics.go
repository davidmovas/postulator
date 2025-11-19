package dto

import "github.com/davidmovas/postulator/internal/domain/entities"

type Topic struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	CreatedAt string `json:"createdAt"`
}

func NewTopic(entity *entities.Topic) *Topic {
	t := &Topic{}
	return t.FromEntity(entity)
}

func (d *Topic) ToEntity() (*entities.Topic, error) {
	createdAt, err := StringToTime(d.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &entities.Topic{
		ID:        d.ID,
		Title:     d.Title,
		CreatedAt: createdAt,
	}, nil
}

func (d *Topic) FromEntity(entity *entities.Topic) *Topic {
	d.ID = entity.ID
	d.Title = entity.Title
	d.CreatedAt = TimeToString(entity.CreatedAt)
	return d
}

type BatchResult struct {
	Created       int      `json:"created"`
	Skipped       int      `json:"skipped"`
	SkippedTitles []string `json:"skippedTitles"`
	CreatedTopics []*Topic `json:"createdTopics"`
}

func NewBatchResult(entity *entities.BatchResult) *BatchResult {
	b := &BatchResult{}
	return b.FromEntity(entity)
}

func (d *BatchResult) ToEntity() *entities.BatchResult {
	var createdTopics []*entities.Topic
	for _, topic := range d.CreatedTopics {
		entity, _ := topic.ToEntity()
		createdTopics = append(createdTopics, entity)
	}

	return &entities.BatchResult{
		Created:       d.Created,
		Skipped:       d.Skipped,
		SkippedTitles: d.SkippedTitles,
		CreatedTopics: createdTopics,
	}
}

func (d *BatchResult) FromEntity(entity *entities.BatchResult) *BatchResult {
	d.Created = entity.Created
	d.Skipped = entity.Skipped
	d.SkippedTitles = entity.SkippedTitles

	var createdTopics []*Topic
	for _, topic := range entity.CreatedTopics {
		createdTopics = append(createdTopics, NewTopic(topic))
	}
	d.CreatedTopics = createdTopics

	return d
}

type JobTopicsStatus struct {
	Count  int      `json:"count"`
	Topics []*Topic `json:"topics"`
}

func NewJobTopicsStatus(topics []*entities.Topic, count int) *JobTopicsStatus {
	dtoTopics := make([]*Topic, 0, len(topics))
	for _, t := range topics {
		dtoTopics = append(dtoTopics, NewTopic(t))
	}
	return &JobTopicsStatus{Count: count, Topics: dtoTopics}
}
