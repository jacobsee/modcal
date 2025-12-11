package models

import "time"

// Event represents a calendar event from a plugin
type Event struct {
	UID         string
	Summary     string
	Description string
	Location    string
	StartTime   time.Time
	EndTime     time.Time
	AllDay      bool
	URL         string
	Categories  []string
}
