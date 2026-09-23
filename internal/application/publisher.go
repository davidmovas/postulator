package application

import "github.com/davidmovas/postulator/internal/application/events"

type Publisher interface {
	Publish(eventType events.Type, payload any) error
}
