package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/grafana/alerting/notify/historian/lokiclient"
	ngmetrics "github.com/grafana/grafana/pkg/services/ngalert/metrics"

	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/infra/tracing"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	ErrLokiAnnotationsInternal = errors.New("loki annotations internal error")
	ErrLokiAnnotationsNotFound = errors.New("loki annotations not found")
)

type lokiAnnotationsClient interface {
	Push(ctx context.Context, streams []lokiclient.Stream) error
	RangeQuery(ctx context.Context, query string, start, end, limit int64) (lokiclient.QueryRes, error)
	MaxQuerySize() int
}

type Store struct {
	client  lokiAnnotationsClient
	log     log.Logger
	lokiURL *url.URL
}

func NewStore(cfg Config) (*Store, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("Loki URL must be provided")
	}

	lokiURL, err := url.Parse(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Loki URL: %w", err)
	}

	// From annotations history store
	// Use JSONEncoder to support structured metadata for Time and TimeEnd
	// Structured metadata is only supported in JSON format, not protobuf
	lokiCfg := lokiclient.LokiConfig{
		ReadPathURL:  lokiURL,
		WritePathURL: lokiURL,
		//BasicAuthUser:     cfg.BasicAuthUser,
		//BasicAuthPassword: cfg.BasicAuthPassword,
		ExternalLabels: make(map[string]string),
		MaxQueryLength: 0,
		MaxQuerySize:   0,
		Encoder:        lokiclient.JSONEncoder{},
	}

	logger := log.New("test-loki-annotations")
	historianMetrics := ngmetrics.NewHistorianMetrics(prometheus.NewRegistry(), "annotations")
	client := lokiclient.NewLokiClient(
		lokiCfg,
		lokiclient.NewRequester(),
		historianMetrics.BytesWritten,
		historianMetrics.WriteDuration,
		logger,
		tracing.NewNoopTracerService(),
		"annotations.loki",
	)

	return &Store{
		client:  client,
		log:     logger,
		lokiURL: lokiURL,
	}, nil
}

// Add creates a new annotation in the main stream
func (s *Store) Add(ctx context.Context, item *Annotation) error {
	// Generate ID if not set
	if item.ID == 0 {
		// Use timestamp-based ID for simplicity
		item.ID = time.Now().UnixNano() / int64(time.Millisecond)
	}

	// Set defaults for timestamps
	now := time.Now().UnixMilli()
	if item.Time == 0 {
		item.Time = now
	}
	if item.TimeEnd == 0 {
		item.TimeEnd = now
	}

	entry := AnnotationEntry{
		ID:           fmt.Sprintf("ann-%d", item.ID),
		OrgID:        item.OrgID,
		UserID:       item.UserID,
		DashboardUID: item.DashboardUID,
		PanelID:      item.PanelID,
		Text:         item.Text,
		Tags:         item.Tags,
		Time:         item.Time,
		TimeEnd:      item.TimeEnd,
		Created:      now,
	}

	logLine, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal annotation entry: %w", err)
	}

	labels := map[string]string{
		"stream": annotationsStream,
		"org_id": strconv.FormatInt(item.OrgID, 10),
	}
	if item.DashboardUID != "" {
		labels["dashboard_uid"] = item.DashboardUID
	}
	if item.PanelID > 0 {
		labels["panel_id"] = strconv.FormatInt(item.PanelID, 10)
	}

	// Create stream with structured metadata for Time and TimeEnd
	// This allows querying by these fields using LogQL: {stream="..."} | Time="1234567890"
	streamWithMetadata := StreamWithMetadata{
		Stream: labels,
		Values: []SampleWithMetadata{{
			T: time.Now(),
			V: string(logLine),
			StructuredMetadata: map[string]string{
				"Time":    strconv.FormatInt(item.Time, 10),
				"TimeEnd": strconv.FormatInt(item.TimeEnd, 10),
			},
		}},
	}

	// Push with structured metadata using JSON format
	return s.pushWithStructuredMetadata(ctx, streamWithMetadata)
}

// Update writes a change entry to the changes stream
func (s *Store) Update(ctx context.Context, item *Annotation) error {
	changeEntry := ChangeEntry{
		AnnotationID: fmt.Sprintf("ann-%d", item.ID),
		Operation:    "update",
		Created:      time.Now().UnixMilli(),
		Text:         item.Text,
		Tags:         item.Tags,
	}

	logLine, err := json.Marshal(changeEntry)
	if err != nil {
		return fmt.Errorf("failed to marshal change entry: %w", err)
	}

	stream := lokiclient.Stream{
		Stream: map[string]string{
			"stream": changesStream,
			"org_id": strconv.FormatInt(item.OrgID, 10),
		},
		Values: []lokiclient.Sample{{
			T: time.Now(),
			V: string(logLine),
		}},
	}

	return s.client.Push(ctx, []lokiclient.Stream{stream})
}

// Delete writes a delete entry to the changes stream
func (s *Store) Delete(ctx context.Context, params *DeleteParams) error {
	changeEntry := ChangeEntry{
		AnnotationID: fmt.Sprintf("ann-%d", params.ID),
		Operation:    "delete",
		Created:      time.Now().UnixMilli(),
	}

	logLine, err := json.Marshal(changeEntry)
	if err != nil {
		return fmt.Errorf("failed to marshal delete entry: %w", err)
	}

	stream := lokiclient.Stream{
		Stream: map[string]string{
			"stream": changesStream,
			"org_id": strconv.FormatInt(params.OrgID, 10),
		},
		Values: []lokiclient.Sample{{
			T: time.Now(),
			V: string(logLine),
		}},
	}

	return s.client.Push(ctx, []lokiclient.Stream{stream})
}

