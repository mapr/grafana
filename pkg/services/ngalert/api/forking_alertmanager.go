package api

import (
	"strings"

	amv2 "github.com/prometheus/alertmanager/api/v2/models"
	"gopkg.in/yaml.v3"

	"github.com/grafana/grafana/pkg/api/response"
	contextmodel "github.com/grafana/grafana/pkg/services/contexthandler/model"
	"github.com/grafana/grafana/pkg/services/datasources"
	apimodels "github.com/grafana/grafana/pkg/services/ngalert/api/tooling/definitions"
	"github.com/grafana/grafana/pkg/util"
	"github.com/grafana/grafana/pkg/web"
)

const externalConfigPrefix = "__grafana-converted-external-config-"

type AlertmanagerApiHandler struct {
	AMSvc           *LotexAM
	GrafanaSvc      *AlertmanagerSrv
	ConvertSvc      *ConvertPrometheusSrv
	DatasourceCache datasources.CacheService
}

// NewForkingAM implements a set of routes that proxy to various Alertmanager-compatible backends.
func NewForkingAM(datasourceCache datasources.CacheService, proxy *LotexAM, grafana *AlertmanagerSrv, convertSvc *ConvertPrometheusSrv) *AlertmanagerApiHandler {
	return &AlertmanagerApiHandler{
		AMSvc:           proxy,
		GrafanaSvc:      grafana,
		ConvertSvc:      convertSvc,
		DatasourceCache: datasourceCache,
	}
}

func (f *AlertmanagerApiHandler) getService(ctx *contextmodel.ReqContext) (*LotexAM, error) {
	// If this is not an external config request, we should check that the datasource exists and is of the correct type.
	if isExternal, _ := f.isExternalConfig(ctx); !isExternal {
		_, err := getDatasourceByUID(ctx, f.DatasourceCache, apimodels.AlertmanagerBackend)
		if err != nil {
			return nil, err
		}
	}

	return f.AMSvc, nil
}

// isExternalConfig checks if the datasourceUID represents an external config.
// External configs are the alertmanager configurations that were saved using the Prometheus conversion API.
func (f *AlertmanagerApiHandler) isExternalConfig(ctx *contextmodel.ReqContext) (bool, string) {
	datasourceUID := web.Params(ctx.Req)[":DatasourceUID"]
	if strings.HasPrefix(datasourceUID, externalConfigPrefix) {
		identifier := strings.TrimPrefix(datasourceUID, externalConfigPrefix)
		return true, identifier
	}
	return false, ""
}

func (f *AlertmanagerApiHandler) handleRouteGetAMStatus(ctx *contextmodel.ReqContext, dsUID string) response.Response {
	if isExternal, _ := f.isExternalConfig(ctx); isExternal {
		status := apimodels.GettableStatus{
			Cluster: &amv2.ClusterStatus{
				Status: util.Pointer("ready"),
			},
		}
		return response.JSON(200, status)
	}

	s, err := f.getService(ctx)
	if err != nil {
		return errorToResponse(err)
	}

	return s.RouteGetAMStatus(ctx)
}

func (f *AlertmanagerApiHandler) handleRouteCreateSilence(ctx *contextmodel.ReqContext, body apimodels.PostableSilence, dsUID string) response.Response {
	if isExternal, _ := f.isExternalConfig(ctx); isExternal {
		return f.GrafanaSvc.RouteCreateSilence(ctx, body)
	}

	s, err := f.getService(ctx)
	if err != nil {
		return errorToResponse(err)
	}

	return s.RouteCreateSilence(ctx, body)
}

func (f *AlertmanagerApiHandler) handleRouteDeleteAlertingConfig(ctx *contextmodel.ReqContext, dsUID string) response.Response {
	if isExternal, identifier := f.isExternalConfig(ctx); isExternal {
		ctx.Req.Header.Set(configIdentifierHeader, identifier)
		return f.ConvertSvc.RouteConvertPrometheusDeleteAlertmanagerConfig(ctx)
	}

	s, err := f.getService(ctx)
	if err != nil {
		return errorToResponse(err)
	}

	return s.RouteDeleteAlertingConfig(ctx)
}

