package api

import (
	"net/http"
	"strings"

	amv2 "github.com/prometheus/alertmanager/api/v2/models"

	"github.com/grafana/grafana/pkg/api/response"
	contextmodel "github.com/grafana/grafana/pkg/services/contexthandler/model"
	"github.com/grafana/grafana/pkg/services/datasources"
	"github.com/grafana/grafana/pkg/services/featuremgmt"
	apimodels "github.com/grafana/grafana/pkg/services/ngalert/api/tooling/definitions"
	"github.com/grafana/grafana/pkg/services/org"
	"github.com/grafana/grafana/pkg/util"
	"github.com/grafana/grafana/pkg/web"
)

const extraConfigPrefix = "__grafana-converted-extra-config-"

func convertPostableToGettableReceivers(postableReceivers []*apimodels.PostableApiReceiver) []*apimodels.GettableApiReceiver {
	if postableReceivers == nil {
		return nil
	}

	gettableReceivers := make([]*apimodels.GettableApiReceiver, 0, len(postableReceivers))

	for _, postableReceiver := range postableReceivers {
		gettableGrafanaReceivers := make([]*apimodels.GettableGrafanaReceiver, 0, len(postableReceiver.GrafanaManagedReceivers))

		for _, postableGrafanaReceiver := range postableReceiver.GrafanaManagedReceivers {
			secureFields := make(map[string]bool, len(postableGrafanaReceiver.SecureSettings))
			for key := range postableGrafanaReceiver.SecureSettings {
				secureFields[key] = true
			}

			gettableGrafanaReceiver := &apimodels.GettableGrafanaReceiver{
				UID:                   postableGrafanaReceiver.UID,
				Name:                  postableGrafanaReceiver.Name,
				Type:                  postableGrafanaReceiver.Type,
				DisableResolveMessage: postableGrafanaReceiver.DisableResolveMessage,
				Settings:              postableGrafanaReceiver.Settings,
				SecureFields:          secureFields,
			}
			gettableGrafanaReceivers = append(gettableGrafanaReceivers, gettableGrafanaReceiver)
		}

		gettableReceiver := &apimodels.GettableApiReceiver{
			GettableGrafanaReceivers: apimodels.GettableGrafanaReceivers{
				GrafanaManagedReceivers: gettableGrafanaReceivers,
			},
		}
		gettableReceiver.Name = postableReceiver.Name

		gettableReceivers = append(gettableReceivers, gettableReceiver)
	}

	return gettableReceivers
}

func convertGettableToPostableReceivers(gettableReceivers []*apimodels.GettableApiReceiver) []*apimodels.PostableApiReceiver {
	if gettableReceivers == nil {
		return nil
	}

	postableReceivers := make([]*apimodels.PostableApiReceiver, 0, len(gettableReceivers))

	for _, gettableReceiver := range gettableReceivers {
		postableGrafanaReceivers := make([]*apimodels.PostableGrafanaReceiver, 0, len(gettableReceiver.GrafanaManagedReceivers))

		for _, gettableGrafanaReceiver := range gettableReceiver.GrafanaManagedReceivers {
			secureSettings := make(map[string]string, len(gettableGrafanaReceiver.SecureFields))
			for key, isSecure := range gettableGrafanaReceiver.SecureFields {
				if isSecure {
					secureSettings[key] = ""
				}
			}

			postableGrafanaReceiver := &apimodels.PostableGrafanaReceiver{
				UID:                   gettableGrafanaReceiver.UID,
				Name:                  gettableGrafanaReceiver.Name,
				Type:                  gettableGrafanaReceiver.Type,
				DisableResolveMessage: gettableGrafanaReceiver.DisableResolveMessage,
				Settings:              gettableGrafanaReceiver.Settings,
				SecureSettings:        secureSettings,
			}
			postableGrafanaReceivers = append(postableGrafanaReceivers, postableGrafanaReceiver)
		}

		postableReceiver := &apimodels.PostableApiReceiver{
			PostableGrafanaReceivers: apimodels.PostableGrafanaReceivers{
				GrafanaManagedReceivers: postableGrafanaReceivers,
			},
		}
		postableReceiver.Name = gettableReceiver.Name

		postableReceivers = append(postableReceivers, postableReceiver)
	}

	return postableReceivers
}

type ConvertService interface {
	RouteConvertPrometheusGetAlertmanagerConfig(ctx *contextmodel.ReqContext) response.Response
}

