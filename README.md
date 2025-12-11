# modcal

A modular calendar aggregation server that combines events from multiple sources and serves them in iCal format. Built with a plugin architecture for easy extensibility.

Initially for helping me (you?) track TV show air dates across several disparate services, but I will expand it to any other event sources useful to me in the future.

## Features

- Plugin-based architecture for event sources
- Create unified calendars from multiple event sources
- Standard iCal format compatible with all calendar apps
- Built-in web server with optional API key authentication
- Periodic auto-refresh of events

## Quick Start

### Docker (Recommended)

```bash
# Copy example config and edit with your settings
cp example.config.yaml config.yaml

# Run with Docker Compose
docker-compose up -d
```

### Native Build

```bash
# Build
go build -o modcal ./cmd/modcal

# Configure
cp example.config.yaml config.yaml
# Edit config.yaml with your settings

# Run
./modcal -config config.yaml
```

### Usage

- List calendars: `http://localhost:8080/calendars`
- Get calendar: `http://localhost:8080/calendar/tv-shows`
- With API key: `http://localhost:8080/calendar/tv-shows?apikey=your-key`

## Configuration

See `example.config.yaml` for a complete configuration template. Key sections:

- **server**: Host and port settings
- **auth**: Authentication method (`none` or `apikey`)
- **scheduler**: How often to refresh events (e.g., `15m`)
- **plugins**: Plugin instances with their configurations
- **calendars**: Calendar definitions combining multiple plugins

## Available Plugins

### Example
Demo plugin that generates sample events.

### Trakt
Fetches TV show episodes from your Trakt.tv watching list.

**Setup**: Get OAuth credentials at https://trakt.tv/oauth/applications, then run:
```bash
go build -o trakt-auth ./cmd/trakt-auth
./trakt-auth -client-id YOUR_ID -client-secret YOUR_SECRET
```

See `plugins/trakt/README.md` for details.

### AniList
Fetches anime episodes from your AniList watching list.

**Setup**: Get OAuth credentials at https://anilist.co/settings/developer, then run:
```bash
go build -o anilist-auth ./cmd/anilist-auth
./anilist-auth -client-id YOUR_ID -client-secret YOUR_SECRET
```

See `plugins/anilist/README.md` for details.

### MyAnimeList
Fetches anime broadcast schedules from your MAL watching list. Generates weekly recurring events (MAL doesn't provide specific episode dates).

**Setup**: Get OAuth credentials at https://myanimelist.net/apiconfig, then run:
```bash
go build -o mal-auth ./cmd/mal-auth
./mal-auth -client-id YOUR_ID -client-secret YOUR_SECRET
```

TODO: Persist tokens and refresh automatically, since MAL token expiry is only 1 month.

See `plugins/mal/README.md` for details.

## Creating a Plugin

Plugins implement the `plugin.Plugin` interface with three methods: `Name()`, `Create(config)`, and `FetchEvents(ctx)`. See `plugins/example/` for a complete example.

Register your plugin in `cmd/modcal/main.go` in the `registerPlugins` function.

## License

MIT
