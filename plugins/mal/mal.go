package mal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jacobsee/modcal/internal/models"
	"github.com/jacobsee/modcal/internal/plugin"
)

const (
	baseURL = "https://api.myanimelist.net/v2"
)

// MALPlugin fetches anime from MyAnimeList
type MALPlugin struct {
	clientID     string
	accessToken  string
	refreshToken string
	weeksBack    int
	weeksForward int
	client       *http.Client
}

// New creates a new MAL plugin instance
func New() *MALPlugin {
	return &MALPlugin{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (p *MALPlugin) Name() string {
	return "mal"
}

func (p *MALPlugin) Create(config map[string]interface{}) (plugin.Plugin, error) {
	instance := &MALPlugin{
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

	// Optional refresh token
	// TODO: implement token refresh logic
	if refreshToken, ok := config["refreshToken"].(string); ok {
		instance.refreshToken = refreshToken
	}

	// Optional: weeks to look back (default: 1)
	if weeksBack, ok := config["weeksBack"].(int); ok {
		instance.weeksBack = weeksBack
	} else {
		instance.weeksBack = 1
	}

	// Optional: weeks to look forward (default: 2)
	if weeksForward, ok := config["weeksForward"].(int); ok {
		instance.weeksForward = weeksForward
	} else {
		instance.weeksForward = 2
	}

	return instance, nil
}

func (p *MALPlugin) FetchEvents(ctx context.Context) ([]models.Event, error) {
	watching, err := p.getWatchingList(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get watching list: %w", err)
	}

	if len(watching) == 0 {
		return []models.Event{}, nil
	}

	// Generate events for each anime's broadcast schedule
	var events []models.Event
	for _, anime := range watching {
		if anime.Node.Broadcast.DayOfWeek == "" {
			continue // Skip anime without broadcast info
		}

		animeEvents := p.generateEventsForAnime(anime.Node)
		events = append(events, animeEvents...)
	}

	return events, nil
}

func (p *MALPlugin) getWatchingList(ctx context.Context) ([]AnimeListItem, error) {
	url := fmt.Sprintf("%s/users/@me/animelist?status=watching&fields=broadcast,num_episodes&limit=100", baseURL)

	var allItems []AnimeListItem
	for url != "" {
		var response struct {
			Data   []AnimeListItem `json:"data"`
			Paging struct {
				Next string `json:"next"`
			} `json:"paging"`
		}

		if err := p.makeRequest(ctx, url, &response); err != nil {
			return nil, err
		}

		allItems = append(allItems, response.Data...)
		url = response.Paging.Next
	}

	return allItems, nil
}

func (p *MALPlugin) makeRequest(ctx context.Context, url string, result interface{}) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("X-MAL-CLIENT-ID", p.clientID)
	req.Header.Set("Authorization", "Bearer "+p.accessToken)

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	return json.Unmarshal(body, result)
}

func (p *MALPlugin) generateEventsForAnime(anime Anime) []models.Event {
	var events []models.Event

	// Parse broadcast day and time
	weekday := p.parseDayOfWeek(anime.Broadcast.DayOfWeek)
	if weekday == -1 {
		return events
	}

	broadcastTime := p.parseTime(anime.Broadcast.StartTime)

	// Calculate date range
	now := time.Now()
	startDate := now.AddDate(0, 0, -7*p.weeksBack)
	endDate := now.AddDate(0, 0, 7*p.weeksForward)

	// Find all occurrences of this weekday within the range
	current := p.nextWeekday(startDate, weekday)
	for current.Before(endDate) || current.Equal(endDate) {
		// Set the broadcast time (in JST)
		jst := time.FixedZone("JST", 9*60*60)
		airTime := time.Date(
			current.Year(),
			current.Month(),
			current.Day(),
			broadcastTime.Hour(),
			broadcastTime.Minute(),
			0, 0,
			jst,
		)

		// Convert to local time
		airTime = airTime.In(time.Local)

		// Create event
		uid := fmt.Sprintf("mal-%d-%s",
			anime.ID,
			airTime.Format("2006-01-02"),
		)

		summary := fmt.Sprintf("%s - New Episode", anime.Title)

		description := "New episode airs"
		if anime.NumEpisodes > 0 {
			description = fmt.Sprintf("New episode airs (Total: %d episodes)", anime.NumEpisodes)
		}

		// Default 24 minute duration
		endTime := airTime.Add(24 * time.Minute)

		event := models.Event{
			UID:         uid,
			Summary:     summary,
			Description: description,
			StartTime:   airTime,
			EndTime:     endTime,
			AllDay:      false,
			URL:         fmt.Sprintf("https://myanimelist.net/anime/%d", anime.ID),
			Categories:  []string{"anime", "mal"},
		}

		events = append(events, event)

		// Move to next week
		current = current.AddDate(0, 0, 7)
	}

	return events
}

func (p *MALPlugin) parseDayOfWeek(day string) time.Weekday {
	day = strings.ToLower(strings.TrimSpace(day))
	switch day {
	case "sunday":
		return time.Sunday
	case "monday":
		return time.Monday
	case "tuesday":
		return time.Tuesday
	case "wednesday":
		return time.Wednesday
	case "thursday":
		return time.Thursday
	case "friday":
		return time.Friday
	case "saturday":
		return time.Saturday
	default:
		return -1
	}
}

func (p *MALPlugin) parseTime(timeStr string) time.Time {
	// Format: "19:30" or "1:30" (24-hour format)
	parts := strings.Split(timeStr, ":")
	if len(parts) != 2 {
		return time.Time{}
	}

	hour, _ := strconv.Atoi(parts[0])
	minute, _ := strconv.Atoi(parts[1])

	return time.Date(0, 1, 1, hour, minute, 0, 0, time.UTC)
}

func (p *MALPlugin) nextWeekday(from time.Time, weekday time.Weekday) time.Time {
	days := int(weekday) - int(from.Weekday())
	if days < 0 {
		days += 7
	}
	return from.AddDate(0, 0, days)
}

// AnimeListItem represents an item in the user's anime list
type AnimeListItem struct {
	Node Anime `json:"node"`
}

// Anime represents anime information
type Anime struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	NumEpisodes int       `json:"num_episodes"`
	Broadcast   Broadcast `json:"broadcast"`
}

// Broadcast represents broadcast information
type Broadcast struct {
	DayOfWeek string `json:"day_of_the_week"`
	StartTime string `json:"start_time"`
}
