package calendar

import (
	"context"
	"fmt"
	"sync"

	"github.com/jacobsee/modcal/internal/models"
	"github.com/jacobsee/modcal/internal/plugin"
)

// Manager handles calendar aggregation and event fetching
type Manager struct {
	mu            sync.RWMutex
	calendars     map[string]*CalendarDefinition
	eventCache    map[string][]models.Event
	pluginManager *PluginManager
}

// CalendarDefinition defines a calendar with its associated plugins
type CalendarDefinition struct {
	Name        string
	Description string
	PluginIDs   []string
}

// PluginManager manages plugin instances
type PluginManager struct {
	mu        sync.RWMutex
	instances map[string]plugin.Plugin
}

// NewPluginManager creates a new plugin manager
func NewPluginManager() *PluginManager {
	return &PluginManager{
		instances: make(map[string]plugin.Plugin),
	}
}

// AddInstance adds a plugin instance with a specific ID
func (pm *PluginManager) AddInstance(id string, p plugin.Plugin) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.instances[id] = p
}

// GetInstance retrieves a plugin instance by ID
func (pm *PluginManager) GetInstance(id string) (plugin.Plugin, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	p, ok := pm.instances[id]
	return p, ok
}

// NewManager creates a new calendar manager
func NewManager(pm *PluginManager) *Manager {
	return &Manager{
		calendars:     make(map[string]*CalendarDefinition),
		eventCache:    make(map[string][]models.Event),
		pluginManager: pm,
	}
}

// AddCalendar registers a calendar definition
func (m *Manager) AddCalendar(cal *CalendarDefinition) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calendars[cal.Name] = cal
}

// GetCalendar retrieves a calendar with its current events
func (m *Manager) GetCalendar(name string) (*models.Calendar, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	calDef, exists := m.calendars[name]
	if !exists {
		return nil, fmt.Errorf("calendar %s not found", name)
	}

	var allEvents []models.Event
	for _, pluginID := range calDef.PluginIDs {
		if events, ok := m.eventCache[pluginID]; ok {
			allEvents = append(allEvents, events...)
		}
	}

	return &models.Calendar{
		Name:        calDef.Name,
		Description: calDef.Description,
		Events:      allEvents,
	}, nil
}

// RefreshEvents fetches events from all plugins
func (m *Manager) RefreshEvents(ctx context.Context) error {
	m.pluginManager.mu.RLock()
	instances := make(map[string]plugin.Plugin, len(m.pluginManager.instances))
	for id, p := range m.pluginManager.instances {
		instances[id] = p
	}
	m.pluginManager.mu.RUnlock()

	var wg sync.WaitGroup
	errChan := make(chan error, len(instances))

	for id, p := range instances {
		wg.Add(1)
		go func(pluginID string, plug plugin.Plugin) {
			defer wg.Done()

			events, err := plug.FetchEvents(ctx)
			if err != nil {
				errChan <- fmt.Errorf("plugin %s: %w", pluginID, err)
				return
			}

			m.mu.Lock()
			m.eventCache[pluginID] = events
			m.mu.Unlock()
		}(id, p)
	}

	wg.Wait()
	close(errChan)

	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return errs[0]
	}

	return nil
}

// ListCalendars returns all calendar names
func (m *Manager) ListCalendars() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.calendars))
	for name := range m.calendars {
		names = append(names, name)
	}
	return names
}
