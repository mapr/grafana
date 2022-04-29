package prometheus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana/pkg/infra/httpclient"
	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/infra/tracing"
	"github.com/grafana/grafana/pkg/services/featuremgmt"
	"github.com/grafana/grafana/pkg/setting"
	"github.com/grafana/grafana/pkg/tsdb/prometheus/buffered"
	"github.com/grafana/grafana/pkg/tsdb/prometheus/promclient"
	"github.com/grafana/grafana/pkg/util/maputil"
	apiv1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

var plog = log.New("tsdb.prometheus")

type Service struct {
	im instancemgmt.InstanceManager
}

func ProvideService(httpClientProvider httpclient.Provider, cfg *setting.Cfg, features featuremgmt.FeatureToggles, tracer tracing.Tracer) *Service {
	plog.Debug("initializing")
	return &Service{
		im: datasource.NewInstanceManager(newInstanceSettings(httpClientProvider, cfg, features, tracer)),
	}
}

func newInstanceSettings(httpClientProvider httpclient.Provider, cfg *setting.Cfg, features featuremgmt.FeatureToggles, tracer tracing.Tracer) datasource.InstanceFactoryFunc {
	return func(settings backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
		var jsonData map[string]interface{}
		err := json.Unmarshal(settings.JSONData, &jsonData)
		if err != nil {
			return nil, fmt.Errorf("error reading settings: %w", err)
		}

		p := promclient.NewProvider(settings, jsonData, httpClientProvider, cfg, features, plog)
		pc, err := promclient.NewProviderCache(p)
		if err != nil {
			return nil, err
		}

		timeInterval, err := maputil.GetStringOptional(jsonData, "timeInterval")
		if err != nil {
			return nil, err
		}

		mdl := buffered.DatasourceInfo{
			ID:           settings.ID,
			URL:          settings.URL,
			TimeInterval: timeInterval,
			Buffered:     buffered.New(tracer),
			GetClient:    pc.GetClient,
		}

		return mdl, nil
	}
}

func (s *Service) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	if len(req.Queries) == 0 {
		return &backend.QueryDataResponse{}, fmt.Errorf("query contains no queries")
	}

	q := req.Queries[0]
	dsInfo, err := s.getDSInfo(req.PluginContext)
	if err != nil {
		return nil, err
	}

	var result *backend.QueryDataResponse
	switch q.QueryType {
	case "timeSeriesQuery":
		fallthrough
	default:
		result, err = dsInfo.Buffered.ExecuteTimeSeriesQuery(ctx, req, dsInfo)
	}

	return result, err
}

func (s *Service) getDSInfo(pluginCtx backend.PluginContext) (*buffered.DatasourceInfo, error) {
	i, err := s.im.Get(pluginCtx)
	if err != nil {
		return nil, err
	}

	instance := i.(buffered.DatasourceInfo)

	return &instance, nil
}

// IsAPIError returns whether err is or wraps a Prometheus error.
func IsAPIError(err error) bool {
	// Check if the right error type is in err's chain.
	var e *apiv1.Error
	return errors.As(err, &e)
}

func ConvertAPIError(err error) error {
	var e *apiv1.Error
	if errors.As(err, &e) {
		return fmt.Errorf("%s: %s", e.Msg, e.Detail)
	}
	return err
}
