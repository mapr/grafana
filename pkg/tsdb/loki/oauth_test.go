package loki

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/infra/tracing"
	"github.com/stretchr/testify/require"
)

type mockedRoundTripperForOauth struct {
	requestCallback func(req *http.Request)
}

func (mockedRT *mockedRoundTripperForOauth) RoundTrip(req *http.Request) (*http.Response, error) {
	mockedRT.requestCallback(req)
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{},
		Body:       io.NopCloser(bytes.NewReader([]byte("ok"))),
	}, nil
}

type mockedCallResourceResponseSenderForOauth struct{}

func (s *mockedCallResourceResponseSenderForOauth) Send(resp *backend.CallResourceResponse) error {
	return nil
}

func makeMockedDsInfoForOauth(oauthPassThru bool, requestCallback func(req *http.Request)) datasourceInfo {
	client := http.Client{
		Transport: &mockedRoundTripperForOauth{requestCallback: requestCallback},
	}

	return datasourceInfo{
		HTTPClient:    &client,
		OauthPassThru: oauthPassThru,
	}
}

func TestOauthForwardIdentity(t *testing.T) {

	tt := []struct {
		name          string
		oauthPassThru bool
		headersGiven  bool
		headersSent   bool
	}{
		{name: "when enabled and headers exist => add headers", oauthPassThru: true, headersGiven: true, headersSent: true},
		{name: "when disabled and headers exist => do not add headers", oauthPassThru: false, headersGiven: true, headersSent: false},
		{name: "when enabled and no headers exist => do not add headers", oauthPassThru: true, headersGiven: false, headersSent: false},
		{name: "when disabled and no headers exist => do not add headers", oauthPassThru: false, headersGiven: false, headersSent: false},
	}

	authName := "Authorization"
	authValue := "auth"
	xidName := "X-ID-Token"
	xidValue := "xid"

	for _, test := range tt {
		t.Run("QueryData: "+test.name, func(t *testing.T) {

			clientUsed := false
			dsInfo := makeMockedDsInfoForOauth(test.oauthPassThru, func(req *http.Request) {
				clientUsed = true
				if test.headersSent {
					require.Equal(t, authValue, req.Header.Get(authName))
					require.Equal(t, xidValue, req.Header.Get(xidName))
				} else {
					require.Equal(t, "", req.Header.Get(authName))
					require.Equal(t, "", req.Header.Get(xidName))
				}
			})

			req := backend.QueryDataRequest{
				Headers: map[string]string{},
				Queries: []backend.DataQuery{
					{
						JSON: []byte("{}"),
					},
				},
			}

			if test.headersGiven {
				req.Headers[authName] = authValue
				req.Headers[xidName] = xidValue
			}

			tracer, err := tracing.InitializeTracerForTest()
			require.NoError(t, err)

			// we do not care about the result of QueryData
			queryData(context.Background(), &req, &dsInfo, log.New("testlog"), tracer)

			// we need to be sure the client-callback was triggered
			require.True(t, clientUsed)
		})
	}

	for _, test := range tt {
		t.Run("CallResource: "+test.name, func(t *testing.T) {

			clientUsed := false
			dsInfo := makeMockedDsInfoForOauth(test.oauthPassThru, func(req *http.Request) {
				clientUsed = true
				if test.headersSent {
					require.Equal(t, authValue, req.Header.Get(authName))
					require.Equal(t, xidValue, req.Header.Get(xidName))
				} else {
					require.Equal(t, "", req.Header.Get(authName))
					require.Equal(t, "", req.Header.Get(xidName))
				}
			})

			req := backend.CallResourceRequest{
				Headers: map[string][]string{},
				Method:  "GET",
				URL:     "/loki/api/v1/labels?",
			}

			if test.headersGiven {
				req.Headers[authName] = []string{authValue}
				req.Headers[xidName] = []string{xidValue}
			}

			sender := &mockedCallResourceResponseSenderForOauth{}

			// we do not care about the result of QueryData
			callResource(context.Background(), &req, sender, &dsInfo, log.New("testlog"))

			// we need to be sure the client-callback was triggered
			require.True(t, clientUsed)
		})
	}
}
