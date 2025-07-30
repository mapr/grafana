package usagestatssvcs

import (
	acdatabase "github.com/grafana/grafana/pkg/extensions/accesscontrol/database"
	"github.com/grafana/grafana/pkg/registry"
	"github.com/grafana/grafana/pkg/services/accesscontrol"
	"github.com/grafana/grafana/pkg/services/user"
)

func ProvideUsageStatsProvidersRegistry(
	accesscontrol accesscontrol.Service,
	acDB *acdatabase.AccessControlStore,
	user user.Service,
) *UsageStatsProvidersRegistry {
	// TODO this change breaks wire generation
	// `make gen-go` fails due to a class inheritance issue
	// wire: /Users/cory.forseth/dev/grafana/pkg/server/wireexts_enterprise.go:268:2: *github.com/grafana/grafana/pkg/extensions/secret/secretkeeper.EnterpriseKeeperService does not implement github.com/grafana/grafana/pkg/registry/apis/secret/contracts.KeeperService
	// maybe caused by `go mod tidy` failing
	// go: github.com/grafana/grafana/pkg/extensions/apiserver imports
	//	github.com/grafana/grafana/pkg/registry/apis/secret/worker: no matching versions for query "latest"
	return NewUsageStatsProvidersRegistry(
		accesscontrol,
		acDB,
		user,
	)
}

type UsageStatsProvidersRegistry struct {
	Services []registry.ProvidesUsageStats
}

func NewUsageStatsProvidersRegistry(services ...registry.ProvidesUsageStats) *UsageStatsProvidersRegistry {
	return &UsageStatsProvidersRegistry{services}
}

func (r *UsageStatsProvidersRegistry) GetServices() []registry.ProvidesUsageStats {
	return r.Services
}
