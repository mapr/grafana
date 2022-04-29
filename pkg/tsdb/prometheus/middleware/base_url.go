package middleware

import (
	"net/http"
	"net/url"
	"path"

	sdkhttpclient "github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
)

func BaseURL(s string) sdkhttpclient.Middleware {
	u, err := url.Parse(s)
	return sdkhttpclient.NamedMiddlewareFunc("base-url", func(opts sdkhttpclient.Options, next http.RoundTripper) http.RoundTripper {
		return sdkhttpclient.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			if err != nil {
				return nil, err
			}
			req.URL.Scheme = u.Scheme
			req.URL.Host = u.Host
			req.URL.Path = path.Join(req.URL.Path, u.Path)
			return next.RoundTrip(req)
		})
	})
}
