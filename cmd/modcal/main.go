package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/jacobsee/modcal/internal/auth"
	"github.com/jacobsee/modcal/internal/calendar"
	"github.com/jacobsee/modcal/internal/config"
	"github.com/jacobsee/modcal/internal/plugin"
	"github.com/jacobsee/modcal/internal/server"

	// Plugins
	"github.com/jacobsee/modcal/plugins/anilist"
	"github.com/jacobsee/modcal/plugins/example"
	"github.com/jacobsee/modcal/plugins/mal"
	"github.com/jacobsee/modcal/plugins/trakt"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	flag.Parse()

	cfg, err := config.LoadFromFile(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	registry := plugin.NewRegistry()
	if err := registerPlugins(registry); err != nil {
		log.Fatalf("Failed to register plugins: %v", err)
	}

	pluginManager := calendar.NewPluginManager()
	if err := initializePlugins(cfg, registry, pluginManager); err != nil {
		log.Fatalf("Failed to initialize plugins: %v", err)
	}

	calManager := calendar.NewManager(pluginManager)
	for _, calCfg := range cfg.Calendars {
		calManager.AddCalendar(&calendar.CalendarDefinition{
			Name:        calCfg.Name,
			Description: calCfg.Description,
			PluginIDs:   calCfg.PluginIDs,
		})
	}

	log.Println("Performing initial event fetch...")
	if err := calManager.RefreshEvents(context.Background()); err != nil {
		log.Printf("Warning: Initial event fetch failed: %v", err)
	}

	go startScheduler(calManager, cfg.Scheduler.Interval)

	authenticator := auth.NewAuthenticator(cfg.Auth.Method, cfg.Auth.APIKey)
	srv := server.New(calManager, authenticator, cfg.Server.Host, cfg.Server.Port)

	log.Fatal(srv.Start())
}

func registerPlugins(registry *plugin.Registry) error {
	plugins := []plugin.Plugin{
		example.New(),
		trakt.New(),
		anilist.New(),
		mal.New(),
	}

	for _, p := range plugins {
		if err := registry.Register(p); err != nil {
			return err
		}
	}

	return nil
}

func initializePlugins(cfg *config.Config, registry *plugin.Registry, pm *calendar.PluginManager) error {
	for _, pluginCfg := range cfg.Plugins {
		template, err := registry.Get(pluginCfg.Type)
		if err != nil {
			return err
		}

		instance, err := template.Create(pluginCfg.Config)
		if err != nil {
			return fmt.Errorf("failed to create plugin %s: %w", pluginCfg.ID, err)
		}

		pm.AddInstance(pluginCfg.ID, instance)
		log.Printf("Initialized plugin: %s (type: %s)", pluginCfg.ID, pluginCfg.Type)
	}

	return nil
}

func startScheduler(calManager *calendar.Manager, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("Starting event refresh scheduler (interval: %v)", interval)

	for range ticker.C {
		log.Println("Refreshing events from all plugins...")
		if err := calManager.RefreshEvents(context.Background()); err != nil {
			log.Printf("Error refreshing events: %v", err)
		} else {
			log.Println("Events refreshed successfully")
		}
	}
}
