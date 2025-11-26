package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// SampleWithMetadata extends lokiclient.Sample to support structured metadata
type SampleWithMetadata struct {
	T                  time.Time
	V                  string
	StructuredMetadata map[string]string
}

// StreamWithMetadata extends lokiclient.Stream to support structured metadata
type StreamWithMetadata struct {
	Stream map[string]string
	Values []SampleWithMetadata
}

// MarshalJSON custom marshaling to include structured metadata in the values array
// Format: [timestamp, log_line, {structuredMetadata: {...}}]
func (s *SampleWithMetadata) MarshalJSON() ([]byte, error) {
	if len(s.StructuredMetadata) == 0 {
		// Standard format: [timestamp, log_line]
		return json.Marshal([2]string{
			fmt.Sprintf("%d", s.T.UnixNano()),
			s.V,
		})
	}
	// Format with structured metadata: [timestamp, log_line, {structuredMetadata: {...}}]
	return json.Marshal([]interface{}{
		fmt.Sprintf("%d", s.T.UnixNano()),
		s.V,
		map[string]interface{}{
			"structuredMetadata": s.StructuredMetadata,
		},
	})
}

// pushWithStructuredMetadata pushes a stream with structured metadata to Loki
// This method manually constructs the JSON payload to include structured metadata
// in the format Loki expects: [timestamp, log_line, {structuredMetadata: {...}}]
func (s *Store) pushWithStructuredMetadata(ctx context.Context, stream StreamWithMetadata) error {
	// Create JSON payload with structured metadata
	streamsJSON := struct {
		Streams []StreamWithMetadata `json:"streams"`
	}{
		Streams: []StreamWithMetadata{stream},
	}

	jsonData, err := json.Marshal(streamsJSON)
	if err != nil {
		return fmt.Errorf("failed to marshal stream with structured metadata: %w", err)
	}

	// Push to Loki using HTTP directly
	uri := s.lokiURL.JoinPath("/loki/api/v1/push")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uri.String(), bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create Loki request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Use a simple HTTP client to send the request
	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request to Loki: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Loki returned non-200 status code: %d", resp.StatusCode)
	}

	return nil
}
