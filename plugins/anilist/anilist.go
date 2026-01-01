package anilist

import (
	"bytes"
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
	graphqlURL = "https://graphql.anilist.co"
)

// AniListPlugin fetches episode release info from AniList
type AniListPlugin struct {
	accessToken string
	daysBack    int
	daysForward int
	client      *http.Client
}

// New creates a new AniList plugin instance
func New() *AniListPlugin {
	return &AniListPlugin{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (p *AniListPlugin) Name() string {
	return "anilist"
}

func (p *AniListPlugin) Create(config map[string]interface{}) (plugin.Plugin, error) {
	instance := &AniListPlugin{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

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

func (p *AniListPlugin) FetchEvents(ctx context.Context) ([]models.Event, error) {
	userID, err := p.getAuthenticatedUserID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get user ID: %w", err)
	}

	watchingList, err := p.getWatchingList(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get watching list: %w", err)
	}

	if len(watchingList) == 0 {
		return []models.Event{}, nil
	}

	now := time.Now()
	startTime := now.AddDate(0, 0, -p.daysBack).Unix()
	endTime := now.AddDate(0, 0, p.daysForward).Unix()

	schedules, err := p.getAiringSchedules(ctx, watchingList, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get airing schedules: %w", err)
	}

	return p.convertToEvents(schedules), nil
}

func (p *AniListPlugin) getAuthenticatedUserID(ctx context.Context) (int, error) {
	query := `
	query {
		Viewer {
			id
			name
		}
	}
	`

	var response struct {
		Data struct {
			Viewer struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			} `json:"Viewer"`
		} `json:"data"`
	}

	if err := p.executeQuery(ctx, query, nil, &response); err != nil {
		return 0, err
	}

	return response.Data.Viewer.ID, nil
}

func (p *AniListPlugin) getWatchingList(ctx context.Context, userID int) ([]int, error) {
	query := `
	query ($userId: Int, $type: MediaType, $status: MediaListStatus) {
		MediaListCollection(userId: $userId, type: $type, status: $status) {
			lists {
				entries {
					media {
						id
					}
				}
			}
		}
	}
	`

	type mediaListResponse struct {
		Data struct {
			MediaListCollection struct {
				Lists []struct {
					Entries []struct {
						Media struct {
							ID int `json:"id"`
						} `json:"media"`
					} `json:"entries"`
				} `json:"lists"`
			} `json:"MediaListCollection"`
		} `json:"data"`
	}

	currentVariables := map[string]interface{}{
		"userId": userID,
		"type":   "ANIME",
		"status": "CURRENT",
	}

	var currentResponse mediaListResponse
	if err := p.executeQuery(ctx, query, currentVariables, &currentResponse); err != nil {
		return nil, err
	}

	planningVariables := map[string]interface{}{
		"userId": userID,
		"type":   "ANIME",
		"status": "PLANNING",
	}

	var planningResponse mediaListResponse
	if err := p.executeQuery(ctx, query, planningVariables, &planningResponse); err != nil {
		return nil, err
	}

	mediaIDMap := make(map[int]bool)
	var mediaIDs []int

	for _, list := range currentResponse.Data.MediaListCollection.Lists {
		for _, entry := range list.Entries {
			if !mediaIDMap[entry.Media.ID] {
				mediaIDMap[entry.Media.ID] = true
				mediaIDs = append(mediaIDs, entry.Media.ID)
			}
		}
	}

	for _, list := range planningResponse.Data.MediaListCollection.Lists {
		for _, entry := range list.Entries {
			if !mediaIDMap[entry.Media.ID] {
				mediaIDMap[entry.Media.ID] = true
				mediaIDs = append(mediaIDs, entry.Media.ID)
			}
		}
	}

	return mediaIDs, nil
}

func (p *AniListPlugin) getAiringSchedules(ctx context.Context, mediaIDs []int, startTime, endTime int64) ([]AiringSchedule, error) {
	query := `
	query ($mediaIds: [Int], $airingAt_greater: Int, $airingAt_lesser: Int) {
		Page(page: 1, perPage: 50) {
			airingSchedules(mediaId_in: $mediaIds, airingAt_greater: $airingAt_greater, airingAt_lesser: $airingAt_lesser, sort: TIME) {
				id
				airingAt
				episode
				mediaId
				media {
					id
					title {
						romaji
						english
						native
					}
					episodes
					duration
					coverImage {
						large
					}
					siteUrl
				}
			}
		}
	}
	`

	variables := map[string]interface{}{
		"mediaIds":          mediaIDs,
		"airingAt_greater":  startTime,
		"airingAt_lesser":   endTime,
	}

	var response struct {
		Data struct {
			Page struct {
				AiringSchedules []AiringSchedule `json:"airingSchedules"`
			} `json:"Page"`
		} `json:"data"`
	}

	if err := p.executeQuery(ctx, query, variables, &response); err != nil {
		return nil, err
	}

	return response.Data.Page.AiringSchedules, nil
}

func (p *AniListPlugin) executeQuery(ctx context.Context, query string, variables map[string]interface{}, result interface{}) error {
	requestBody := map[string]interface{}{
		"query": query,
	}
	if variables != nil {
		requestBody["variables"] = variables
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", graphqlURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if p.accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+p.accessToken)
	}

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

func (p *AniListPlugin) convertToEvents(schedules []AiringSchedule) []models.Event {
	var events []models.Event

	for _, schedule := range schedules {
		airTime := time.Unix(int64(schedule.AiringAt), 0)

		title := schedule.Media.Title.Romaji
		if schedule.Media.Title.English != "" {
			title = schedule.Media.Title.English
		}

		summary := fmt.Sprintf("%s - Episode %d", title, schedule.Episode)

		// Calculate end time based on duration
		endTime := airTime
		if schedule.Media.Duration > 0 {
			endTime = airTime.Add(time.Duration(schedule.Media.Duration) * time.Minute)
		} else {
			endTime = airTime.Add(24 * time.Minute) // Default anime episode length, not precise
		}

		uid := fmt.Sprintf("anilist-%d-ep%d-%d",
			schedule.MediaID,
			schedule.Episode,
			schedule.AiringAt,
		)

		description := ""
		if schedule.Media.Episodes > 0 {
			description = fmt.Sprintf("Episode %d of %d", schedule.Episode, schedule.Media.Episodes)
		}

		event := models.Event{
			UID:         uid,
			Summary:     summary,
			Description: description,
			StartTime:   airTime,
			EndTime:     endTime,
			AllDay:      false,
			URL:         schedule.Media.SiteURL,
			Categories:  []string{"anime", "anilist"},
		}

		events = append(events, event)
	}

	return events
}

// AiringSchedule represents an airing schedule entry
type AiringSchedule struct {
	ID        int   `json:"id"`
	AiringAt  int   `json:"airingAt"`
	Episode   int   `json:"episode"`
	MediaID   int   `json:"mediaId"`
	Media     Media `json:"media"`
}

// Media represents anime information
type Media struct {
	ID       int       `json:"id"`
	Title    Title     `json:"title"`
	Episodes int       `json:"episodes"`
	Duration int       `json:"duration"`
	SiteURL  string    `json:"siteUrl"`
}

// Title represents anime titles in different languages
type Title struct {
	Romaji  string `json:"romaji"`
	English string `json:"english"`
	Native  string `json:"native"`
}
