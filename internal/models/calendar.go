package models

// Calendar represents an aggregated calendar with events from multiple plugins
type Calendar struct {
	Name        string
	Description string
	Events      []Event
}
