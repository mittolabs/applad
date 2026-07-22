package realtime

import (
	"fmt"
	"time"
)

// EventPublisher is the interface services use to publish realtime events.
// This decouples services from the Hub implementation.
type EventPublisher interface {
	Publish(event Event)
}

// PublishResourceEvent is a convenience function for publishing CRUD events.
func PublishResourceEvent(pub EventPublisher, service, resource, action, projectID, resourceID string, payload interface{}) {
	if pub == nil {
		return
	}
	pub.Publish(Event{
		Type:      fmt.Sprintf("%s.%s.%s", service, resource, action),
		Channel:   fmt.Sprintf("projects.%s.%s.%s", projectID, service, resource),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Payload:   payload,
	})
}
