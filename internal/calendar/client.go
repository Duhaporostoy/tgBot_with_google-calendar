package calendar

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/apognu/gocal"
)

type Event struct {
	ID          string
	Title       string
	Start       time.Time
	End         time.Time
	Description string
	Location    string
	MeetLink    string
	Links       []string
	Updated     time.Time
}

func FetchEvents(icalURL string, from, to time.Time, loc *time.Location) ([]*Event, error) {
	resp, err := http.Get(icalURL)
	if err != nil {
		return nil, fmt.Errorf("ошибка загрузки календаря: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("календарь вернул статус %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	parser := gocal.NewParser(strings.NewReader(string(body)))
	parser.Start = &from
	parser.End = &to

	if err := parser.Parse(); err != nil {
		return nil, fmt.Errorf("ошибка парсинга ics: %w", err)
	}

	var events []*Event
	for _, e := range parser.Events {
		ev := convertEvent(e, loc)
		if ev != nil {
			events = append(events, ev)
		}
	}
	return events, nil
}

func TodayEvents(icalURL string, loc *time.Location) ([]*Event, error) {
	now := time.Now().In(loc)
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	end := start.Add(24 * time.Hour)
	return FetchEvents(icalURL, start, end, loc)
}

func UpcomingEvents(icalURL string, loc *time.Location) ([]*Event, error) {
	now := time.Now().In(loc)
	return FetchEvents(icalURL, now, now.Add(7*24*time.Hour), loc)
}

func convertEvent(e gocal.Event, loc *time.Location) *Event {
	if e.Summary == "" {
		return nil
	}

	ev := &Event{
		ID:          e.Uid,
		Title:       e.Summary,
		Description: e.Description,
		Location:    e.Location,
	}

	if e.Start != nil {
		ev.Start = e.Start.In(loc)
	}
	if e.End != nil {
		ev.End = e.End.In(loc)
	}
	if e.LastModified != nil {
		ev.Updated = *e.LastModified
	}

	for _, text := range []string{e.Description, e.Location} {
		if link := findMeetLink(text); link != "" {
			ev.MeetLink = link
			break
		}
	}

	ev.Links = extractLinks(e.Description)

	return ev
}

func findMeetLink(text string) string {
	for _, word := range splitWords(text) {
		if strings.Contains(word, "meet.google.com") {
			return trimLinkSuffix(word)
		}
	}
	return ""
}

func extractLinks(text string) []string {
	if text == "" {
		return nil
	}
	var links []string
	seen := map[string]bool{}
	for _, w := range splitWords(text) {
		if len(w) > 8 && (strings.HasPrefix(w, "http://") || strings.HasPrefix(w, "https://")) {
			link := trimLinkSuffix(w)
			if !seen[link] {
				seen[link] = true
				links = append(links, link)
			}
		}
	}
	return links
}

func splitWords(s string) []string {
	var words []string
	start := -1
	for i, c := range s {
		switch c {
		case ' ', '\n', '\t', '\r', '<', '>', '"', '\\':
			if start >= 0 {
				words = append(words, s[start:i])
				start = -1
			}
		default:
			if start < 0 {
				start = i
			}
		}
	}
	if start >= 0 {
		words = append(words, s[start:])
	}
	return words
}

func trimLinkSuffix(s string) string {
	for len(s) > 0 {
		switch s[len(s)-1] {
		case '.', ',', ';', ':', '!', '?', ')', '"', '\'':
			s = s[:len(s)-1]
		default:
			return s
		}
	}
	return s
}
