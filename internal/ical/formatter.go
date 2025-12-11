package ical

import (
	"fmt"
	"strings"
	"time"

	"github.com/jacobsee/modcal/internal/models"
)

const (
	dateTimeFormat = "20060102T150405Z"
	dateFormat     = "20060102"
)

// Format converts a calendar model as an iCal
func Format(cal *models.Calendar) string {
	var builder strings.Builder

	builder.WriteString("BEGIN:VCALENDAR\r\n")
	builder.WriteString("VERSION:2.0\r\n")
	builder.WriteString("PRODID:-//modcal//modcal//EN\r\n")
	builder.WriteString(fmt.Sprintf("X-WR-CALNAME:%s\r\n", escapeText(cal.Name)))
	if cal.Description != "" {
		builder.WriteString(fmt.Sprintf("X-WR-CALDESC:%s\r\n", escapeText(cal.Description)))
	}

	for _, event := range cal.Events {
		builder.WriteString(formatEvent(&event))
	}

	builder.WriteString("END:VCALENDAR\r\n")

	return builder.String()
}

func formatEvent(event *models.Event) string {
	var builder strings.Builder

	builder.WriteString("BEGIN:VEVENT\r\n")
	builder.WriteString(fmt.Sprintf("UID:%s\r\n", escapeText(event.UID)))
	builder.WriteString(fmt.Sprintf("DTSTAMP:%s\r\n", formatDateTime(time.Now())))

	if event.AllDay {
		builder.WriteString(fmt.Sprintf("DTSTART;VALUE=DATE:%s\r\n", formatDate(event.StartTime)))
		if !event.EndTime.IsZero() {
			builder.WriteString(fmt.Sprintf("DTEND;VALUE=DATE:%s\r\n", formatDate(event.EndTime)))
		}
	} else {
		builder.WriteString(fmt.Sprintf("DTSTART:%s\r\n", formatDateTime(event.StartTime)))
		if !event.EndTime.IsZero() {
			builder.WriteString(fmt.Sprintf("DTEND:%s\r\n", formatDateTime(event.EndTime)))
		}
	}

	if event.Summary != "" {
		builder.WriteString(fmt.Sprintf("SUMMARY:%s\r\n", escapeText(event.Summary)))
	}

	if event.Description != "" {
		builder.WriteString(fmt.Sprintf("DESCRIPTION:%s\r\n", escapeText(event.Description)))
	}

	if event.Location != "" {
		builder.WriteString(fmt.Sprintf("LOCATION:%s\r\n", escapeText(event.Location)))
	}

	if event.URL != "" {
		builder.WriteString(fmt.Sprintf("URL:%s\r\n", event.URL))
	}

	if len(event.Categories) > 0 {
		builder.WriteString(fmt.Sprintf("CATEGORIES:%s\r\n", strings.Join(event.Categories, ",")))
	}

	builder.WriteString("END:VEVENT\r\n")

	return builder.String()
}

func formatDateTime(t time.Time) string {
	return t.UTC().Format(dateTimeFormat)
}

func formatDate(t time.Time) string {
	return t.Format(dateFormat)
}

func escapeText(text string) string {
	text = strings.ReplaceAll(text, "\\", "\\\\")
	text = strings.ReplaceAll(text, ";", "\\;")
	text = strings.ReplaceAll(text, ",", "\\,")
	text = strings.ReplaceAll(text, "\n", "\\n")
	return text
}
