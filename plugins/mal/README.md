## MyAnimeList Plugin

This plugin fetches anime broadcast schedules from [MyAnimeList](https://myanimelist.net) for anime you're currently watching. It uses the MAL API v2 to retrieve your watching list and generates calendar events based on weekly broadcast times.

> [!WARNING]  
> This plugin does not yet handle token refresh. Access tokens expire after 31 days. You will need to manually obtain a new access token using the instructions below when it expires. This is a pain and I will get to it if I start to use MAL more often.

## Features

- Fetches anime from your "Watching" list on MyAnimeList
- Generates weekly recurring events based on broadcast schedule
- Configurable time window (look back and look forward in weeks)
- Includes anime details (title, episode count)
- Links to MyAnimeList anime pages
- Automatically converts JST broadcast times to your local timezone

## Configuration

```yaml
plugins:
  - id: "my-mal"
    type: "mal"
    config:
      clientId: "your-client-id"           # Required: Your MAL API client ID
      accessToken: "your-access-token"     # Required: OAuth access token
      refreshToken: "your-refresh-token"   # Optional but recommended, however not yet implemented
      weeksBack: 1                         # Optional: Weeks to look back (default: 1)
      weeksForward: 2                      # Optional: Weeks to look forward (default: 2)
```

### Configuration Options

- **clientId** (required): Your MyAnimeList API client ID
- **accessToken** (required): OAuth access token for your MAL account
- **refreshToken** (optional): OAuth refresh token to renew expired access tokens
- **weeksBack** (optional): Number of weeks in the past to generate events (default: 1)
- **weeksForward** (optional): Number of weeks in the future to generate events (default: 2)

## Setup Instructions

### 1. Create a MyAnimeList API Application

1. Go to [MyAnimeList API Config](https://myanimelist.net/apiconfig)
2. Sign in to your MyAnimeList account
3. Click "Create ID"
4. Fill in the required fields:
   - **App Name**: modcal (or any name you prefer)
   - **App Type**: web
   - **App Description**: Personal calendar integration
   - **App Redirect URL**: `http://localhost` (not actually used but must be set to this value)
   - **Homepage URL**: `http://localhost`
   - **Commercial / Non-Commercial**: Non-Commercial
5. Submit the application
6. Note your **Client ID** and **Client Secret**

### 2. Get an OAuth Access Token

**Important**: MyAnimeList uses OAuth2 with PKCE (Proof Key for Code Exchange) for security. You need to go through an authorization flow to get an access token.

#### Recommended: Use the Built-in Helper Tool

The easiest way to get an access token is using the included `mal-auth` helper:

```bash
# Build the helper (if not already built)
go build -o mal-auth ./cmd/mal-auth

# Run it with your credentials
# IMPORTANT: The redirect-uri must match what you set in your MAL app config
# If you used something other than "http://localhost", specify it with -redirect-uri
./mal-auth -client-id YOUR_CLIENT_ID -client-secret YOUR_CLIENT_SECRET

# If you used a different redirect URI in your app config:
./mal-auth -client-id YOUR_CLIENT_ID -client-secret YOUR_CLIENT_SECRET -redirect-uri "YOUR_REDIRECT_URI"
```

### 3. Update Configuration

Add your credentials to `config.yaml`:

```yaml
plugins:
  - id: "mal-watching"
    type: "mal"
    config:
      clientId: "abc123..."
      accessToken: "eyJhbGc..."
      refreshToken: "def502..."
      weeksBack: 1
      weeksForward: 2

calendars:
  - name: "anime-mal"
    description: "My Anime"
    plugins:
      - "mal-watching"
```

## How It Works

The plugin performs the following steps:

1. **Fetches your watching list** from MyAnimeList with broadcast information
2. **Parses broadcast schedules** - MAL provides day of week and time (e.g., "thursday 19:30 JST")
3. **Generates weekly events** - Creates recurring events for each broadcast time window
4. **Converts to local time** - Broadcast times are in JST and converted to your timezone

### Important Notes on Broadcast Schedules

Unlike AniList or Trakt, MyAnimeList doesn't provide specific episode air dates with episode numbers. Instead, it only provides:
- Day of week (e.g., "thursday")
- Start time (e.g., "19:30" in JST)

This means the plugin generates **weekly recurring events** for "New Episode" rather than specific episode numbers. Each event represents the weekly broadcast time slot for that anime.

## Event Format

Each broadcast slot becomes a calendar event with:

- **Summary**: `Anime Title - New Episode`
- **Description**: Episode count information (e.g., "New episode airs (Total: 12 episodes)")
- **Start Time**: Broadcast time (converted from JST to your local timezone)
- **End Time**: Start time + 24 minutes (default anime episode length)
- **URL**: Link to the anime page on MyAnimeList
- **Categories**: `anime`, `mal`

## Token Refresh

Access tokens from MyAnimeList expire after 31 days. To refresh:

```bash
curl -X POST https://myanimelist.net/v1/oauth2/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "client_id=YOUR_CLIENT_ID" \
  -d "client_secret=YOUR_CLIENT_SECRET" \
  -d "grant_type=refresh_token" \
  -d "refresh_token=YOUR_REFRESH_TOKEN"
```

The response will include a new `access_token` and `refresh_token`. Update your config with the new tokens.

## Notes

- The plugin respects the configured refresh interval from the main config
- Only anime marked as "Watching" are included
- Broadcast information must be available in MAL's database
- Times are automatically converted from JST to your local timezone
- The plugin generates events for N weeks back and N weeks forward from today
