package plugin

import (
	"context"

	"github.com/jacobsee/modcal/internal/models"
)

// Plugin is the interface that all calendar plugins must implement
type Plugin interface {
	// Name returns the unique name of this plugin type
	Name() string

	// Create returns a new configured instance of this plugin
	Create(config map[string]interface{}) (Plugin, error)

	// FetchEvents retrieves events from the plugin source
	FetchEvents(ctx context.Context) ([]models.Event, error)
}
