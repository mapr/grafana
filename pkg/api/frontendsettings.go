package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/grafana/grafana/pkg/api/dtos"
	"github.com/grafana/grafana/pkg/models"
	"github.com/grafana/grafana/pkg/plugins"
	"github.com/grafana/grafana/pkg/services/accesscontrol"
	"github.com/grafana/grafana/pkg/services/datasources"
	"github.com/grafana/grafana/pkg/services/featuremgmt"
	"github.com/grafana/grafana/pkg/services/licensing"
	"github.com/grafana/grafana/pkg/services/pluginsettings"
	"github.com/grafana/grafana/pkg/services/secrets/kvstore"
	"github.com/grafana/grafana/pkg/services/thumbs"
	"github.com/grafana/grafana/pkg/setting"
	"github.com/grafana/grafana/pkg/tsdb/grafanads"
	"github.com/grafana/grafana/pkg/util"
)

type FrontendSettingsAuthDTO struct {
	OAuthSkipOrgRoleUpdateSync bool `json:"OAuthSkipOrgRoleUpdateSync"`
	SAMLSkipOrgRoleSync        bool `json:"SAMLSkipOrgRoleSync"`
	LDAPSkipOrgRoleSync        bool `json:"LDAPSkipOrgRoleSync"`
	GoogleSkipOrgRoleSync      bool `json:"GoogleSkipOrgRoleSync"`
	JWTAuthSkipOrgRoleSync     bool `json:"JWTAuthSkipOrgRoleSync"`
	GrafanaComSkipOrgRoleSync  bool `json:"GrafanaComSkipOrgRoleSync"`
	AzureADSkipOrgRoleSync     bool `json:"AzureADSkipOrgRoleSync"`
	GithubSkipOrgRoleSync      bool `json:"GithubSkipOrgRoleSync"`
	GitLabSkipOrgRoleSync      bool `json:"GitLabSkipOrgRoleSync"`
	DisableSyncLock            bool `json:"DisableSyncLock"`
}

type FrontendSettingsBuildInfoDTO struct {
	HideVersion   bool   `json:"hideVersion"`
	Version       string `json:"version"`
	Commit        string `json:"commit"`
	Buildstamp    int64  `json:"buildstamp"`
	Edition       string `json:"edition"`
	LatestVersion string `json:"latestVersion"`
	HasUpdate     bool   `json:"hasUpdate"`
	Env           string `json:"env"`
}

type FrontendSettingsLicenseInfoDTO struct {
	Expiry          int64           `json:"expiry"`
	StateInfo       string          `json:"stateInfo"`
	LicenseUrl      string          `json:"licenseUrl"`
	Edition         string          `json:"edition"`
	EnabledFeatures map[string]bool `json:"enabledFeatures"`
}

type FrontendSettingsAzureDTO struct {
	Cloud                  string `json:"cloud"`
	ManagedIdentityEnabled bool   `json:"managedIdentityEnabled"`
}

type FrontendSettingsCachingDTO struct {
	Enabled bool `json:"enabled"`
}

type FrontendSettingsRecordedQueriesDTO struct {
	Enabled bool `json:"enabled"`
}

type FrontendSettingsReportingDTO struct {
	Enabled bool `json:"enabled"`
}

type FrontendSettingsUnifiedAlertingDTO struct {
	MinInterval string `json:"minInterval"`
}

