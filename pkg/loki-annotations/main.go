package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

func main() {
	var (
		lokiURL = flag.String("loki-url", "http://localhost:3100", "Loki URL")
		//basicAuthUser     = flag.String("basic-auth-user", "", "Basic auth username")
		//basicAuthPassword = flag.String("basic-auth-password", "", "Basic auth password")

		text         = flag.String("text", "Test annotation", "Annotation text")
		tags         = flag.String("tags", "", "Comma-separated tags")
		orgID        = flag.Int64("org-id", 1, "Organization ID")
		dashboardUID = flag.String("dashboard-uid", "xxx-yyy-zzz", "Dashboard UID")
		panelID      = flag.Int64("panel-id", 1, "Panel ID")
		annotationID = flag.Int64("annotation-id", 0, "Annotation ID (for get/update/delete)")
		limit        = flag.Int64("limit", 100, "Query limit")
	)
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: action is required\n")
		fmt.Fprintf(os.Stderr, "Usage: %s <action> [flags]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Available actions: create, get, list-tags, update, delete\n")
		flag.Usage()
		os.Exit(1)
	}

	action := args[0]

	cfg := Config{
		URL: *lokiURL,
		//BasicAuthUser:     *basicAuthUser,
		//BasicAuthPassword: *basicAuthPassword,
	}

	// Create store
	store, err := NewStore(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating store: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	switch action {
	case "create":
		id, err := addAnnotation(ctx, store, *orgID, *text, *dashboardUID, *panelID, *tags)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating annotation: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ Annotation created with ID: %d\n", id)

	case "get":
		anns, err := getAnnotations(ctx, store, *orgID, *dashboardUID, *panelID, *annotationID, *limit)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting annotations: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ Found %d annotation(s):\n", len(anns))
		for _, ann := range anns {
			fmt.Printf("  ID: %d, Text: %s, Time: %s, Tags: %v\n",
				ann.ID, ann.Text, time.Unix(0, ann.Time*1e6).Format(time.RFC3339), ann.Tags)
		}

	case "list-tags":
		tagsResult, err := getTags(ctx, store, *orgID, *tags)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing tags: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ Found %d tag(s):\n", len(tagsResult.Tags))
		for _, tag := range tagsResult.Tags {
			fmt.Printf("  %s: %d\n", tag.Tag, tag.Count)
		}

	case "update":
		if err := updateAnnotation(ctx, store, *orgID, *annotationID, *text, *tags); err != nil {
			fmt.Fprintf(os.Stderr, "Error updating annotation: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ Annotation %d updated\n", *annotationID)

	case "delete":
		if err := deleteAnnotation(ctx, store, *orgID, *annotationID); err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting annotation: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ Annotation %d deleted\n", *annotationID)

	default:
		fmt.Fprintf(os.Stderr, "Error: unknown action: %s\n", action)
		fmt.Fprintf(os.Stderr, "Available actions: create, get, list-tags, update, delete\n")
		flag.Usage()
		os.Exit(1)
	}
}

func addAnnotation(ctx context.Context, store *Store, orgID int64, text, dashboardUID string, panelID int64, tagsStr string) (int64, error) {
	now := time.Now().UnixMilli()
	ann := &Annotation{
		OrgID:        orgID,
		UserID:       1,
		DashboardUID: dashboardUID,
		PanelID:      panelID,
		Text:         text,
		Time:         now,
		Created:      now,
	}

	if tagsStr != "" {
		tags := []string{}
		for _, tag := range splitTags(tagsStr) {
			if tag != "" {
				tags = append(tags, tag)
			}
		}
		ann.Tags = tags
	}

	if err := store.Add(ctx, ann); err != nil {
		return 0, err
	}

	return ann.ID, nil
}

func getAnnotations(ctx context.Context, store *Store, orgID int64, dashboardUID string, panelID, annotationID, limit int64) ([]*Annotation, error) {
	now := time.Now().UnixMilli()
	thirtyDaysAgo := now - int64(30*24*time.Hour.Milliseconds())

	query := Query{
		OrgID:        orgID,
		From:         thirtyDaysAgo,
		To:           now,
		DashboardUID: dashboardUID,
		PanelID:      panelID,
		AnnotationID: annotationID,
		Limit:        limit,
	}

	return store.Get(ctx, query)
}

func getTags(ctx context.Context, store *Store, orgID int64, tagFilter string) (TagsResult, error) {
	query := TagsQuery{
		OrgID: orgID,
		Tag:   tagFilter,
		Limit: 100,
	}

	return store.GetTags(ctx, query)
}

func updateAnnotation(ctx context.Context, store *Store, orgID, annotationID int64, text, tagsStr string) error {
	now := time.Now().UnixMilli()
	ann := &Annotation{
		ID:      annotationID,
		OrgID:   orgID,
		Text:    text,
		Time:    now,
		Created: now,
	}

	if tagsStr != "" {
		tags := []string{}
		for _, tag := range splitTags(tagsStr) {
			if tag != "" {
				tags = append(tags, tag)
			}
		}
		ann.Tags = tags
	}

	return store.Update(ctx, ann)
}

func deleteAnnotation(ctx context.Context, store *Store, orgID, annotationID int64) error {
	params := &DeleteParams{
		OrgID: orgID,
		ID:    annotationID,
	}

	return store.Delete(ctx, params)
}

func splitTags(tagsStr string) []string {
	tags := []string{}
	for _, tag := range strings.Split(tagsStr, ",") {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			tags = append(tags, tag)
		}
	}
	return tags
}
