package historian

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/stretchr/testify/require"
)

// This function can be used for local testing, just remove the skip call.
func TestLokiHTTPClient(t *testing.T) {
	t.Skip()

	t.Run("smoke test pinging Loki", func(t *testing.T) {
		url, err := url.Parse("https://logs-prod-eu-west-0.grafana.net")
		require.NoError(t, err)

		client := newLokiClient(LokiConfig{
			Url: url,
		}, log.NewNopLogger())

		// Unauthorized request should fail against Grafana Cloud.
		err = client.ping()
		require.Error(t, err)

		client.cfg.BasicAuthUser = "<your_username>"
		client.cfg.BasicAuthPassword = "<your_password>"

		// When running on prem, you might need to set the tenant id,
		// so the x-scope-orgid header is set.
		// client.cfg.TenantID = "<your_tenant_id>"

		// Authorized request should not fail against Grafana Cloud.
		err = client.ping()
		require.NoError(t, err)
	})

	t.Run("smoke test querying Loki", func(t *testing.T) {
		url, err := url.Parse("https://logs-prod-eu-west-0.grafana.net")
		require.NoError(t, err)

		client := newLokiClient(LokiConfig{
			Url:               url,
			BasicAuthUser:     "<your_username>",
			BasicAuthPassword: "<your_password>",
		}, log.NewNopLogger())

		// When running on prem, you might need to set the tenant id,
		// so the x-scope-orgid header is set.
		// client.cfg.TenantID = "<your_tenant_id>"

		// Create an array of selectors that should be used for the
		// query.
		selectors := [][3]string{
			{"", "", ""},
		}

		// Define the query time range
		start := time.Now().UnixMilli() - 10000
		end := time.Now().UnixMilli()

		// Authorized request should not fail against Grafana Cloud.
		res, err := client.query(context.Background(), selectors, start, end)
		require.NoError(t, err)
		require.NotNil(t, res)
	})
}

func TestSelectorString(t *testing.T) {
	selectors := [][3]string{{"name", "=", "Bob"}, {"age", "=~", "30"}}
	expected := "{name=\"Bob\",age=~\"30\"}"
	result, err := selectorString(selectors)
	require.NoError(t, err)
	require.Equal(t, expected, result)

	selectors = [][3]string{{"name", "?", "Bob"}}
	expected = ""
	result, err = selectorString(selectors)
	require.Error(t, err)
	require.Equal(t, expected, result)

	selectors = [][3]string{}
	expected = "{}"
	result, err = selectorString(selectors)
	require.NoError(t, err)
	require.Equal(t, expected, result)
}