type FrontendSettingsDTO struct {
	DefaultDatasource          string                           `json:"defaultDatasource"`
	Datasources                map[string]plugins.DataSourceDTO `json:"datasources"`
	MinRefreshInterval         string                           `json:"minRefreshInterval"`
	Panels                     map[string]plugins.PanelDTO      `json:"panels"`
	AppUrl                     string                           `json:"appUrl"`
	AppSubUrl                  string                           `json:"appSubUrl"`
	AllowOrgCreate             bool                             `json:"allowOrgCreate"`
	AuthProxyEnabled           bool                             `json:"authProxyEnabled"`
	LdapEnabled                bool                             `json:"ldapEnabled"`
	JwtHeaderName              string                           `json:"jwtHeaderName"`
	JwtUrlLogin                bool                             `json:"jwtUrlLogin"`
	AlertingEnabled            *bool                            `json:"alertingEnabled"`
	AlertingErrorOrTimeout     string                           `json:"alertingErrorOrTimeout"`
	AlertingNoDataOrNullValues string                           `json:"alertingNoDataOrNullValues"`
	AlertingMinInterval        int64                            `json:"alertingMinInterval"`
	LiveEnabled                bool                             `json:"liveEnabled"`
	AutoAssignOrg              bool                             `json:"autoAssignOrg"`

	VerifyEmailEnabled  bool `json:"verifyEmailEnabled"`
	SigV4AuthEnabled    bool `json:"sigV4AuthEnabled"`
	AzureAuthEnabled    bool `json:"azureAuthEnabled"`
	RbacEnabled         bool `json:"rbacEnabled"`
	ExploreEnabled      bool `json:"exploreEnabled"`
	HelpEnabled         bool `json:"helpEnabled"`
	ProfileEnabled      bool `json:"profileEnabled"`
	QueryHistoryEnabled bool `json:"queryHistoryEnabled"`

	GoogleAnalyticsId                   string `json:"googleAnalyticsId"`
	GoogleAnalytics4Id                  string `json:"googleAnalytics4Id"`
	GoogleAnalytics4SendManualPageViews bool   `json:"GoogleAnalytics4SendManualPageViews"`

	RudderstackWriteKey     string `json:"rudderstackWriteKey"`
	RudderstackDataPlaneUrl string `json:"rudderstackDataPlaneUrl"`
	RudderstackSdkUrl       string `json:"rudderstackSdkUrl"`
	RudderstackConfigUrl    string `json:"rudderstackConfigUrl"`

	FeedbackLinksEnabled                bool                     `json:"feedbackLinksEnabled"`
	ApplicationInsightsConnectionString string                   `json:"applicationInsightsConnectionString"`
	ApplicationInsightsEndpointUrl      string                   `json:"applicationInsightsEndpointUrl"`
	DisableLoginForm                    bool                     `json:"disableLoginForm"`
	DisableUserSignUp                   bool                     `json:"disableUserSignUp"`
	LoginHint                           string                   `json:"loginHint"`
	PasswordHint                        string                   `json:"passwordHint"`
	ExternalUserMngInfo                 string                   `json:"externalUserMngInfo"`
	ExternalUserMngLinkUrl              string                   `json:"externalUserMngLinkUrl"`
	ExternalUserMngLinkName             string                   `json:"externalUserMngLinkName"`
	ViewersCanEdit                      bool                     `json:"viewersCanEdit"`
	AngularSupportEnabled               bool                     `json:"angularSupportEnabled"`
	EditorsCanAdmin                     bool                     `json:"editorsCanAdmin"`
	DisableSanitizeHtml                 bool                     `json:"disableSanitizeHtml"`
	PluginsToPreload                    []*plugins.PreloadPlugin `json:"pluginsToPreload"`

	Auth FrontendSettingsAuthDTO `json:"auth"`

	BuildInfo FrontendSettingsBuildInfoDTO `json:"buildInfo"`

	LicenseInfo FrontendSettingsLicenseInfoDTO `json:"licenseInfo"`

	FeatureToggles                   map[string]bool                `json:"featureToggles"`
	RendererAvailable                bool                           `json:"rendererAvailable"`
	RendererVersion                  string                         `json:"rendererVersion"`
	SecretsManagerPluginEnabled      bool                           `json:"secretsManagerPluginEnabled"`
	Http2Enabled                     bool                           `json:"http2Enabled"`
	Sentry                           setting.Sentry                 `json:"sentry"`
	GrafanaJavascriptAgent           setting.GrafanaJavascriptAgent `json:"grafanaJavascriptAgent"`
	PluginCatalogURL                 string                         `json:"pluginCatalogURL"`
	PluginAdminEnabled               bool                           `json:"pluginAdminEnabled"`
	PluginAdminExternalManageEnabled bool                           `json:"pluginAdminExternalManageEnabled"`
	PluginCatalogHiddenPlugins       []string                       `json:"pluginCatalogHiddenPlugins"`
	ExpressionsEnabled               bool                           `json:"expressionsEnabled"`
	AwsAllowedAuthProviders          []string                       `json:"awsAllowedAuthProviders"`
	AwsAssumeRoleEnabled             bool                           `json:"awsAssumeRoleEnabled"`
	SupportBundlesEnabled            bool                           `json:"supportBundlesEnabled"`
	SnapshotEnabled                  bool                           `json:"snapshotEnabled"`

	Azure FrontendSettingsAzureDTO `json:"azure"`

	Caching                 FrontendSettingsCachingDTO         `json:"caching"`
	RecordedQueries         FrontendSettingsRecordedQueriesDTO `json:"recordedQueries"`
	Reporting               FrontendSettingsReportingDTO       `json:"reporting"`
	UnifiedAlertingEnabled  *bool                              `json:"unifiedAlertingEnabled"`
	UnifiedAlerting         FrontendSettingsUnifiedAlertingDTO `json:"unifiedAlerting"`
	Oauth                   map[string]interface{}             `json:"oauth"`
	SamlEnabled             bool                               `json:"samlEnabled"`
	SamlName                string                             `json:"samlName"`
	TokenExpirationDayLimit int                                `json:"tokenExpirationDayLimit"`

	DashboardPreviews *thumbs.DashboardPreviewsSetupConfig `json:"dashboardPreviews"`

	GeomapDefaultBaseLayerConfig *map[string]interface{} `json:"geomapDefaultBaseLayerConfig"`
	GeomapDisableCustomBaseLayer bool                    `json:"geomapDisableCustomBaseLayer"`
}