type AlertmanagerApiHandler struct {
	AMSvc           *LotexAM
	GrafanaSvc      *AlertmanagerSrv
	ConvertSvc      ConvertService
	DatasourceCache datasources.CacheService
	FeatureManager  featuremgmt.FeatureToggles
}

// NewForkingAM implements a set of routes that proxy to various Alertmanager-compatible backends.
func NewForkingAM(datasourceCache datasources.CacheService, proxy *LotexAM, grafana *AlertmanagerSrv, convertSvc ConvertService, featureManager featuremgmt.FeatureToggles) *AlertmanagerApiHandler {
	return &AlertmanagerApiHandler{
		AMSvc:           proxy,
		GrafanaSvc:      grafana,
		ConvertSvc:      convertSvc,
		DatasourceCache: datasourceCache,
		FeatureManager:  featureManager,
	}
}

func (f *AlertmanagerApiHandler) getService(ctx *contextmodel.ReqContext) (*LotexAM, error) {
	// If this is not an extra config request, we should check that the datasource exists and is of the correct type.
	if isExtra, _ := f.isExtraConfig(ctx); !isExtra {
		_, err := getDatasourceByUID(ctx, f.DatasourceCache, apimodels.AlertmanagerBackend)
		if err != nil {
			return nil, err
		}
	}

	return f.AMSvc, nil
}

// isExtraConfig checks if the datasourceUID represents an extra config.
// Extra configs are the alertmanager configurations that were saved using the Prometheus conversion API.
func (f *AlertmanagerApiHandler) isExtraConfig(ctx *contextmodel.ReqContext) (bool, string) {
	// Only enabled if feature flag is on
	if !f.FeatureManager.IsEnabledGlobally(featuremgmt.FlagAlertingImportAlertmanagerUI) {
		return false, ""
	}

	datasourceUID := web.Params(ctx.Req)[":DatasourceUID"]
	if strings.HasPrefix(datasourceUID, extraConfigPrefix) {
		identifier := strings.TrimPrefix(datasourceUID, extraConfigPrefix)
		return true, identifier
	}
	return false, ""
}

func (f *AlertmanagerApiHandler) handleRouteGetAMStatus(ctx *contextmodel.ReqContext, dsUID string) response.Response {
	if isExtra, _ := f.isExtraConfig(ctx); isExtra {
		status := apimodels.GettableStatus{
			Cluster: &amv2.ClusterStatus{
				Status: util.Pointer("ready"),
			},
		}
		return response.JSON(http.StatusOK, status)
	}

	s, err := f.getService(ctx)
	if err != nil {
		return errorToResponse(err)
	}

	return s.RouteGetAMStatus(ctx)
}

func (f *AlertmanagerApiHandler) handleRouteCreateSilence(ctx *contextmodel.ReqContext, body apimodels.PostableSilence, dsUID string) response.Response {
	if isExtra, _ := f.isExtraConfig(ctx); isExtra {
		return response.Error(http.StatusForbidden, "Read-only configuration", nil)
	}

	s, err := f.getService(ctx)
	if err != nil {
		return errorToResponse(err)
	}

	return s.RouteCreateSilence(ctx, body)
}

func (f *AlertmanagerApiHandler) handleRouteDeleteAlertingConfig(ctx *contextmodel.ReqContext, dsUID string) response.Response {
	if isExtra, _ := f.isExtraConfig(ctx); isExtra {
		return response.Error(http.StatusForbidden, "Read-only configuration", nil)
	}

	s, err := f.getService(ctx)
	if err != nil {
		return errorToResponse(err)
	}

	return s.RouteDeleteAlertingConfig(ctx)
}

func (f *AlertmanagerApiHandler) handleRouteDeleteSilence(ctx *contextmodel.ReqContext, silenceID string, dsUID string) response.Response {
	if isExtra, _ := f.isExtraConfig(ctx); isExtra {
		return response.Error(http.StatusForbidden, "Read-only configuration", nil)
	}

	s, err := f.getService(ctx)
	if err != nil {
		return errorToResponse(err)
	}

	return s.RouteDeleteSilence(ctx, silenceID)
}

