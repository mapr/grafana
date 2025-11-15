package loki_annotations

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/grafana/alerting/notify/historian/lokiclient"
	ngmetrics "github.com/grafana/grafana/pkg/services/ngalert/metrics"

	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/infra/tracing"
	"github.com/grafana/grafana/pkg/services/annotations"
	"github.com/grafana/grafana/pkg/services/annotations/accesscontrol"
	"github.com/grafana/grafana/pkg/services/ngalert/metrics"
	"github.com/grafana/grafana/pkg/setting"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/grafana/grafana/pkg/apimachinery/errutil"
)

var (
	ErrLokiAnnotationsInternal = errutil.Internal("annotations.loki.internal")
	ErrLokiAnnotationsNotFound = errutil.NotFound("annotations.loki.notFound")
)

type lokiAnnotationsClient interface {
	Push(ctx context.Context, streams []lokiclient.Stream) error
	RangeQuery(ctx context.Context, query string, start, end, limit int64) (lokiclient.QueryRes, error)
	MaxQuerySize() int
}

// LokiAnnotationsStore implements readStore and writeStore for annotations using Loki
type LokiAnnotationsStore struct {
	client  lokiAnnotationsClient
	log     log.Logger
	metrics *metrics.Historian
}

// NewLokiAnnotationsStore creates a new Loki store for annotations
func NewLokiAnnotationsStore(
	cfg setting.AnnotationsLokiSettings,
	log log.Logger,
	tracer tracing.Tracer,
	reg prometheus.Registerer,
) (*LokiAnnotationsStore, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	if cfg.URL == "" {
		return nil, fmt.Errorf("Loki URL must be provided")
	}

	lokiURL, err := url.Parse(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Loki URL: %w", err)
	}

	lokiCfg := lokiclient.LokiConfig{
		ReadPathURL:       lokiURL,
		WritePathURL:      lokiURL,
		BasicAuthUser:     cfg.BasicAuthUser,
		BasicAuthPassword: cfg.BasicAuthPassword,
		TenantID:          cfg.TenantID,
		ExternalLabels:    make(map[string]string),
		MaxQueryLength:    0,
		MaxQuerySize:      0,
		Encoder:           lokiclient.SnappyProtoEncoder{},
	}

	historianMetrics := ngmetrics.NewHistorianMetrics(reg, "annotations")
	client := lokiclient.NewLokiClient(
		lokiCfg,
		lokiclient.NewRequester(),
		historianMetrics.BytesWritten,
		historianMetrics.WriteDuration,
		log,
		tracer,
		"annotations.loki",
	)

	return &LokiAnnotationsStore{
		client:  client,
		log:     log,
		metrics: historianMetrics,
	}, nil
}

func (s *LokiAnnotationsStore) Type() string {
	return "loki-annotations"
}

// === Write operations ===

