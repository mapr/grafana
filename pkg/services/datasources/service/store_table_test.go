package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana/pkg/infra/db"
	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/services/datasources"
	"github.com/grafana/grafana/pkg/services/quota"
	"github.com/grafana/grafana/pkg/storage/legacysql"
	"github.com/grafana/grafana/pkg/util/testutil"
)

// TestIntegrationStoreUsesProvidedTableName asserts every query resolves the table
// name through the LegacyDatabaseProvider rather than hardcoding "data_source".
func TestIntegrationStoreUsesProvidedTableName(t *testing.T) {
	testutil.SkipIntegrationTestInShortMode(t)

	sqlStore := db.InitTestDB(t) //nolint:staticcheck // legacy shared-DB test setup; migrate to NewTestStore
	ss := SqlStore{
		logger: log.NewNopLogger(),
		dbProvider: func(ctx context.Context) (*legacysql.LegacyDatabaseHelper, error) {
			return &legacysql.LegacyDatabaseHelper{
				DB:    sqlStore,
				Table: func(n string) string { return "tenant_" + n },
			}, nil
		},
	}

	ctx := context.Background()
	tests := map[string]func() error{
		"GetDataSource": func() error {
			_, err := ss.GetDataSource(ctx, &datasources.GetDataSourceQuery{OrgID: 1, UID: "uid"})
			return err
		},
		"GetDataSourceInNamespace": func() error {
			_, err := ss.GetDataSourceInNamespace(ctx, "default", "uid", "prometheus")
			return err
		},
		"GetDataSources": func() error {
			_, err := ss.GetDataSources(ctx, &datasources.GetDataSourcesQuery{OrgID: 1})
			return err
		},
		"GetDataSourcesWithLimit": func() error {
			_, err := ss.GetDataSources(ctx, &datasources.GetDataSourcesQuery{OrgID: 1, DataSourceLimit: 10})
			return err
		},
		"GetAllDataSources": func() error {
			_, err := ss.GetAllDataSources(ctx, &datasources.GetAllDataSourcesQuery{})
			return err
		},
		"GetDataSourcesByType": func() error {
			_, err := ss.GetDataSourcesByType(ctx, &datasources.GetDataSourcesByTypeQuery{Type: "prometheus"})
			return err
		},
		"GetDataSourcesByTypeInOrg": func() error {
			_, err := ss.GetDataSourcesByType(ctx, &datasources.GetDataSourcesByTypeQuery{Type: "prometheus", OrgID: 1})
			return err
		},
		"GetPrunableProvisionedDataSources": func() error {
			_, err := ss.GetPrunableProvisionedDataSources(ctx)
			return err
		},
		"AddDataSource": func() error {
			_, err := ss.AddDataSource(ctx, &datasources.AddDataSourceCommand{OrgID: 1, Name: "ds", Type: "prometheus"})
			return err
		},
		"UpdateDataSource": func() error {
			_, err := ss.UpdateDataSource(ctx, &datasources.UpdateDataSourceCommand{ID: 1, OrgID: 1, Name: "ds"})
			return err
		},
		"DeleteDataSource": func() error {
			return ss.DeleteDataSource(ctx, &datasources.DeleteDataSourceCommand{ID: 1, OrgID: 1})
		},
		"Count": func() error {
			_, err := ss.Count(ctx, nil)
			return err
		},
	}

	for name, run := range tests {
		t.Run(name, func(t *testing.T) {
			err := run()
			require.Error(t, err)
			require.ErrorContains(t, err, "tenant_data_source")
		})
	}
}

// TestIntegrationStoreUsesProvidedDatabase asserts every query runs on the connection
// from the LegacyDatabaseProvider. The store holds no connection of its own, so a tenant
// whose tables live on another server is queried on that server.
func TestIntegrationStoreUsesProvidedDatabase(t *testing.T) {
	testutil.SkipIntegrationTestInShortMode(t)

	tenant := db.NewTestStore(t)
	ss := CreateStore(log.NewNopLogger(), legacysql.NewDatabaseProvider(tenant))

	ctx := context.Background()
	countInTenant := func(t *testing.T) int64 {
		t.Helper()
		var count int64
		require.NoError(t, tenant.WithDbSession(ctx, func(sess *db.Session) error {
			var err error
			count, err = sess.Table("data_source").Count(&datasources.DataSource{})
			return err
		}))
		return count
	}

	ds, err := ss.AddDataSource(ctx, &datasources.AddDataSourceCommand{OrgID: 1, Name: "ds", Type: "prometheus"})
	require.NoError(t, err)
	require.Equal(t, int64(1), countInTenant(t))

	got, err := ss.GetDataSource(ctx, &datasources.GetDataSourceQuery{OrgID: 1, UID: ds.UID})
	require.NoError(t, err)
	require.Equal(t, ds.ID, got.ID) //nolint:staticcheck // Preserve legacy field compatibility.

	all, err := ss.GetDataSources(ctx, &datasources.GetDataSourcesQuery{OrgID: 1})
	require.NoError(t, err)
	require.Len(t, all, 1)

	quotas, err := ss.Count(ctx, &quota.ScopeParameters{OrgID: 1})
	require.NoError(t, err)
	orgTag, err := quota.NewTag(datasources.QuotaTargetSrv, datasources.QuotaTarget, quota.OrgScope)
	require.NoError(t, err)
	orgCount, ok := quotas.Get(orgTag)
	require.True(t, ok)
	require.Equal(t, int64(1), orgCount)

	_, err = ss.UpdateDataSource(ctx, &datasources.UpdateDataSourceCommand{ID: ds.ID, OrgID: 1, Name: "renamed", Type: "prometheus"}) //nolint:staticcheck // Preserve legacy field compatibility.
	require.NoError(t, err)

	require.NoError(t, ss.DeleteDataSource(ctx, &datasources.DeleteDataSourceCommand{UID: ds.UID, OrgID: 1}))
	require.Zero(t, countInTenant(t))
}
