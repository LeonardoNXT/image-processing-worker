package domain

import (
	"encoding/json"
	"time"
)

type Outbox struct {
	ID                  string
	RequestEventID      string
	AggregateID         string
	EventType           string
	AggregateType       string
	Reason              *string
	Payload             json.RawMessage
	OutputTopic         string
	OccurredAt          time.Time
	RetryLimit          int
	RetryCount          int
	Status              string
	ProcessingStartedAt *time.Time
	LastAttemptAt       *time.Time
	PublishedAt         *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}
