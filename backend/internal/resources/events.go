package resources

import (
	"sort"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/Ad1tya-007/Agent-Sandbox/backend/internal/models"
)

func TranslateEvents(events []corev1.Event) []models.TimelineEvent {
	if len(events) == 0 {
		return []models.TimelineEvent{}
	}
	sorted := append([]corev1.Event(nil), events...)
	sort.SliceStable(sorted, func(i, j int) bool {
		ti, tj := eventTime(sorted[i]), eventTime(sorted[j])
		if !ti.Equal(tj) {
			return ti.Before(tj)
		}
		return sorted[i].Name < sorted[j].Name
	})

	out := make([]models.TimelineEvent, 0, len(sorted))
	seen := make(map[string]struct{}, len(sorted))
	for _, ev := range sorted {
		item := TranslateEvent(ev)
		if item.ID == "" {
			continue
		}
		if _, ok := seen[item.ID]; ok {
			continue
		}
		seen[item.ID] = struct{}{}
		out = append(out, item)
	}
	return out
}

func TranslateEvent(ev corev1.Event) models.TimelineEvent {
	at := formatTime(eventTime(ev))
	id := string(ev.UID)
	if id == "" {
		id = ev.Name + "|" + at
	}
	title, detail := titleDetail(ev)
	return models.TimelineEvent{
		ID:     id,
		Title:  title,
		Detail: detail,
		At:     at,
	}
}

func titleDetail(ev corev1.Event) (title, detail string) {
	switch ev.Reason {
	case "Created", "SuccessfulCreate":
		if ev.InvolvedObject.Kind == "Pod" {
			return "Container Started", "sandbox container started"
		}
		if ev.Reason == "SuccessfulCreate" {
			return "Sandbox Created", "controller created child"
		}
		return "Sandbox Created", "API server accepted the Sandbox resource"
	case "Started":
		return "Container Started", "sandbox container started"
	case "Scheduled":
		return "Scheduled", scheduledDetail(ev.Message)
	case "Pulling", "Pulled":
		return "Image Pulled", imagePulledDetail(ev.Message)
	case "Killing", "BackOff", "Unhealthy":
		return "Restarted", ev.Message
	case "Failed":
		return "Failed", ev.Message
	case "FailedScheduling":
		return "Pending", ev.Message
	default:
		title = ev.Reason
		if title == "" {
			title = "Event"
		}
		return title, ev.Message
	}
}

func scheduledDetail(message string) string {
	if node := lastAfter(message, " to "); node != "" {
		return "Assigned to " + node
	}
	if message != "" {
		return message
	}
	return "Assigned"
}

func imagePulledDetail(message string) string {
	if img := quoted(message); img != "" {
		return img + " pulled"
	}
	if message != "" {
		return message
	}
	return "image pulled"
}

func quoted(s string) string {
	i := strings.Index(s, `"`)
	if i < 0 {
		return ""
	}
	rest := s[i+1:]
	j := strings.Index(rest, `"`)
	if j < 0 {
		return ""
	}
	return rest[:j]
}

func lastAfter(s, sep string) string {
	i := strings.LastIndex(s, sep)
	if i < 0 {
		return ""
	}
	return strings.TrimSpace(s[i+len(sep):])
}

func eventTime(ev corev1.Event) time.Time {
	switch {
	case !ev.LastTimestamp.IsZero():
		return ev.LastTimestamp.UTC()
	case !ev.EventTime.IsZero():
		return ev.EventTime.UTC()
	case !ev.FirstTimestamp.IsZero():
		return ev.FirstTimestamp.UTC()
	case !ev.CreationTimestamp.IsZero():
		return ev.CreationTimestamp.UTC()
	default:
		return time.Time{}
	}
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