func (hs *HTTPServer) GetFrontendSettings(c *models.ReqContext) {
	settings, err := hs.getFrontendSettingsMap(c)
	if err != nil {
		c.JsonApiErr(400, "Failed to get frontend settings", err)
		return
	}

	c.JSON(http.StatusOK, settings)
}

// getFrontendSettingsMap returns a json object with all the settings needed for front end initialisation.
func (hs *HTTPServer) getFrontendSettingsMap(c *models.ReqContext) (*FrontendSettingsDTO, error) {
	enabledPlugins, err := hs.enabledPlugins(c.Req.Context(), c.OrgID)
	if err != nil {
		return nil, err
	}

	pluginsToPreload := make([]*plugins.PreloadPlugin, 0)
	for _, app := range enabledPlugins[plugins.App] {
		if app.Preload {
			pluginsToPreload = append(pluginsToPreload, &plugins.PreloadPlugin{
				Path:    app.Module,
				Version: app.Info.Version,
			})
		}
	}

	dataSources, err := hs.getFSDataSources(c, enabledPlugins)
	if err != nil {
		return nil, err
	}

	defaultDS := "-- Grafana --"
	for n, ds := range dataSources {
		if ds.IsDefault {
			defaultDS = n
		}
	}

	panels := make(map[string]plugins.PanelDTO)
	for _, panel := range enabledPlugins[plugins.Panel] {
		if panel.State == plugins.AlphaRelease && !hs.Cfg.PluginsEnableAlpha {
			continue
		}

		hideFromList := panel.HideFromList
		if panel.ID == "flamegraph" {
			hideFromList = !hs.Features.IsEnabled(featuremgmt.FlagFlameGraph)
		}

		panels[panel.ID] = plugins.PanelDTO{
			ID:            panel.ID,
			Name:          panel.Name,
			Info:          panel.Info,
			Module:        panel.Module,
			BaseURL:       panel.BaseURL,
			SkipDataQuery: panel.SkipDataQuery,
			HideFromList:  hideFromList,
			ReleaseState:  string(panel.State),
			Signature:     string(panel.Signature),
			Sort:          getPanelSort(panel.ID),
		}
	}

	hideVersion := hs.Cfg.AnonymousHideVersion && !c.IsSignedIn
	version := setting.BuildVersion
	commit := setting.BuildCommit
	buildstamp := setting.BuildStamp

	if hideVersion {
		version = ""
		commit = ""
		buildstamp = 0
	}

	hasAccess := accesscontrol.HasAccess(hs.AccessControl, c)
	secretsManagerPluginEnabled := kvstore.EvaluateRemoteSecretsPlugin(c.Req.Context(), hs.secretsPluginManager, hs.Cfg) == nil

	frontendSettings := &dtos.FrontendSettingsDTO{
		DefaultDatasource:                   defaultDS,
		Datasources:                         dataSources,
		MinRefreshInterval:                  setting.MinRefreshInterval,
		Panels:                              panels,
		AppUrl:                              hs.Cfg.AppURL,
		AppSubUrl:                           hs.Cfg.AppSubURL,
		AllowOrgCreate:                      (setting.AllowUserOrgCreate && c.IsSignedIn) || c.IsGrafanaAdmin,
		AuthProxyEnabled:                    setting.AuthProxyEnabled,
		LdapEnabled:                         hs.Cfg.LDAPEnabled,
		JwtHeaderName:                       hs.Cfg.JWTAuthHeaderName,
		JwtUrlLogin:                         hs.Cfg.JWTAuthURLLogin,
		AlertingEnabled:                     setting.AlertingEnabled,
		AlertingErrorOrTimeout:              setting.AlertingErrorOrTimeout,
		AlertingNoDataOrNullValues:          setting.AlertingNoDataOrNullValues,
		AlertingMinInterval:                 setting.AlertingMinInterval,
		LiveEnabled:                         hs.Cfg.LiveMaxConnections != 0,
		AutoAssignOrg:                       setting.AutoAssignOrg,
		VerifyEmailEnabled:                  setting.VerifyEmailEnabled,
		SigV4AuthEnabled:                    setting.SigV4AuthEnabled,
		AzureAuthEnabled:                    setting.AzureAuthEnabled,
		RbacEnabled:                         hs.Cfg.RBACEnabled,
		ExploreEnabled:                      setting.ExploreEnabled,
		HelpEnabled:                         setting.HelpEnabled,
		ProfileEnabled:                      setting.ProfileEnabled,
		QueryHistoryEnabled:                 hs.Cfg.QueryHistoryEnabled,
		GoogleAnalyticsId:                   setting.GoogleAnalyticsId,
		GoogleAnalytics4Id:                  setting.GoogleAnalytics4Id,
		GoogleAnalytics4SendManualPageViews: setting.GoogleAnalytics4SendManualPageViews,
		RudderstackWriteKey:                 setting.RudderstackWriteKey,
		RudderstackDataPlaneUrl:             setting.RudderstackDataPlaneUrl,
		RudderstackSdkUrl:                   setting.RudderstackSdkUrl,
		RudderstackConfigUrl:                setting.RudderstackConfigUrl,
		FeedbackLinksEnabled:                hs.Cfg.FeedbackLinksEnabled,
		ApplicationInsightsConnectionString: hs.Cfg.ApplicationInsightsConnectionString,
		ApplicationInsightsEndpointUrl:      hs.Cfg.ApplicationInsightsEndpointUrl,
		DisableLoginForm:                    setting.DisableLoginForm,
		DisableUserSignUp:                   !setting.AllowUserSignUp,
		LoginHint:                           setting.LoginHint,
		PasswordHint:                        setting.PasswordHint,
		ExternalUserMngInfo:                 setting.ExternalUserMngInfo,
		ExternalUserMngLinkUrl:              setting.ExternalUserMngLinkUrl,
		ExternalUserMngLinkName:             setting.ExternalUserMngLinkName,
		ViewersCanEdit:                      setting.ViewersCanEdit,
		AngularSupportEnabled:               hs.Cfg.AngularSupportEnabled,
		EditorsCanAdmin:                     hs.Cfg.EditorsCanAdmin,
		DisableSanitizeHtml:                 hs.Cfg.DisableSanitizeHtml,
		PluginsToPreload:                    pluginsToPreload,

		Auth: setting.FrontendSettingsAuthDTO{
			OAuthSkipOrgRoleUpdateSync: hs.Cfg.OAuthSkipOrgRoleUpdateSync,
			SAMLSkipOrgRoleSync:        hs.Cfg.SectionWithEnvOverrides("auth.saml").Key("skip_org_role_sync").MustBool(false),
			LDAPSkipOrgRoleSync:        hs.Cfg.LDAPSkipOrgRoleSync,
			GoogleSkipOrgRoleSync:      hs.Cfg.GoogleSkipOrgRoleSync,
			JWTAuthSkipOrgRoleSync:     hs.Cfg.JWTAuthSkipOrgRoleSync,
			GrafanaComSkipOrgRoleSync:  hs.Cfg.GrafanaComSkipOrgRoleSync,
			AzureADSkipOrgRoleSync:     hs.Cfg.AzureADSkipOrgRoleSync,
			GithubSkipOrgRoleSync:      hs.Cfg.GithubSkipOrgRoleSync,
			GitLabSkipOrgRoleSync:      hs.Cfg.GitLabSkipOrgRoleSync,
			DisableSyncLock:            hs.Cfg.DisableSyncLock,
		},

		BuildInfo: setting.FrontendSettingsBuildInfoDTO{
			HideVersion:   hideVersion,
			Version:       version,
			Commit:        commit,
			Buildstamp:    buildstamp,
			Edition:       hs.License.Edition(),
			LatestVersion: hs.grafanaUpdateChecker.LatestVersion(),
			HasUpdate:     hs.grafanaUpdateChecker.UpdateAvailable(),
			Env:           setting.Env,
		},

		LicenseInfo: setting.FrontendSettingsLicenseInfoDTO{
			Expiry:          hs.License.Expiry(),
			StateInfo:       hs.License.StateInfo(),
			LicenseUrl:      hs.License.LicenseURL(hasAccess(accesscontrol.ReqGrafanaAdmin, licensing.PageAccess)),
			Edition:         hs.License.Edition(),
			EnabledFeatures: hs.License.EnabledFeatures(),
		},

		FeatureToggles:                   hs.Features.GetEnabled(c.Req.Context()),
		RendererAvailable:                hs.RenderService.IsAvailable(c.Req.Context()),
		RendererVersion:                  hs.RenderService.Version(),
		SecretsManagerPluginEnabled:      secretsManagerPluginEnabled,
		Http2Enabled:                     hs.Cfg.Protocol == setting.HTTP2Scheme,
		Sentry:                           hs.Cfg.Sentry,
		GrafanaJavascriptAgent:           hs.Cfg.GrafanaJavascriptAgent,
		PluginCatalogURL:                 hs.Cfg.PluginCatalogURL,
		PluginAdminEnabled:               hs.Cfg.PluginAdminEnabled,
		PluginAdminExternalManageEnabled: hs.Cfg.PluginAdminEnabled && hs.Cfg.PluginAdminExternalManageEnabled,
		PluginCatalogHiddenPlugins:       hs.Cfg.PluginCatalogHiddenPlugins,
		ExpressionsEnabled:               hs.Cfg.ExpressionsEnabled,
		AwsAllowedAuthProviders:          hs.Cfg.AWSAllowedAuthProviders,
		AwsAssumeRoleEnabled:             hs.Cfg.AWSAssumeRoleEnabled,
		SupportBundlesEnabled:            isSupportBundlesEnabled(hs),

		Azure: setting.FrontendSettingsAzureDTO{
			Cloud:                  hs.Cfg.Azure.Cloud,
			ManagedIdentityEnabled: hs.Cfg.Azure.ManagedIdentityEnabled,
		},

		Caching: setting.FrontendSettingsCachingDTO{
			Enabled: hs.Cfg.SectionWithEnvOverrides("caching").Key("enabled").MustBool(true),
		},
		RecordedQueries: setting.FrontendSettingsRecordedQueriesDTO{
			Enabled: hs.Cfg.SectionWithEnvOverrides("recorded_queries").Key("enabled").MustBool(true),
		},
		Reporting: setting.FrontendSettingsReportingDTO{
			Enabled: hs.Cfg.SectionWithEnvOverrides("reporting").Key("enabled").MustBool(true),
		},

		UnifiedAlertingEnabled: hs.Cfg.UnifiedAlerting.Enabled,
		UnifiedAlerting: setting.FrontendSettingsUnifiedAlertingDTO{
			MinInterval: hs.Cfg.UnifiedAlerting.MinInterval.String(),
		},

		Oauth:                   hs.getEnabledOAuthProviders(),
		SamlEnabled:             hs.samlEnabled(),
		SamlName:                hs.samlName(),
		TokenExpirationDayLimit: hs.Cfg.SATokenExpirationDayLimit,

		SnapshotEnabled: hs.Cfg.SnapshotEnabled,
	}

	if hs.ThumbService != nil {
		frontendSettings.DashboardPreviews = hs.ThumbService.GetDashboardPreviewsSetupSettings(c)
	}

	if hs.Cfg.GeomapDefaultBaseLayerConfig != nil {
		frontendSettings.GeomapDefaultBaseLayerConfig = &hs.Cfg.GeomapDefaultBaseLayerConfig
	}
	if !hs.Cfg.GeomapEnableCustomBaseLayers {
		frontendSettings.GeomapDisableCustomBaseLayer = true
	}

	return frontendSettings, nil
}

