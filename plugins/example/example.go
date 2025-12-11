package example

import (
	"context"
	"fmt"
	"time"

	"github.com/jacobsee/modcal/internal/models"
	"github.com/jacobsee/modcal/internal/plugin"
)

// ExamplePlugin demonstrates a simple plugin implementation
type ExamplePlugin struct {
	message string
}

// New creates a new example plugin instance
func New() *ExamplePlugin {
	return &ExamplePlugin{}
}

func (p *ExamplePlugin) Name() string {
	return "example"
}

func (p *ExamplePlugin) Create(config map[string]interface{}) (plugin.Plugin, error) {
	instance := &ExamplePlugin{}
	if msg, ok := config["message"].(string); ok {
		instance.message = msg
	} else {
		instance.message = "Default example event"
	}
	return instance, nil
}

func (p *ExamplePlugin) FetchEvents(ctx context.Context) ([]models.Event, error) {
	now := time.Now()

	events := []models.Event{
		{
			UID:         fmt.Sprintf("example-1-%d", now.Unix()),
			Summary:     p.message,
			Description: "This is an example event from the example plugin",
			Location:    "Example Location",
			StartTime:   now.Add(24 * time.Hour),
			EndTime:     now.Add(25 * time.Hour),
			AllDay:      false,
			Categories:  []string{"example"},
		},
		{
			UID:         fmt.Sprintf("example-2-%d", now.Unix()),
			Summary:     "Another Example Event",
			Description: "This is another example event",
			StartTime:   now.Add(48 * time.Hour),
			EndTime:     now.Add(48 * time.Hour),
			AllDay:      true,
			Categories:  []string{"example", "all-day"},
		},
	}

	return events, nil
}
