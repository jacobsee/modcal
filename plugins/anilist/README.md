# AniList Plugin

This plugin fetches anime episode air dates from [AniList](https://anilist.co) for anime you're currently watching. It uses the AniList GraphQL API to retrieve upcoming and recent episodes within a configurable time window.

> [!WARNING]  
> This plugin does not yet handle token refresh. Access tokens expire after 1 year. You will need to manually obtain a new access token using the instructions below when it expires.

## Features

- Fetches episodes from anime you're currently watching on AniList
- Configurable time window (look back and look forward)
- Includes episode details (title, episode number, air time)
- Links to AniList anime pages
- Supports both English and Romaji titles

## Configuration

```yaml
plugins:
  - id: "my-anilist"
    type: "anilist"
    config:
      accessToken: "your-access-token"     # Required: OAuth access token
      daysBack: 7                          # Optional: Days to look back (default: 7)
      daysForward: 14                      # Optional: Days to look forward (default: 14)
```

### Configuration Options

- **accessToken** (required): OAuth access token for your AniList account
- **daysBack** (optional): Number of days in the past to fetch episodes (default: 7)
- **daysForward** (optional): Number of days in the future to fetch episodes (default: 14)

## Setup Instructions

### 1. Create an AniList Application

1. Go to [AniList Developer Settings](https://anilist.co/settings/developer)
2. Sign in to your AniList account
3. Click "Create New Client"
4. Fill in the required fields:
   - **Name**: modcal (or any name you prefer)
   - **Redirect URI**: `https://anilist.co/api/v2/oauth/pin` (for pin-based flow)
5. Save the application
6. Note your **Client ID** and **Client Secret**

### 2. Get an OAuth Access Token

**Important**: The Client ID and Client Secret are just your app credentials. You need an OAuth access token to access your personal AniList data.

#### Recommended: Use the Built-in Helper Tool

The easiest way to get an access token is using the included `anilist-auth` helper:

```bash
# Build the helper (if not already built)
go build -o anilist-auth ./cmd/anilist-auth

# Run it with your credentials
./anilist-auth -client-id YOUR_CLIENT_ID -client-secret YOUR_CLIENT_SECRET
```

#### Alternative: Manual OAuth Flow

If you prefer to do it manually:

1. **Build the authorization URL:**
   ```
   https://anilist.co/api/v2/oauth/authorize?client_id=YOUR_CLIENT_ID&redirect_uri=https://anilist.co/api/v2/oauth/pin&response_type=code
   ```

2. **Visit that URL** in your browser and authorize the application

3. **Copy the authorization code** from the redirect page

4. **Exchange the code for a token:**
   ```bash
   curl -X POST https://anilist.co/api/v2/oauth/token \
     -H "Content-Type: application/x-www-form-urlencoded" \
     -H "Accept: application/json" \
     -d "grant_type=authorization_code" \
     -d "client_id=YOUR_CLIENT_ID" \
     -d "client_secret=YOUR_CLIENT_SECRET" \
     -d "redirect_uri=https://anilist.co/api/v2/oauth/pin" \
     -d "code=YOUR_AUTH_CODE"
   ```

5. The response will include your `access_token`

### 3. Update Configuration

Add your access token to `config.yaml`:

```yaml
plugins:
  - id: "anilist-watching"
    type: "anilist"
    config:
      accessToken: "eyJ0eXAiOi..."
      daysBack: 7
      daysForward: 14

calendars:
  - name: "anime"
    description: "My Anime"
    plugins:
      - "anilist-watching"
```

## Event Format

Each episode becomes a calendar event with:

- **Summary**: `Anime Title - Episode X`
- **Description**: Episode count information (e.g., "Episode 5 of 12")
- **Start Time**: Episode air time (in your local timezone)
- **End Time**: Start time + episode duration (or +24 minutes if duration unknown)
- **URL**: Link to the anime page on AniList
- **Categories**: `anime`, `anilist`

## Notes

- Access tokens from AniList are valid for **1 year** from issuance
- The plugin only fetches anime marked as "Currently Watching" (not "Completed", "Planning", etc.)
- Episodes are only included if they have confirmed airing schedule data
- AniList's GraphQL API has rate limiting - the plugin makes 3 requests per refresh cycle
