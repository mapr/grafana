package connectors

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana/pkg/login/social"
	ssoModels "github.com/grafana/grafana/pkg/services/ssosettings/models"
	"github.com/grafana/grafana/pkg/services/user"
)

func TestEnsureDefaultAttributeClaimsSetsDefaults(t *testing.T) {
	info := &social.OAuthInfo{Extra: map[string]string{}}

	ensureDefaultAttributeClaims(info)

	require.Equal(t, string(defaultNameAttributeClaim), info.Extra[nameAttributePathKey])
	require.Equal(t, string(defaultLoginAttributeClaim), info.Extra[loginAttributePathKey])
}

func TestValidateStandardAttributeClaimsRejectsInvalid(t *testing.T) {
	info := &social.OAuthInfo{Extra: map[string]string{
		nameAttributePathKey:  "custom",
		loginAttributePathKey: string(defaultLoginAttributeClaim),
	}}

	err := validateStandardAttributeClaims(info)

	require.Error(t, err)
}

func TestSocialOIDCValidateRejectsInvalidClaims(t *testing.T) {
	provider := &SocialOIDC{}

	settings := ssoModels.SSOSettings{Settings: map[string]any{
		"client_id":           "client-id",
		"auth_url":            "https://example.com/auth",
		"token_url":           "https://example.com/token",
		nameAttributePathKey:  "custom",
		loginAttributePathKey: string(defaultLoginAttributeClaim),
	}}

	err := provider.Validate(context.Background(), settings, ssoModels.SSOSettings{}, &user.SignedInUser{IsGrafanaAdmin: false})

	require.Error(t, err)
}

func TestSocialOIDCValidateRequiresDiscoveryURL(t *testing.T) {
	provider := &SocialOIDC{}

	settings := ssoModels.SSOSettings{Settings: map[string]any{
		"client_id": "client-id",
	}}

	err := provider.Validate(context.Background(), settings, ssoModels.SSOSettings{}, &user.SignedInUser{IsGrafanaAdmin: false})

	require.Error(t, err)
}

func TestSocialOIDCValidateRejectsManualEndpoints(t *testing.T) {
	provider := &SocialOIDC{}

	settings := ssoModels.SSOSettings{Settings: map[string]any{
		"client_id":           "client-id",
		serverDiscoveryURLKey: "https://idp.example/.well-known/openid-configuration",
		"auth_url":            "https://example.com/auth",
		"token_url":           "https://example.com/token",
		"api_url":             "https://example.com/userinfo",
		nameAttributePathKey:  string(defaultNameAttributeClaim),
		loginAttributePathKey: string(defaultLoginAttributeClaim),
	}}

	err := provider.Validate(context.Background(), settings, ssoModels.SSOSettings{}, &user.SignedInUser{IsGrafanaAdmin: false})

	require.Error(t, err)
}

func TestSocialOIDCValidateRejectsEmailAttributePath(t *testing.T) {
	provider := &SocialOIDC{}

	settings := ssoModels.SSOSettings{Settings: map[string]any{
		"client_id":            "client-id",
		serverDiscoveryURLKey:  "https://idp.example/.well-known/openid-configuration",
		"email_attribute_path": "email",
		nameAttributePathKey:   string(defaultNameAttributeClaim),
		loginAttributePathKey:  string(defaultLoginAttributeClaim),
	}}

	err := provider.Validate(context.Background(), settings, ssoModels.SSOSettings{}, &user.SignedInUser{IsGrafanaAdmin: false})

	require.Error(t, err)
}
