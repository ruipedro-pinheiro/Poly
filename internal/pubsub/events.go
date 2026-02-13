package pubsub

// EventType describes the kind of event
type EventType string

const (
	CreatedEvent EventType = "created"
	UpdatedEvent EventType = "updated"
	DeletedEvent EventType = "deleted"
)

// Event wraps a typed payload with its event type
type Event[T any] struct {
	Type    EventType
	Payload T
}
