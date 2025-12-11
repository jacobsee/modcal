package server

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/jacobsee/modcal/internal/auth"
	"github.com/jacobsee/modcal/internal/calendar"
	"github.com/jacobsee/modcal/internal/ical"
)

// Server represents the HTTP server
type Server struct {
	calManager *calendar.Manager
	auth       auth.Authenticator
	host       string
	port       int
}

// New creates a new server instance
func New(calManager *calendar.Manager, authenticator auth.Authenticator, host string, port int) *Server {
	return &Server{
		calManager: calManager,
		auth:       authenticator,
		host:       host,
		port:       port,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	mux := http.NewServeMux()

	mux.HandleFunc("/calendars", s.authMiddleware(s.handleListCalendars))
	mux.HandleFunc("/calendar/", s.authMiddleware(s.handleGetCalendar))

	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	log.Printf("Starting server on %s", addr)

	return http.ListenAndServe(addr, mux)
}

func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.auth.Authenticate(r) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func (s *Server) handleListCalendars(w http.ResponseWriter, r *http.Request) {
	calendars := s.calManager.ListCalendars()

	w.Header().Set("Content-Type", "text/plain")
	for _, name := range calendars {
		fmt.Fprintf(w, "%s\n", name)
	}
}

func (s *Server) handleGetCalendar(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/calendar/")
	if name == "" {
		http.Error(w, "Calendar name required", http.StatusBadRequest)
		return
	}

	cal, err := s.calManager.GetCalendar(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	icalData := ical.Format(cal)

	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.ics", name))
	if _, err := w.Write([]byte(icalData)); err != nil {
		log.Printf("Error writing calendar response: %v", err)
	}
}
