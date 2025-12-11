package trakt

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/jacobsee/modcal/internal/models"
	"github.com/jacobsee/modcal/internal/plugin"
)

const (
	baseURL    = "https://api.trakt.tv"
	apiVersion = "2"
)

// TraktPlugin fetches TV show episodes from Trakt
type TraktPlugin struct {
	clientID    string
	accessToken string
	daysBack    int
	daysForward int
	client      *http.Client
}

// New creates a new Trakt plugin instance
func New() *TraktPlugin {
	return &TraktPlugin{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (p *TraktPlugin) Name() string {
	return "trakt"
}

func (p *TraktPlugin) Create(config map[string]interface{}) (plugin.Plugin, error) {
	instance := &TraktPlugin{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	clientID, ok := config["clientId"].(string)
	if !ok || clientID == "" {
		return nil, fmt.Errorf("clientId is required")
	}
	instance.clientID = clientID

	accessToken, ok := config["accessToken"].(string)
	if !ok || accessToken == "" {
		return nil, fmt.Errorf("accessToken is required")
	}
	instance.accessToken = accessToken

	// Optional: days to look back (default: 7)
	if daysBack, ok := config["daysBack"].(int); ok {
		instance.daysBack = daysBack
	} else {
		instance.daysBack = 7
	}

	// Optional: days to look forward (default: 14)
	if daysForward, ok := config["daysForward"].(int); ok {
		instance.daysForward = daysForward
	} else {
		instance.daysForward = 14
	}

	return instance, nil
}

func (p *TraktPlugin) FetchEvents(ctx context.Context) ([]models.Event, error) {
	startDate := time.Now().AddDate(0, 0, -p.daysBack)
	totalDays := p.daysBack + p.daysForward

	startDateStr := startDate.Format("2006-01-02")

	url := fmt.Sprintf("%s/calendars/my/shows/%s/%d", baseURL, startDateStr, totalDays)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("trakt-api-version", apiVersion)
	req.Header.Set("trakt-api-key", p.clientID)
	req.Header.Set("Authorization", "Bearer "+p.accessToken)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch calendar: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("trakt API returned status %d: %s", resp.StatusCode, string(body))
	}

	var calendarItems []CalendarItem
	if err := json.NewDecoder(resp.Body).Decode(&calendarItems); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return p.convertToEvents(calendarItems), nil
}

func (p *TraktPlugin) convertToEvents(items []CalendarItem) []models.Event {
	var events []models.Event

	for _, item := range items {
		airTime, err := time.Parse(time.RFC3339, item.FirstAired)
		if err != nil {
			continue
		}

		uid := fmt.Sprintf("trakt-%s-s%02de%02d-%d",
			item.Show.IDs.Slug,
			item.Episode.Season,
			item.Episode.Number,
			airTime.Unix(),
		)

		summary := fmt.Sprintf("%s - S%02dE%02d",
			item.Show.Title,
			item.Episode.Season,
			item.Episode.Number,
		)
		if item.Episode.Title != "" {
			summary += fmt.Sprintf(": %s", item.Episode.Title)
		}

		description := ""
		if item.Episode.Overview != "" {
			description = item.Episode.Overview
		}
		if item.Show.Network != "" {
			if description != "" {
				description += "\n\n"
			}
			description += fmt.Sprintf("Network: %s", item.Show.Network)
		}

		endTime := airTime
		if item.Show.Runtime > 0 {
			endTime = airTime.Add(time.Duration(item.Show.Runtime) * time.Minute)
		} else {
			endTime = airTime.Add(1 * time.Hour) // Default to 1 hour
		}

		event := models.Event{
			UID:         uid,
			Summary:     summary,
			Description: description,
			StartTime:   airTime,
			EndTime:     endTime,
			AllDay:      false,
			Categories:  []string{"tv", "trakt"},
		}

		if item.Show.IDs.Slug != "" {
			event.URL = fmt.Sprintf("https://trakt.tv/shows/%s/seasons/%d/episodes/%d",
				item.Show.IDs.Slug,
				item.Episode.Season,
				item.Episode.Number,
			)
		}

		events = append(events, event)
	}

	return events
}

// CalendarItem represents an item in the Trakt calendar response
type CalendarItem struct {
	FirstAired string  `json:"first_aired"`
	Episode    Episode `json:"episode"`
	Show       Show    `json:"show"`
}

// Episode represents episode information
type Episode struct {
	Season   int    `json:"season"`
	Number   int    `json:"number"`
	Title    string `json:"title"`
	Overview string `json:"overview"`
}

// Show represents show information
type Show struct {
	Title   string  `json:"title"`
	Year    int     `json:"year"`
	Network string  `json:"network"`
	Runtime int     `json:"runtime"`
	IDs     ShowIDs `json:"ids"`
}

// ShowIDs contains various show identifiers
type ShowIDs struct {
	Trakt int    `json:"trakt"`
	Slug  string `json:"slug"`
	IMDB  string `json:"imdb"`
	TMDB  int    `json:"tmdb"`
}
