package historian

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana/pkg/components/simplejson"
	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/services/ngalert/eval"
	"github.com/grafana/grafana/pkg/services/ngalert/models"
	"github.com/grafana/grafana/pkg/services/ngalert/state"
)

const (
	OrgIDLabel     = "orgID"
	RuleUIDLabel   = "ruleUID"
	GroupLabel     = "group"
	FolderUIDLabel = "folderUID"
)

type remoteLokiClient interface {
	ping() error
	push([]stream) error
	query(ctx context.Context, selectors []Selector, start, end int64) (QueryRes, error)
}

type RemoteLokiBackend struct {
	client remoteLokiClient
	log    log.Logger
}

func NewRemoteLokiBackend(cfg LokiConfig) *RemoteLokiBackend {
	logger := log.New("ngalert.state.historian", "backend", "loki")
	return &RemoteLokiBackend{
		client: newLokiClient(cfg, logger),
		log:    logger,
	}
}

func (h *RemoteLokiBackend) TestConnection() error {
	return h.client.ping()
}

func (h *RemoteLokiBackend) RecordStatesAsync(ctx context.Context, rule *models.AlertRule, states []state.StateTransition) {
	logger := h.log.FromContext(ctx)
	streams := h.statesToStreams(rule, states, logger)
	h.recordStreamsAsync(ctx, streams, logger)
}

func (h *RemoteLokiBackend) QueryStates(ctx context.Context, query models.HistoryQuery) (*data.Frame, error) {
	if query.RuleUID == "" {
		return nil, errors.New("the RuleUID is not set but required")
	}
	selectors, err := buildSelectors(query)
	if err != nil {
		return nil, fmt.Errorf("failed to build the provided selectors: %w", err)
	}
	res, err := h.client.query(ctx, selectors, query.From.Unix(), query.To.Unix())
	if err != nil {
		return nil, err
	}
	return merge(res)
}

func buildSelectors(query models.HistoryQuery) ([]Selector, error) {
	// +2 as we put the RuleID and OrgID also into a label selector.
	selectors := make([]Selector, len(query.Labels)+2)

	// Set the predefined selector rule_id
	selector, err := NewSelector("rule_id", "=", query.RuleUID)
	if err != nil {
		return nil, err
	}
	selectors[0] = selector

	// Set the predefined selector org_id
	selector, err = NewSelector("org_id", "=", fmt.Sprintf("%d", query.OrgID))
	if err != nil {
		return nil, err
	}
	selectors[1] = selector

	// Set all the other selectors
	i := 2
	for label, val := range query.Labels {
		selector, err = NewSelector(label, "=", val)
		if err != nil {
			return nil, err
		}
		selectors[i] = selector
		i++
	}
	return selectors, nil
}

// merge will put all the results in one array sorted by timestamp.
func merge(res QueryRes) (*data.Frame, error) {
	// Find the total number of elements in all arrays
	totalLen := 0
	for _, arr := range res.Data.Result {
		totalLen += len(arr.Values)
	}
	// Create a new slice to store the merged elements
	df := data.NewFrame("result")
	// Initialize a slice of pointers to the current position in each array
	pointers := make([]int, len(res.Data.Result))
	for {
		// Find the minimum element among all arrays
		minVal := (^int64(0) >> 1) // set initial value to max int
		minIdx := -1
		minEl := [2]string{}
		for i, stream := range res.Data.Result {
			curVal, err := strconv.ParseInt(stream.Values[pointers[i]][0], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse timestamp from loki repsonse: %w", err)
			}
			if pointers[i] < len(stream.Values) && curVal < minVal {
				minVal = curVal
				minEl = stream.Values[pointers[i]]
				minIdx = i
			}
		}
		// If all pointers have reached the end of their arrays, we're done
		if minIdx == -1 {
			break
		}
		// Append the minimum element to the merged slice and move the pointer
		df.AppendRow(minEl)
		pointers[minIdx]++
	}
	return df, nil
}

func (h *RemoteLokiBackend) statesToStreams(rule *models.AlertRule, states []state.StateTransition, logger log.Logger) []stream {
	buckets := make(map[string][]row) // label repr -> entries
	for _, state := range states {
		if !shouldRecord(state) {
			continue
		}

		labels := removePrivateLabels(state.State.Labels)
		labels[OrgIDLabel] = fmt.Sprint(rule.OrgID)
		labels[RuleUIDLabel] = fmt.Sprint(rule.UID)
		labels[GroupLabel] = fmt.Sprint(rule.RuleGroup)
		labels[FolderUIDLabel] = fmt.Sprint(rule.NamespaceUID)
		repr := labels.String()

		entry := lokiEntry{
			SchemaVersion: 1,
			Previous:      state.PreviousFormatted(),
			Current:       state.Formatted(),
			Values:        valuesAsDataBlob(state.State),
		}
		jsn, err := json.Marshal(entry)
		if err != nil {
			logger.Error("Failed to construct history record for state, skipping", "error", err)
			continue
		}
		line := string(jsn)

		buckets[repr] = append(buckets[repr], row{
			At:  state.State.LastEvaluationTime,
			Val: line,
		})
	}

	result := make([]stream, 0, len(buckets))
	for repr, rows := range buckets {
		labels, err := data.LabelsFromString(repr)
		if err != nil {
			logger.Error("Failed to parse frame labels, skipping state history batch: %w", err)
			continue
		}
		result = append(result, stream{
			Stream: labels,
			Values: rows,
		})
	}

	return result
}

func (h *RemoteLokiBackend) recordStreamsAsync(ctx context.Context, streams []stream, logger log.Logger) {
	go func() {
		if err := h.recordStreams(ctx, streams, logger); err != nil {
			logger.Error("Failed to save alert state history batch", "error", err)
		}
	}()
}

func (h *RemoteLokiBackend) recordStreams(ctx context.Context, streams []stream, logger log.Logger) error {
	if err := h.client.push(streams); err != nil {
		return err
	}
	logger.Debug("Done saving alert state history batch")
	return nil
}

type lokiEntry struct {
	SchemaVersion int              `json:"schemaVersion"`
	Previous      string           `json:"previous"`
	Current       string           `json:"current"`
	Values        *simplejson.Json `json:"values"`
}

func valuesAsDataBlob(state *state.State) *simplejson.Json {
	jsonData := simplejson.New()

	switch state.State {
	case eval.Error:
		if state.Error == nil {
			jsonData.Set("error", nil)
		} else {
			jsonData.Set("error", state.Error.Error())
		}
	case eval.NoData:
		jsonData.Set("noData", true)
	default:
		keys := make([]string, 0, len(state.Values))
		for k := range state.Values {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		jsonData.Set("values", simplejson.NewFromAny(state.Values))
	}
	return jsonData
}