func (f *AlertmanagerApiHandler) handleRouteGetAlertingConfig(ctx *contextmodel.ReqContext, dsUID string) response.Response {
	if isExtra, identifier := f.isExtraConfig(ctx); isExtra {
		// Get the full Grafana configuration with the requested extra config
		canSeeAutogen := ctx.HasRole(org.RoleAdmin)
		config, err := f.GrafanaSvc.mam.GetAlertmanagerConfiguration(ctx.Req.Context(), ctx.GetOrgID(), canSeeAutogen)
		if err != nil {
			return response.Error(http.StatusInternalServerError, "Failed to get alertmanager config", err)
		}

		var foundExtraConfig *apimodels.ExtraConfiguration
		for i := range config.ExtraConfigs {
			if config.ExtraConfigs[i].Identifier == identifier {
				foundExtraConfig = &config.ExtraConfigs[i]
				break
			}
		}

		if foundExtraConfig == nil {
			return response.Error(http.StatusNotFound, "Extra configuration not found", nil)
		}

		postableConfig := apimodels.PostableUserConfig{
			TemplateFiles: config.TemplateFiles,
			AlertmanagerConfig: apimodels.PostableApiAlertingConfig{
				Config:    config.AlertmanagerConfig.Config,
				Receivers: convertGettableToPostableReceivers(config.AlertmanagerConfig.Receivers),
			},
			ExtraConfigs: []apimodels.ExtraConfiguration{*foundExtraConfig},
		}

		mergeResult, err := postableConfig.GetMergedAlertmanagerConfig()
		if err != nil {
			return response.Error(http.StatusInternalServerError, "Failed to merge configuration", err)
		}

		// TODO: kind
		mergedTemplates := postableConfig.GetMergedTemplateDefinitions()
		mergedTemplateFiles := make(map[string]string, len(mergedTemplates))
		for _, t := range mergedTemplates {
			mergedTemplateFiles[t.Name] = t.Content
		}

		convertedReceivers := convertPostableToGettableReceivers(mergeResult.Config.Receivers)

		result := apimodels.GettableUserConfig{
			TemplateFiles: mergedTemplateFiles,
			AlertmanagerConfig: apimodels.GettableApiAlertingConfig{
				Config:    mergeResult.Config.Config,
				Receivers: convertedReceivers,
			},
		}

		return response.JSON(http.StatusOK, result)
	}

	s, err := f.getService(ctx)
	if err != nil {
		return errorToResponse(err)
	}

	return s.RouteGetAlertingConfig(ctx)
}

func (f *AlertmanagerApiHandler) handleRouteGetAMAlertGroups(ctx *contextmodel.ReqContext, dsUID string) response.Response {
	if isExtra, _ := f.isExtraConfig(ctx); isExtra {
		return f.GrafanaSvc.RouteGetAMAlertGroups(ctx)
	}

	s, err := f.getService(ctx)
	if err != nil {
		return errorToResponse(err)
	}

	return s.RouteGetAMAlertGroups(ctx)
}

func (f *AlertmanagerApiHandler) handleRouteGetAMAlerts(ctx *contextmodel.ReqContext, dsUID string) response.Response {
	if isExtra, _ := f.isExtraConfig(ctx); isExtra {
		return f.GrafanaSvc.RouteGetAMAlerts(ctx)
	}

	s, err := f.getService(ctx)
	if err != nil {
		return errorToResponse(err)
	}

	return s.RouteGetAMAlerts(ctx)
}

func (f *AlertmanagerApiHandler) handleRouteGetSilence(ctx *contextmodel.ReqContext, silenceID string, dsUID string) response.Response {
	if isExtra, _ := f.isExtraConfig(ctx); isExtra {
		return f.GrafanaSvc.RouteGetSilence(ctx, silenceID)
	}

	s, err := f.getService(ctx)
	if err != nil {
		return errorToResponse(err)
	}

	return s.RouteGetSilence(ctx, silenceID)
}

func (f *AlertmanagerApiHandler) handleRouteGetSilences(ctx *contextmodel.ReqContext, dsUID string) response.Response {
	if isExtra, _ := f.isExtraConfig(ctx); isExtra {
		return f.GrafanaSvc.RouteGetSilences(ctx)
	}

	s, err := f.getService(ctx)
	if err != nil {
		return errorToResponse(err)
	}

	return s.RouteGetSilences(ctx)
}

func (f *AlertmanagerApiHandler) handleRoutePostAlertingConfig(ctx *contextmodel.ReqContext, body apimodels.PostableUserConfig, dsUID string) response.Response {
	if isExtra, _ := f.isExtraConfig(ctx); isExtra {
		return response.Error(http.StatusForbidden, "Read-only configuration", nil)
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
	if isExtra, _ := f.isExtraConfig(ctx); isExtra {
		return response.Error(http.StatusForbidden, "Read-only configuration", nil)
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
