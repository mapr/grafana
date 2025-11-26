package main

// Annotation represents an annotation item (used for both input and output)
type Annotation struct {
	ID           int64
	OrgID        int64
	UserID       int64
	DashboardUID string // empty string means not set
	PanelID      int64
	Text         string
	Tags         []string
	Time         int64 // event start time, milliseconds
	TimeEnd      int64 // event end time, milliseconds, optional
	Created      int64 // annotation creation time, milliseconds
}

// Query represents a query for annotations
type Query struct {
	OrgID        int64
	From         int64
	To           int64
	DashboardUID string
	PanelID      int64
	AnnotationID int64
	Limit        int64
}

// TagsQuery represents a query for tags
type TagsQuery struct {
	OrgID int64
	Tag   string
	Limit int64
}

// Tag represents a tag result
type Tag struct {
	Tag   string
	Count int64
}

// TagsResult represents the result of a tags search
type TagsResult struct {
	Tags []*Tag
}

// DeleteParams represents parameters for deleting an annotation
type DeleteParams struct {
	OrgID int64
	ID    int64
}

// Config represents configuration for Loki annotations storage
type Config struct {
	URL string
	//BasicAuthUser     string
	//BasicAuthPassword string
}