func isSupportBundlesEnabled(hs *HTTPServer) bool {
	return hs.Cfg.SectionWithEnvOverrides("support_bundles").Key("enabled").MustBool(false) &&
		hs.Features.IsEnabled(featuremgmt.FlagSupportBundles)
}

func (hs *HTTPServer) getFSDataSources(c *models.ReqContext, enabledPlugins EnabledPlugins) (map[string]plugins.DataSourceDTO, error) {
	orgDataSources := make([]*datasources.DataSource, 0)
	if c.OrgID != 0 {
		query := datasources.GetDataSourcesQuery{OrgId: c.OrgID, DataSourceLimit: hs.Cfg.DataSourceLimit}
		err := hs.DataSourcesService.GetDataSources(c.Req.Context(), &query)
		if err != nil {
			return nil, err
		}

		if c.IsPublicDashboardView {
			// If RBAC is enabled, it will filter out all datasources for a public user, so we need to skip it
			orgDataSources = query.Result
		} else {
			filtered, err := hs.filterDatasourcesByQueryPermission(c.Req.Context(), c.SignedInUser, query.Result)
			if err != nil {
				return nil, err
			}
			orgDataSources = filtered
		}
	}

	dataSources := make(map[string]plugins.DataSourceDTO)

	for _, ds := range orgDataSources {
		url := ds.Url

		if ds.Access == datasources.DS_ACCESS_PROXY {
			url = "/api/datasources/proxy/" + strconv.FormatInt(ds.Id, 10)
		}

		dsDTO := plugins.DataSourceDTO{
			ID:        ds.Id,
			UID:       ds.Uid,
			Type:      ds.Type,
			Name:      ds.Name,
			URL:       url,
			IsDefault: ds.IsDefault,
			Access:    string(ds.Access),
			ReadOnly:  ds.ReadOnly,
		}

		plugin, exists := enabledPlugins.Get(plugins.DataSource, ds.Type)
		if !exists {
			c.Logger.Error("Could not find plugin definition for data source", "datasource_type", ds.Type)
			continue
		}
		dsDTO.Preload = plugin.Preload
		dsDTO.Module = plugin.Module
		dsDTO.PluginMeta = &plugins.PluginMetaDTO{
			JSONData:  plugin.JSONData,
			Signature: plugin.Signature,
			Module:    plugin.Module,
			BaseURL:   plugin.BaseURL,
		}

		if ds.JsonData == nil {
			dsDTO.JSONData = make(map[string]interface{})
		} else {
			dsDTO.JSONData = ds.JsonData.MustMap()
		}

		if ds.Access == datasources.DS_ACCESS_DIRECT {
			if ds.BasicAuth {
				password, err := hs.DataSourcesService.DecryptedBasicAuthPassword(c.Req.Context(), ds)
				if err != nil {
					return nil, err
				}

				dsDTO.BasicAuth = util.GetBasicAuthHeader(
					ds.BasicAuthUser,
					password,
				)
			}
			if ds.WithCredentials {
				dsDTO.WithCredentials = ds.WithCredentials
			}

			if ds.Type == datasources.DS_INFLUXDB_08 {
				password, err := hs.DataSourcesService.DecryptedPassword(c.Req.Context(), ds)
				if err != nil {
					return nil, err
				}

				dsDTO.Username = ds.User
				dsDTO.Password = password
				dsDTO.URL = url + "/db/" + ds.Database
			}

			if ds.Type == datasources.DS_INFLUXDB {
				password, err := hs.DataSourcesService.DecryptedPassword(c.Req.Context(), ds)
				if err != nil {
					return nil, err
				}

				dsDTO.Username = ds.User
				dsDTO.Password = password
				dsDTO.URL = url
			}
		}

		if (ds.Type == datasources.DS_INFLUXDB) || (ds.Type == datasources.DS_ES) {
			dsDTO.Database = ds.Database
		}

		if ds.Type == datasources.DS_PROMETHEUS {
			// add unproxied server URL for link to Prometheus web UI
			ds.JsonData.Set("directUrl", ds.Url)
		}

		dataSources[ds.Name] = dsDTO
	}

	// add data sources that are built in (meaning they are not added via data sources page, nor have any entry in
	// the datasource table)
	for _, ds := range hs.pluginStore.Plugins(c.Req.Context(), plugins.DataSource) {
		if ds.BuiltIn {
			dto := plugins.DataSourceDTO{
				Type:     string(ds.Type),
				Name:     ds.Name,
				JSONData: make(map[string]interface{}),
				PluginMeta: &plugins.PluginMetaDTO{
					JSONData:  ds.JSONData,
					Signature: ds.Signature,
					Module:    ds.Module,
					BaseURL:   ds.BaseURL,
				},
			}
			if ds.Name == grafanads.DatasourceName {
				dto.ID = grafanads.DatasourceID
				dto.UID = grafanads.DatasourceUID
			}
			dataSources[ds.Name] = dto
		}
	}

	return dataSources, nil
}

