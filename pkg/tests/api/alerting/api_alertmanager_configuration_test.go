package alerting

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/grafana/grafana/pkg/models"
	"github.com/grafana/grafana/pkg/tests/testinfra"
	"github.com/stretchr/testify/require"
)

func TestAlertmanagerConfiguration(t *testing.T) {
	dir, path := testinfra.CreateGrafDir(t, testinfra.GrafanaOpts{
		EnableFeatureToggles: []string{"ngalert"},
		AnonymousUserRole:    models.ROLE_EDITOR,
	})

	store := testinfra.SetUpDatabase(t, dir)
	grafanaListedAddr := testinfra.StartGrafana(t, dir, path, store)

	// On a blank start with no configuration, it saves and delivers the default configuration.
	{
		alertConfigURL := fmt.Sprintf("http://%s/api/alertmanager/grafana/config/api/v1/alerts", grafanaListedAddr)
		// nolint:gosec
		resp, err := http.Get(alertConfigURL)
		require.NoError(t, err)
		t.Cleanup(func() {
			err := resp.Body.Close()
			require.NoError(t, err)
		})
		b, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode)
		require.JSONEq(t, `
{
	"template_files": null,
	"alertmanager_config": {
		"route": {
			"receiver": "grafana-default-email"
		},
		"templates": null,
		"receivers": [{
			"name": "grafana-default-email",
			"grafana_managed_receiver_configs": [{
				"id": 0,
				"uid": "",
				"name": "email receiver",
				"type": "email",
				"isDefault": true,
				"sendReminder": false,
				"disableResolveMessage": false,
				"frequency": "",
				"created": "0001-01-01T00:00:00Z",
				"updated": "0001-01-01T00:00:00Z",
				"settings": {
					"addresses": "\u003cexample@email.com\u003e"
				},
				"secureFields": {}
			}]
		}]
	}
}
`, string(b))
	}

	// When creating new configuration, if it fails to apply - it does not save it.
	{
		alertConfigURL := fmt.Sprintf("http://%s/api/alertmanager/grafana/config/api/v1/alerts", grafanaListedAddr)
		// nolint:gosec
		payload := `
{
	"template_files": {},
	"alertmanager_config": {
		"route": {
			"receiver": "slack.receiver"
		},
		"templates": null,
		"receivers": [{
			"name": "slack.receiver",
			"grafana_managed_receiver_configs": [{
				"settings": {
					"iconEmoji": "",
					"iconUrl": "",
					"mentionGroups": "",
					"mentionUsers": "",
					"recipient": "#unified-alerting-test",
					"username": ""
				},
				"secureSettings": {
					"token": "",
					"url": "https://hooks.slack.com/services/ASJDAHSDJA/ASDBJASDBJASDJH£4/asdjahsdjahsjdas981392812391231bb"
				},
				"type": "slack",
				"sendReminder": true,
				"name": "slack.receiver",
				"disableResolveMessage": false,
				"uid": ""
			}]
		}]
	}
}
`
		resp, err := http.Post(alertConfigURL, "application/json", bytes.NewBufferString(payload))
		require.NoError(t, err)
		require.Equal(t, 202, resp.StatusCode)

		resp, err = http.Get(alertConfigURL)
		require.NoError(t, err)
		t.Cleanup(func() {
			err := resp.Body.Close()
			require.NoError(t, err)
		})
		b, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode)
		fmt.Println(string(b))
		require.JSONEq(t, `
{
	"template_files": {},
	"alertmanager_config": {
		"route": {
			"receiver": "slack.receiver"
		},
		"templates": null,
		"receivers": [{
			"name": "slack.receiver",
			"grafana_managed_receiver_configs": [{
				"id": 0,
				"uid": "",
				"name": "slack.receiver",
				"type": "slack",
				"isDefault": false,
				"sendReminder": true,
				"disableResolveMessage": false,
				"frequency": "",
				"created": "0001-01-01T00:00:00Z",
				"updated": "0001-01-01T00:00:00Z",
				"settings": {
					"iconEmoji": "",
					"iconUrl": "",
					"mentionGroups": "",
					"mentionUsers": "",
					"recipient": "#unified-alerting-test",
					"username": ""
				},
				"secureFields": {
					"token": true,
					"url": true
				}
			}]
		}]
	}
}
`, string(b))
	}
}