func (f *AlertmanagerApiHandler) handleRouteDeleteSilence(ctx *contextmodel.ReqContext, silenceID string, dsUID string) response.Response {
	if isExternal, _ := f.isExternalConfig(ctx); isExternal {
		return f.GrafanaSvc.RouteDeleteSilence(ctx, silenceID)
	}

	s, err := f.getService(ctx)
	if err != nil {
		return errorToResponse(err)
	}

	return s.RouteDeleteSilence(ctx, silenceID)
}

func (f *AlertmanagerApiHandler) handleRouteGetAlertingConfig(ctx *contextmodel.ReqContext, dsUID string) response.Response {
	if isExternal, identifier := f.isExternalConfig(ctx); isExternal {
		ctx.Req.Header.Set(configIdentifierHeader, identifier)

		conversionResp := f.ConvertSvc.RouteConvertPrometheusGetAlertmanagerConfig(ctx)
		if conversionResp.Status() != 200 {
			return conversionResp
		}

		var gettableUserConfig apimodels.GettableUserConfig
		if err := yaml.Unmarshal(conversionResp.Body(), &gettableUserConfig); err != nil {
			return response.Error(500, "Failed to parse alertmanager config", err)
		}

		return response.JSON(200, gettableUserConfig)
	}

	s, err := f.getService(ctx)
	if err != nil {
		return errorToResponse(err)
	}

	return s.RouteGetAlertingConfig(ctx)
}

func (f *AlertmanagerApiHandler) handleRouteGetAMAlertGroups(ctx *contextmodel.ReqContext, dsUID string) response.Response {
	if isExternal, _ := f.isExternalConfig(ctx); isExternal {
		return f.GrafanaSvc.RouteGetAMAlertGroups(ctx)
	}

	s, err := f.getService(ctx)
	if err != nil {
		return errorToResponse(err)
	}

	return s.RouteGetAMAlertGroups(ctx)
}

func (f *AlertmanagerApiHandler) handleRouteGetAMAlerts(ctx *contextmodel.ReqContext, dsUID string) response.Response {
	if isExternal, _ := f.isExternalConfig(ctx); isExternal {
		return f.GrafanaSvc.RouteGetAMAlerts(ctx)
	}

	s, err := f.getService(ctx)
	if err != nil {
		return errorToResponse(err)
	}

	return s.RouteGetAMAlerts(ctx)
}

func (f *AlertmanagerApiHandler) handleRouteGetSilence(ctx *contextmodel.ReqContext, silenceID string, dsUID string) response.Response {
	if isExternal, _ := f.isExternalConfig(ctx); isExternal {
		return f.GrafanaSvc.RouteGetSilence(ctx, silenceID)
	}

	s, err := f.getService(ctx)
	if err != nil {
		return errorToResponse(err)
	}

	return s.RouteGetSilence(ctx, silenceID)
}

func (f *AlertmanagerApiHandler) handleRouteGetSilences(ctx *contextmodel.ReqContext, dsUID string) response.Response {
	if isExternal, _ := f.isExternalConfig(ctx); isExternal {
		return f.GrafanaSvc.RouteGetSilences(ctx)
	}

	s, err := f.getService(ctx)
	if err != nil {
		return errorToResponse(err)
	}

	return s.RouteGetSilences(ctx)
}

func (f *AlertmanagerApiHandler) handleRoutePostAlertingConfig(ctx *contextmodel.ReqContext, body apimodels.PostableUserConfig, dsUID string) response.Response {
	if isExternal, identifier := f.isExternalConfig(ctx); isExternal {
		ctx.Req.Header.Set(configIdentifierHeader, identifier)

		configBytes, err := yaml.Marshal(body.AlertmanagerConfig)
		if err != nil {
			return errorToResponse(err)
		}
		amConfig := apimodels.AlertmanagerUserConfig{
			AlertmanagerConfig: string(configBytes),
			TemplateFiles:      body.TemplateFiles,
		}
		return f.ConvertSvc.RouteConvertPrometheusPostAlertmanagerConfig(ctx, amConfig)
	}

	s, err := f.getService(ctx)
	if err != nil {
		return errorToResponse(err)
	}
	if !body.AlertmanagerConfig.ReceiverType().Can(apimodels.AlertmanagerReceiverType) {
		return errorToResponse(backendTypeDoesNotMatchPayloadTypeError(apimodels.AlertmanagerBackend, body.AlertmanagerConfig.ReceiverType().String()))
	}
	return s.RoutePostAlertingConfig(ctx, body)
}

