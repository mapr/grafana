package builder

import (
	"context"
	"errors"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/server"
	"k8s.io/kube-openapi/pkg/common"

	"github.com/grafana/grafana/pkg/plugins"
	"github.com/grafana/grafana/pkg/plugins/manager/sources"
	"github.com/grafana/grafana/pkg/setting"
)

func ForPlugins(cfg *setting.Cfg, pluginClient plugins.Client) ([]APIGroupBuilder, error) {
	coreDataSourcesSrc := sources.NewLocalSource(
		plugins.ClassExternal,
		[]string{cfg.PluginsPath},
	)

	res, err := coreDataSourcesSrc.Discover(context.Background())
	if err != nil {
		return nil, errors.New("failed to load core data source plugins")
	}

	builders := []APIGroupBuilder{}
	for _, p := range res {
		if p.Primary.JSONData.Backend && p.Primary.JSONData.Type == plugins.TypeApp {
			fmt.Printf("%+v\n", p)
		}
	}
	return builders, nil
}

var _ APIGroupBuilder = (*appBuilder)(nil)

type appBuilder struct {
	jsonData     plugins.JSONData
	pluginClient plugins.Client
}

// AllowedV0Alpha1Resources implements builder.APIGroupBuilder.
func (a *appBuilder) AllowedV0Alpha1Resources() []string {
	return []string{}
}

// GetOpenAPIDefinitions implements builder.APIGroupBuilder.
func (a *appBuilder) GetOpenAPIDefinitions() common.GetOpenAPIDefinitions {
	panic("unimplemented")
}

// InstallSchema implements builder.APIGroupBuilder.
func (a *appBuilder) InstallSchema(scheme *runtime.Scheme) error {
	panic("unimplemented")
}

// UpdateAPIGroupInfo implements builder.APIGroupBuilder.
func (a *appBuilder) UpdateAPIGroupInfo(apiGroupInfo *server.APIGroupInfo, opts APIGroupOptions) error {
	panic("unimplemented")
}
