package main

import (
	"sort"
	"strconv"
)

// Merger handles merging annotations with changes
type Merger struct{}

// NewMerger creates a new merger
func NewMerger() *Merger {
	return &Merger{}
}

// Merge merges annotations with changes, applying the latest change to each annotation
func (m *Merger) Merge(
	annEntries []*AnnotationEntry,
	changes []*ChangeEntry,
	query Query,
) []*Annotation {
	// If no changes, convert annotations directly
	if len(changes) == 0 {
		return m.annotationsToDTOs(annEntries)
	}

	// Group changes by annotation ID, keeping only the latest change for each
	changesByID := make(map[string]*ChangeEntry)
	for _, change := range changes {
		existing, ok := changesByID[change.AnnotationID]
		if !ok || change.Created > existing.Created {
			changesByID[change.AnnotationID] = change
		}
	}

	// Create a map of annotations for quick lookup
	annotationMap := make(map[string]*AnnotationEntry)
	for _, ann := range annEntries {
		annotationMap[ann.ID] = ann
	}

	// Apply changes and build result
	result := make([]*Annotation, 0, len(annEntries))
	for _, ann := range annEntries {
		// Check if there's a change for this annotation
		if change, hasChange := changesByID[ann.ID]; hasChange {
			if change.Operation == "delete" {
				// Skip deleted annotations
				continue
			}
			if change.Operation == "update" {
				// Apply changes
				m.applyChanges(ann, change)
			}
		}

		// Convert to annotation
		annotation := m.annotationEntryToAnnotation(ann)
		result = append(result, annotation)
	}

	// Sort by time (descending)
	sort.Slice(result, func(i, j int) bool {
		if result[i].Time != result[j].Time {
			return result[i].Time > result[j].Time
		}
		return result[i].TimeEnd > result[j].TimeEnd
	})

	// Apply limit
	if query.Limit > 0 && int64(len(result)) > query.Limit {
		result = result[:query.Limit]
	}

	return result
}

// applyChanges applies changes to an annotation entry
func (m *Merger) applyChanges(ann *AnnotationEntry, change *ChangeEntry) {
	if change.Text != "" {
		ann.Text = change.Text
	}
	if change.Tags != nil {
		ann.Tags = change.Tags
	}
	// Time and TimeEnd cannot be changed - they remain from the original annotation
}

// annotationsToDTOs converts annotation entries to annotations
func (m *Merger) annotationsToDTOs(entries []*AnnotationEntry) []*Annotation {
	result := make([]*Annotation, 0, len(entries))
	for _, entry := range entries {
		result = append(result, m.annotationEntryToAnnotation(entry))
	}
	return result
}

// annotationEntryToAnnotation converts a single annotation entry to annotation
func (m *Merger) annotationEntryToAnnotation(entry *AnnotationEntry) *Annotation {
	ann := &Annotation{
		Time:         entry.Time,
		TimeEnd:      entry.TimeEnd,
		Text:         entry.Text,
		Tags:         entry.Tags,
		OrgID:        entry.OrgID,
		UserID:       entry.UserID,
		PanelID:      entry.PanelID,
		Created:      entry.Created,
		DashboardUID: entry.DashboardUID,
	}

	// Parse ID to get numeric ID
	if len(entry.ID) > 4 && entry.ID[:4] == "ann-" {
		if id, err := parseID(entry.ID[4:]); err == nil {
			ann.ID = id
		}
	}

	return ann
}

// parseID parses annotation ID from string
func parseID(idStr string) (int64, error) {
	// Try to parse as int64
	return strconv.ParseInt(idStr, 10, 64)
}