func getPanelSort(id string) int {
	sort := 100
	switch id {
	case "timeseries":
		sort = 1
	case "barchart":
		sort = 2
	case "stat":
		sort = 3
	case "gauge":
		sort = 4
	case "bargauge":
		sort = 5
	case "table":
		sort = 6
	case "singlestat":
		sort = 7
	case "piechart":
		sort = 8
	case "state-timeline":
		sort = 9
	case "heatmap":
		sort = 10
	case "status-history":
		sort = 11
	case "histogram":
		sort = 12
	case "graph":
		sort = 13
	case "text":
		sort = 14
	case "alertlist":
		sort = 15
	case "dashlist":
		sort = 16
	case "news":
		sort = 17
	}
	return sort
}

// EnabledPlugins represents a mapping from plugin types (panel, data source, etc.) to plugin IDs to plugins
// For example ["panel"] -> ["piechart"] -> {pie chart plugin DTO}
type EnabledPlugins map[plugins.Type]map[string]plugins.PluginDTO

func (ep EnabledPlugins) Get(pluginType plugins.Type, pluginID string) (plugins.PluginDTO, bool) {
	if _, exists := ep[pluginType][pluginID]; exists {
		return ep[pluginType][pluginID], true
	}

	return plugins.PluginDTO{}, false
}

