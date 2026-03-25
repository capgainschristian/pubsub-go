package events

import "time"

const (
	RouteNotification = "notification.general"
	RouteAlert        = "notification.alert"
	RouteAll          = "notification.#"
)

type Event struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"`
	Severity  string            `json:"severity"`
	Title     string            `json:"title"`
	Body      string            `json:"body"`
	Source    string            `json:"source"`
	Timestamp time.Time         `json:"timestamp"`
	Meta      map[string]string `json:"meta,omitempty"`
}

const (
	SeverityLow      = "low"
	SeverityMedium   = "medium"
	SeverityHigh     = "high"
	SeverityCritical = "critical"
)

func SeverityColor(s string) string {
	switch s {
	case SeverityLow:
		return "#36a64f" // green
	case SeverityMedium:
		return "#ffae42" // orange
	case SeverityHigh:
		return "#ff0000" // red
	case SeverityCritical:
		return "#8b0000" // dark red
	default:
		return "#808080" // gray
	}
}
