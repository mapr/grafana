package loki_annotations

const (
	annotationsStream = "grafana_annotations"
	changesStream     = "grafana_annotations_changes"
)

// AnnotationEntry represents an annotation entry in the main stream
type AnnotationEntry struct {
	ID           string   `json:"id"`
	OrgID        int64    `json:"org_id"`
	UserID       int64    `json:"user_id,omitempty"`
	DashboardUID string   `json:"dashboard_uid,omitempty"`
	PanelID      int64    `json:"panel_id,omitempty"`
	Text         string   `json:"text"`
	Tags         []string `json:"tags,omitempty"`
	Time         int64    `json:"time"`     // event start time, milliseconds
	TimeEnd      int64    `json:"time_end"` // event end time, milliseconds, optional
	Created      int64    `json:"created"`  // annotation creation time, milliseconds
}

// ChangeEntry represents a change entry in the changes stream
type ChangeEntry struct {
	AnnotationID string   `json:"annotation_id"`
	Operation    string   `json:"operation"`      // "update" or "delete"
	Created      int64    `json:"created"`        // milliseconds - time when the change was made
	Text         string   `json:"text,omitempty"` // only for "update", empty if unchanged
	Tags         []string `json:"tags,omitempty"` // only for "update", nil if unchanged
}