// Add creates a new annotation in the main stream
func (s *LokiAnnotationsStore) Add(ctx context.Context, item *annotations.Item) error {
	// Generate ID if not set
	if item.ID == 0 {
		// Use timestamp-based ID for simplicity
		item.ID = time.Now().UnixNano() / int64(time.Millisecond)
	}

	// Set defaults for timestamps
	now := time.Now().UnixMilli()
	if item.Epoch == 0 {
		item.Epoch = now
	}
	if item.Created == 0 {
		item.Created = item.Epoch
	}

	entry := AnnotationEntry{
		ID:           fmt.Sprintf("ann-%d", item.ID),
		OrgID:        item.OrgID,
		UserID:       item.UserID,
		DashboardUID: item.DashboardUID,
		PanelID:      item.PanelID,
		Text:         item.Text,
		Tags:         item.Tags,
		Time:         item.Epoch,
		TimeEnd:      item.EpochEnd,
		Created:      item.Created,
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

	// Use current time for Loki timestamp to avoid "entry too far behind" errors
	// The original timestamp is preserved in the JSON entry
	lokiTimestamp := time.Now()
	// If the annotation timestamp is recent (within last hour), use it
	if item.Epoch > 0 {
		annotationTime := time.Unix(0, item.Epoch*1e6)
		timeDiff := time.Since(annotationTime)
		// If annotation is less than 1 hour old, use its timestamp
		if timeDiff < time.Hour && timeDiff >= 0 {
			lokiTimestamp = annotationTime
		}
	}

	stream := lokiclient.Stream{
		Stream: labels,
		Values: []lokiclient.Sample{{
			T: lokiTimestamp,
			V: string(logLine),
		}},
	}

	return s.client.Push(ctx, []lokiclient.Stream{stream})
}

// AddMany inserts multiple annotations at once
func (s *LokiAnnotationsStore) AddMany(ctx context.Context, items []annotations.Item) error {
	streams := make([]lokiclient.Stream, 0, len(items))
	now := time.Now().UnixMilli()

	for i := range items {
		item := &items[i]
		if item.ID == 0 {
			item.ID = now + int64(i)
		}

		// Set defaults for timestamps
		if item.Epoch == 0 {
			item.Epoch = now
		}
		if item.Created == 0 {
			item.Created = item.Epoch
		}

		entry := AnnotationEntry{
			ID:           fmt.Sprintf("ann-%d", item.ID),
			OrgID:        item.OrgID,
			UserID:       item.UserID,
			DashboardUID: item.DashboardUID,
			PanelID:      item.PanelID,
			Text:         item.Text,
			Tags:         item.Tags,
			Time:         item.Epoch,
			TimeEnd:      item.EpochEnd,
			Created:      item.Created,
		}

		logLine, err := json.Marshal(entry)
		if err != nil {
			s.log.Warn("Failed to marshal annotation entry", "error", err, "id", item.ID)
			continue
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

		// Use current time for Loki timestamp to avoid "entry too far behind" errors
		lokiTimestamp := time.Now()
		if item.Epoch > 0 {
			annotationTime := time.Unix(0, item.Epoch*1e6)
			timeDiff := time.Since(annotationTime)
			// If annotation is less than 1 hour old, use its timestamp
			if timeDiff < time.Hour && timeDiff >= 0 {
				lokiTimestamp = annotationTime
			}
		}

		streams = append(streams, lokiclient.Stream{
			Stream: labels,
			Values: []lokiclient.Sample{{
				T: lokiTimestamp,
				V: string(logLine),
			}},
		})
	}

	if len(streams) == 0 {
		return nil
	}

	return s.client.Push(ctx, streams)
}

// Update writes a change entry to the changes stream
func (s *LokiAnnotationsStore) Update(ctx context.Context, item *annotations.Item) error {
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
func (s *LokiAnnotationsStore) Delete(ctx context.Context, params *annotations.DeleteParams) error {
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

// CleanAnnotations is a no-op for Loki (Loki handles retention automatically)
func (s *LokiAnnotationsStore) CleanAnnotations(ctx context.Context, cfg setting.AnnotationCleanupSettings, annotationType string) (int64, error) {
	// Loki handles retention automatically based on its configuration
	// This is a no-op
	return 0, nil
}

// CleanOrphanedAnnotationTags is a no-op for Loki
func (s *LokiAnnotationsStore) CleanOrphanedAnnotationTags(ctx context.Context) (int64, error) {
	// Tags are stored inline with annotations in Loki
	return 0, nil
}

// === Read operations ===

// Get retrieves annotations with merge
func (s *LokiAnnotationsStore) Get(
	ctx context.Context,
	query annotations.ItemQuery,
	accessResources *accesscontrol.AccessResources,
) ([]*annotations.ItemDTO, error) {
	// Log query parameters for debugging
	if query.AnnotationID != 0 {
		s.log.Debug("Get annotation by ID",
			"annotation_id", query.AnnotationID,
			"org_id", query.OrgID,
			"from", query.From,
			"to", query.To,
			"skip_access_control", accessResources != nil && accessResources.SkipAccessControlFilter)
	}

	// 1. Query main stream (from...to)
	annotations, err := s.queryMainStream(ctx, query)
	if err != nil {
		return nil, err
	}

	if query.AnnotationID != 0 {
		s.log.Debug("Found annotations in main stream", "count", len(annotations), "annotation_id", query.AnnotationID)
	}

	// 2. Query changes stream (from...now)
	// Limit changes query to last 30 days to avoid Loki query range limits
	now := time.Now().UnixMilli()
	if query.To == 0 {
		query.To = now
	}

	// Limit changes query to reasonable range (30 days max)
	maxChangesRange := 30 * 24 * time.Hour
	changesFrom := query.From
	thirtyDaysAgo := now - int64(maxChangesRange.Milliseconds())

	// If From is 0 or very old, limit to last 30 days
	if changesFrom == 0 || changesFrom < thirtyDaysAgo {
		changesFrom = thirtyDaysAgo
		if query.From != 0 {
			s.log.Debug("Limiting changes query range to last 30 days", "original_from", query.From, "limited_from", changesFrom)
		}
	}

	changes, err := s.queryChangesStream(ctx, changesFrom, now)
	if err != nil {
		return nil, err
	}

	// Log changes for debugging when searching by ID
	if query.AnnotationID != 0 {
		targetID := fmt.Sprintf("ann-%d", query.AnnotationID)
		relevantChanges := make([]string, 0)
		for _, change := range changes {
			if change.AnnotationID == targetID {
				relevantChanges = append(relevantChanges, fmt.Sprintf("%s@%d", change.Operation, change.Created))
			}
		}
		if len(relevantChanges) > 0 {
			s.log.Debug("Found changes for annotation",
				"annotation_id", query.AnnotationID,
				"changes", relevantChanges)
		}
	}

	// 3. Merge annotations with changes
	merger := NewMerger()
	result := merger.Merge(annotations, changes, query)

	if query.AnnotationID != 0 {
		s.log.Debug("After merge", "count", len(result), "annotation_id", query.AnnotationID)
		if len(result) > 0 {
			s.log.Debug("Merged annotation details",
				"id", result[0].ID,
				"dashboard_uid", result[0].DashboardUID,
				"panel_id", result[0].PanelID,
				"text", result[0].Text)
		}
	}

	return result, nil
}

// GetTags returns tags from annotations
func (s *LokiAnnotationsStore) GetTags(ctx context.Context, query annotations.TagsQuery) (annotations.FindTagsResult, error) {
	// Query annotations from the last 30 days to collect tags (Loki limit)
	// This is a reasonable default - can be optimized later with caching
	now := time.Now().UnixMilli()
	maxRange := 30 * 24 * time.Hour
	thirtyDaysAgo := now - int64(maxRange.Milliseconds())

	itemQuery := annotations.ItemQuery{
		OrgID: query.OrgID,
		From:  thirtyDaysAgo,
		To:    now,
		Limit: 10000, // Query a large batch to collect tags
	}

	// Get annotations with changes applied
	anns, err := s.Get(ctx, itemQuery, &accesscontrol.AccessResources{
		SkipAccessControlFilter: true,
	})
	if err != nil {
		return annotations.FindTagsResult{Tags: []*annotations.TagsDTO{}}, err
	}

	// Collect all tags from annotations
	tagCounts := make(map[string]int64)
	for _, ann := range anns {
		for _, tagStr := range ann.Tags {
			if tagStr == "" {
				continue
			}
			// Filter by query.Tag if provided (case-insensitive LIKE)
			if query.Tag != "" {
				tagLower := strings.ToLower(tagStr)
				queryTagLower := strings.ToLower(query.Tag)
				if !strings.Contains(tagLower, queryTagLower) {
					continue
				}
			}
			tagCounts[tagStr]++
		}
	}

	// Convert to TagsDTO
	tags := make([]*annotations.TagsDTO, 0, len(tagCounts))
	for tagStr, count := range tagCounts {
		tags = append(tags, &annotations.TagsDTO{
			Tag:   tagStr,
			Count: count,
		})
	}

	// Sort by tag string (ascending)
	sort.Slice(tags, func(i, j int) bool {
		return tags[i].Tag < tags[j].Tag
	})

	// Apply limit
	limit := query.Limit
	if limit == 0 {
		limit = 100
	}
	if int64(len(tags)) > limit {
		tags = tags[:limit]
	}

	return annotations.FindTagsResult{Tags: tags}, nil
}

// queryMainStream queries the main annotations stream
func (s *LokiAnnotationsStore) queryMainStream(
	ctx context.Context,
	query annotations.ItemQuery,
) ([]*AnnotationEntry, error) {
	// Build LogQL query - all labels must be inside one {} block
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

	// If AnnotationID is specified, we'll filter in code after querying
	// LogQL JSON filtering can be unreliable, so we query all and filter by ID
	if query.AnnotationID != 0 {
		s.log.Debug("Searching for annotation by ID (will filter in code)", "annotation_id", query.AnnotationID)
	}

	// Convert timestamps to nanoseconds
	now := time.Now().UnixNano()
	from := query.From * 1e6
	to := query.To * 1e6
	if to == 0 {
		to = now
	}

	// Limit query range to 30 days to avoid Loki query range limits
	// For AnnotationID queries, we still use 30 days but LogQL filter helps find the annotation
	maxRange := 30 * 24 * time.Hour
	thirtyDaysAgo := now - int64(maxRange.Nanoseconds())
	if from == 0 || from < thirtyDaysAgo {
		from = thirtyDaysAgo
		if query.From != 0 {
			s.log.Debug("Limiting main stream query range to last 30 days", "original_from", query.From, "limited_from", from/1e6)
		}
	}

	limit := query.Limit
	if limit == 0 {
		limit = 100
	}
	// If searching by AnnotationID, we need to get more results to filter
	// because we filter by ID in code, not in LogQL
	// This ensures we find the annotation even if it's not the first one in the range
	if query.AnnotationID != 0 && limit < 1000 {
		limit = 1000 // Get more results to ensure we find the annotation
		s.log.Debug("Increased limit for AnnotationID query", "original_limit", query.Limit, "new_limit", limit)
	}

	// Log the query for debugging
	if query.AnnotationID != 0 {
		s.log.Debug("Executing LogQL query for annotation by ID",
			"logql", logQL,
			"from", from/1e6,
			"to", to/1e6,
			"limit", limit,
			"annotation_id", query.AnnotationID)
	}

	res, err := s.client.RangeQuery(ctx, logQL, from, to, limit)
	if err != nil {
		return nil, ErrLokiAnnotationsInternal.Errorf("failed to query main stream: %w", err)
	}

	// Log query results for debugging
	if query.AnnotationID != 0 {
		totalSamples := 0
		for _, stream := range res.Data.Result {
			totalSamples += len(stream.Values)
		}
		s.log.Debug("LogQL query results",
			"streams", len(res.Data.Result),
			"samples", totalSamples,
			"annotation_id", query.AnnotationID)
	}

	// Parse results
	entries := make([]*AnnotationEntry, 0)
	targetID := ""
	if query.AnnotationID != 0 {
		targetID = fmt.Sprintf("ann-%d", query.AnnotationID)
	}

	allFoundIDs := make([]string, 0)
	for _, stream := range res.Data.Result {
		for _, sample := range stream.Values {
			var entry AnnotationEntry
			if err := json.Unmarshal([]byte(sample.V), &entry); err != nil {
				s.log.Debug("Failed to unmarshal annotation entry", "error", err, "entry", sample.V)
				continue
			}
			allFoundIDs = append(allFoundIDs, entry.ID)

			// If AnnotationID is specified, double-check the ID matches
			if query.AnnotationID != 0 {
				if entry.ID != targetID {
					s.log.Debug("Annotation ID mismatch",
						"expected", targetID,
						"got", entry.ID,
						"found_entry_time", entry.Time,
						"found_entry_text", entry.Text)
					continue
				}
			}
			entries = append(entries, &entry)
		}
	}

	// If searching by ID and not found, log all found IDs for debugging
	if query.AnnotationID != 0 && len(entries) == 0 && len(allFoundIDs) > 0 {
		s.log.Warn("Annotation not found by ID, but found other annotations in the time range",
			"target_id", targetID,
			"found_ids", allFoundIDs,
			"total_found", len(allFoundIDs),
			"annotation_id", query.AnnotationID)
	}

	if query.AnnotationID != 0 {
		s.log.Debug("Found annotations after filtering", "count", len(entries), "annotation_id", query.AnnotationID)
	}

	return entries, nil
}

// queryChangesStream queries the changes stream
func (s *LokiAnnotationsStore) queryChangesStream(
	ctx context.Context,
	from, to int64,
) ([]*ChangeEntry, error) {
	// Build LogQL query
	logQL := fmt.Sprintf(`{stream="%s"}`, changesStream)

	// Convert timestamps to nanoseconds
	fromNs := from * 1e6
	toNs := to * 1e6

	// Query all changes (no limit, as there should be few)
	res, err := s.client.RangeQuery(ctx, logQL, fromNs, toNs, 10000)
	if err != nil {
		return nil, ErrLokiAnnotationsInternal.Errorf("failed to query changes stream: %w", err)
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