func (f *AlertmanagerApiHandler) handleRoutePostAMAlerts(ctx *contextmodel.ReqContext, body apimodels.PostableAlerts, dsUID string) response.Response {
	if isExternal, _ := f.isExternalConfig(ctx); isExternal {
		return response.Error(400, "External configurations do not accept posted alerts", nil)
	}

	s, err := f.getService(ctx)
	if err != nil {
		return errorToResponse(err)
	}

	return s.RoutePostAMAlerts(ctx, body)
}

func (f *AlertmanagerApiHandler) handleRouteDeleteGrafanaSilence(ctx *contextmodel.ReqContext, id string) response.Response {
	return f.GrafanaSvc.RouteDeleteSilence(ctx, id)
}

func (f *AlertmanagerApiHandler) handleRouteDeleteGrafanaAlertingConfig(ctx *contextmodel.ReqContext) response.Response {
	return f.GrafanaSvc.RouteDeleteAlertingConfig(ctx)
}

func (f *AlertmanagerApiHandler) handleRouteCreateGrafanaSilence(ctx *contextmodel.ReqContext, body apimodels.PostableSilence) response.Response {
	return f.GrafanaSvc.RouteCreateSilence(ctx, body)
}

func (f *AlertmanagerApiHandler) handleRouteGetGrafanaAMStatus(ctx *contextmodel.ReqContext) response.Response {
	return f.GrafanaSvc.RouteGetAMStatus(ctx)
}

func (f *AlertmanagerApiHandler) handleRouteGetGrafanaAMAlerts(ctx *contextmodel.ReqContext) response.Response {
	return f.GrafanaSvc.RouteGetAMAlerts(ctx)
}

func (f *AlertmanagerApiHandler) handleRouteGetGrafanaAMAlertGroups(ctx *contextmodel.ReqContext) response.Response {
	return f.GrafanaSvc.RouteGetAMAlertGroups(ctx)
}

func (f *AlertmanagerApiHandler) handleRouteGetGrafanaAlertingConfig(ctx *contextmodel.ReqContext) response.Response {
	return f.GrafanaSvc.RouteGetAlertingConfig(ctx)
}

func (f *AlertmanagerApiHandler) handleRouteGetGrafanaAlertingConfigHistory(ctx *contextmodel.ReqContext) response.Response {
	return f.GrafanaSvc.RouteGetAlertingConfigHistory(ctx)
}

func (f *AlertmanagerApiHandler) handleRoutePostGrafanaAlertingConfigHistoryActivate(ctx *contextmodel.ReqContext, id string) response.Response {
	return f.GrafanaSvc.RoutePostGrafanaAlertingConfigHistoryActivate(ctx, id)
}

func (f *AlertmanagerApiHandler) handleRouteGetGrafanaSilence(ctx *contextmodel.ReqContext, id string) response.Response {
	return f.GrafanaSvc.RouteGetSilence(ctx, id)
}

func (f *AlertmanagerApiHandler) handleRouteGetGrafanaSilences(ctx *contextmodel.ReqContext) response.Response {
	return f.GrafanaSvc.RouteGetSilences(ctx)
}

func (f *AlertmanagerApiHandler) handleRouteGetGrafanaReceivers(ctx *contextmodel.ReqContext) response.Response {
	return f.GrafanaSvc.RouteGetReceivers(ctx)
}

func (f *AlertmanagerApiHandler) handleRoutePostTestGrafanaReceivers(ctx *contextmodel.ReqContext, conf apimodels.TestReceiversConfigBodyParams) response.Response {
	return f.GrafanaSvc.RoutePostTestReceivers(ctx, conf)
}

func (f *AlertmanagerApiHandler) handleRoutePostTestGrafanaTemplates(ctx *contextmodel.ReqContext, conf apimodels.TestTemplatesConfigBodyParams) response.Response {
	return f.GrafanaSvc.RoutePostTestTemplates(ctx, conf)
}