func (hs *HTTPServer) enabledPlugins(ctx context.Context, orgID int64) (EnabledPlugins, error) {
	ep := make(EnabledPlugins)

	pluginSettingMap, err := hs.pluginSettings(ctx, orgID)
	if err != nil {
		return ep, err
	}

	apps := make(map[string]plugins.PluginDTO)
	for _, app := range hs.pluginStore.Plugins(ctx, plugins.App) {
		if b, exists := pluginSettingMap[app.ID]; exists {
			app.Pinned = b.Pinned
			apps[app.ID] = app
		}
	}
	ep[plugins.App] = apps

	dataSources := make(map[string]plugins.PluginDTO)
	for _, ds := range hs.pluginStore.Plugins(ctx, plugins.DataSource) {
		if _, exists := pluginSettingMap[ds.ID]; exists {
			dataSources[ds.ID] = ds
		}
	}
	ep[plugins.DataSource] = dataSources

	panels := make(map[string]plugins.PluginDTO)
	for _, p := range hs.pluginStore.Plugins(ctx, plugins.Panel) {
		if _, exists := pluginSettingMap[p.ID]; exists {
			panels[p.ID] = p
		}
	}
	ep[plugins.Panel] = panels

	return ep, nil
}

func (hs *HTTPServer) pluginSettings(ctx context.Context, orgID int64) (map[string]*pluginsettings.InfoDTO, error) {
	pluginSettings := make(map[string]*pluginsettings.InfoDTO)

	// fill settings from database
	if pss, err := hs.PluginSettings.GetPluginSettings(ctx, &pluginsettings.GetArgs{OrgID: orgID}); err != nil {
		return nil, err
	} else {
		for _, ps := range pss {
			pluginSettings[ps.PluginID] = ps
		}
	}

	// fill settings from app plugins
	for _, plugin := range hs.pluginStore.Plugins(ctx, plugins.App) {
		// ignore settings that already exist
		if _, exists := pluginSettings[plugin.ID]; exists {
			continue
		}

		// add new setting which is enabled depending on if AutoEnabled: true
		pluginSetting := &pluginsettings.InfoDTO{
			PluginID:      plugin.ID,
			OrgID:         orgID,
			Enabled:       plugin.AutoEnabled,
			Pinned:        plugin.AutoEnabled,
			PluginVersion: plugin.Info.Version,
		}

		pluginSettings[plugin.ID] = pluginSetting
	}

	// fill settings from all remaining plugins (including potential app child plugins)
	for _, plugin := range hs.pluginStore.Plugins(ctx) {
		// ignore settings that already exist
		if _, exists := pluginSettings[plugin.ID]; exists {
			continue
		}

		// add new setting which is enabled by default
		pluginSetting := &pluginsettings.InfoDTO{
			PluginID:      plugin.ID,
			OrgID:         orgID,
			Enabled:       true,
			Pinned:        false,
			PluginVersion: plugin.Info.Version,
		}

		// if plugin is included in an app, check app settings
		if plugin.IncludedInAppID != "" {
			// app child plugins are disabled unless app is enabled
			pluginSetting.Enabled = false
			if p, exists := pluginSettings[plugin.IncludedInAppID]; exists {
				pluginSetting.Enabled = p.Enabled
			}
		}
		pluginSettings[plugin.ID] = pluginSetting
	}

	return pluginSettings, nil
}

func (hs *HTTPServer) getEnabledOAuthProviders() map[string]interface{} {
	providers := make(map[string]interface{})
	for key, oauth := range hs.SocialService.GetOAuthInfoProviders() {
		providers[key] = map[string]string{
			"name": oauth.Name,
			"icon": oauth.Icon,
		}
	}
	return providers
}
