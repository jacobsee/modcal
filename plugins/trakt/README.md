# Trakt Plugin

This plugin fetches TV show episode air dates from [Trakt.tv](https://trakt.tv) for shows you're watching. It uses the Trakt calendar API to retrieve upcoming and recent episodes within a configurable time window.

## Features

- Fetches episodes from shows you're watching on Trakt
- Configurable time window (look back and look forward)
- Includes episode details (title, description, air time)
- Links to Trakt.tv episode pages
- Automatically calculates end times based on show runtime

## Configuration

```yaml
plugins:
  - id: "my-trakt"
    type: "trakt"
    config:
      clientId: "your-client-id"           # Required: Your Trakt API client ID
      accessToken: "your-access-token"     # Required: OAuth access token
      daysBack: 7                          # Optional: Days to look back (default: 7)
      daysForward: 14                      # Optional: Days to look forward (default: 14)
```

### Configuration Options

- **clientId** (required): Your Trakt API client ID
- **accessToken** (required): OAuth access token for your Trakt account
- **daysBack** (optional): Number of days in the past to fetch episodes (default: 7)
- **daysForward** (optional): Number of days in the future to fetch episodes (default: 14)

## Setup Instructions

### 1. Create a Trakt Application

1. Go to [Trakt API Applications](https://trakt.tv/oauth/applications)
2. Sign in to your Trakt account
3. Click "New Application"
4. Fill in the required fields:
   - **Name**: modcal (or any name you prefer)
   - **Redirect URI**: `urn:ietf:wg:oauth:2.0:oob`
5. Save the application
6. Note your **Client ID** and **Client Secret**

### 2. Get an OAuth Access Token

**Important**: The Client ID and Client Secret are just your app credentials. You need an OAuth access token to access your personal Trakt data.

#### Recommended: Use the Built-in Helper Tool

The easiest way to get an access token is using the included `trakt-auth` helper:

```bash
# Build the helper (if not already built)
go build -o trakt-auth ./cmd/trakt-auth

# Run it with your credentials
./trakt-auth -client-id YOUR_CLIENT_ID -client-secret YOUR_CLIENT_SECRET
```

#### Alternative: Manual Device Flow

If you prefer to do it manually:

1. **Get a device code:**
   ```bash
   curl -X POST https://api.trakt.tv/oauth/device/code \
     -H "Content-Type: application/json" \
     -d '{"client_id": "YOUR_CLIENT_ID"}'
   ```

2. **Visit the verification URL** and enter the user code from the response

3. **Poll for the token:**
   ```bash
   curl -X POST https://api.trakt.tv/oauth/device/token \
     -H "Content-Type: application/json" \
     -d '{
       "code": "DEVICE_CODE_FROM_STEP_1",
       "client_id": "YOUR_CLIENT_ID",
       "client_secret": "YOUR_CLIENT_SECRET"
     }'
   ```

4. Repeat step 3 until you get an `access_token` instead of `authorization_pending`

### 3. Update Configuration

Add your credentials to `config.yaml`:

```yaml
plugins:
  - id: "trakt-watched"
    type: "trakt"
    config:
      clientId: "abc123..."
      accessToken: "xyz789..."
      daysBack: 7
      daysForward: 14

calendars:
  - name: "tv-shows"
    description: "My TV Shows"
    plugins:
      - "trakt-watched"
```

## Event Format

Each episode becomes a calendar event with:

- **Summary**: `Show Name - S01E05: Episode Title`
- **Description**: Episode overview and network information
- **Start Time**: Episode air time
- **End Time**: Start time + show runtime (or +1 hour if runtime unknown)
- **URL**: Link to the episode on Trakt.tv
- **Categories**: `tv`, `trakt`

## Notes

- Access tokens from Trakt do not expire by default, but can be revoked
- The plugin fetches both past and future episodes within the configured window
- Episodes are only included if they're from shows you're actively watching on Trakt