// Get retrieves annotations with merge from changes stream
func (s *Store) Get(ctx context.Context, query Query) ([]*Annotation, error) {
	// Log query parameters for debugging
	if query.AnnotationID != 0 {
		s.log.Debug("Get annotation by ID",
			"annotation_id", query.AnnotationID,
			"org_id", query.OrgID,
			"from", query.From,
			"to", query.To)
	}

	// Query main stream (from...to)
	annotations, err := s.queryMainStream(ctx, query)
	if err != nil {
		return nil, err
	}
	s.log.Debug("Found annotations in main stream", "count", len(annotations))

	// Query entire changes stream and merge with latest change per annotation ID
	// We need all changes because they're matched to annotations by AnnotationID, not by time.
	// The merge logic will keep only the latest change per annotation ID.
	changes, err := s.queryChangesStream(ctx)
	if err != nil {
		return nil, err
	}

	// Merge annotations with changes
	merger := NewMerger()
	result := merger.Merge(annotations, changes, query)

	return result, nil
}

// GetTags returns tags from annotations
func (s *Store) GetTags(ctx context.Context, query TagsQuery) (TagsResult, error) {
	return TagsResult{}, fmt.Errorf("not implemented")
}

// queryMainStream queries the main annotations stream using structured metadata filters
func (s *Store) queryMainStream(
	ctx context.Context,
	query Query,
) ([]*AnnotationEntry, error) {
	// Build LogQL query with label filters
	var labels []string
	labels = append(labels, fmt.Sprintf(`stream="%s"`, annotationsStream))
	labels = append(labels, fmt.Sprintf(`org_id="%d"`, query.OrgID))

	if query.DashboardUID != "" {
		labels = append(labels, fmt.Sprintf(`dashboard_uid="%s"`, query.DashboardUID))
	}
	if query.PanelID > 0 {
		labels = append(labels, fmt.Sprintf(`panel_id="%d"`, query.PanelID))
	}

	logQL := "{" + strings.Join(labels, ",") + "}"

	var filters []string
	// TODO: Fix filtering by time
	if query.From > 0 {
		filters = append(filters, fmt.Sprintf(`Time >= "%d"`, query.From))
	}
	if query.To > 0 {
		filters = append(filters, fmt.Sprintf(`Time <= "%d"`, query.To))
	}

	if len(filters) > 0 {
		logQL += " | " + strings.Join(filters, " and ")
	}

	// For AnnotationID queries, we still filter in code after querying
	if query.AnnotationID != 0 {
		s.log.Debug("Searching for annotation by ID (will filter in code)", "annotation_id", query.AnnotationID)
	}

	// Use a wide time range for the Loki query itself (last 30 days max)
	// The structured metadata filters will narrow down the results precisely
	// Loki's RangeQuery requires time ranges, but we use a fixed window since
	// structured metadata filters handle the precise filtering
	now := time.Now().UnixNano()
	maxRange := 30 * 24 * time.Hour
	fromNs := now - int64(maxRange.Nanoseconds())
	toNs := now

	res, err := s.client.RangeQuery(ctx, logQL, fromNs, toNs, query.Limit)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to query main stream: %w", ErrLokiAnnotationsInternal, err)
	}

	// Parse results
	entries := make([]*AnnotationEntry, 0)
	targetID := ""
	if query.AnnotationID != 0 {
		targetID = fmt.Sprintf("ann-%d", query.AnnotationID)
	}

	for _, stream := range res.Data.Result {
		for _, sample := range stream.Values {
			var entry AnnotationEntry
			if err := json.Unmarshal([]byte(sample.V), &entry); err != nil {
				s.log.Debug("Failed to unmarshal annotation entry", "error", err, "entry", sample.V)
				continue
			}

			// Filter by AnnotationID if specified
			if query.AnnotationID != 0 && entry.ID != targetID {
				continue
			}

			entries = append(entries, &entry)
		}
	}

	return entries, nil
}

func (s *Store) queryChangesStream(
	ctx context.Context,
) ([]*ChangeEntry, error) {
	logQL := fmt.Sprintf(`{stream="%s"}`, changesStream)

	// Use a very old timestamp (epoch 0) to query the entire stream
	// Loki's MaxQueryLength will clamp this if there's a limit, but we want all available changes

	fromNs := int64(0) // Start from epoch 0 to get all changes
	toNs := time.Now().UnixNano()

	res, err := s.client.RangeQuery(ctx, logQL, fromNs, toNs, 10000)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to query changes stream: %w", ErrLokiAnnotationsInternal, err)
	}

	// Parse results
	changes := make([]*ChangeEntry, 0)
	for _, stream := range res.Data.Result {
		for _, sample := range stream.Values {
			var change ChangeEntry
			if err := json.Unmarshal([]byte(sample.V), &change); err != nil {
				s.log.Debug("Failed to unmarshal change entry", "error", err, "entry", sample.V)
				continue
			}
			changes = append(changes, &change)
		}
	}

	return changes, nil
}
