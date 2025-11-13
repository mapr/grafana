package connectors

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"slices"

	jose "github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"golang.org/x/oauth2"

	"github.com/grafana/grafana/pkg/apimachinery/identity"
	"github.com/grafana/grafana/pkg/infra/remotecache"
	"github.com/grafana/grafana/pkg/login/social"
	"github.com/grafana/grafana/pkg/services/featuremgmt"
	"github.com/grafana/grafana/pkg/services/ssosettings"
	ssoModels "github.com/grafana/grafana/pkg/services/ssosettings/models"
	"github.com/grafana/grafana/pkg/services/ssosettings/validation"
	"github.com/grafana/grafana/pkg/setting"
	"github.com/grafana/grafana/pkg/util"
)

const (
	serverDiscoveryURLKey      = "server_discovery_url"
	oidcDiscoveryCachePrefix   = "oidc_discovery-"
	oidcJWKSCachePrefix        = "oidc_jwks-"
	oidcDefaultCacheExpiration = 5 * time.Minute
)

var ExtraOIDCSettingKeys = map[string]ExtraKeyInfo{
	serverDiscoveryURLKey: {Type: String},
	nameAttributePathKey:  {Type: String, DefaultValue: string(defaultNameAttributeClaim)},
	loginAttributePathKey: {Type: String, DefaultValue: string(defaultLoginAttributeClaim)},
}

type SocialOIDC struct {
	*SocialBase
	cache remotecache.CacheStorage

	discoveryURL       string
	nameAttributePath  string
	loginAttributePath string

	discoveryMu     sync.Mutex
	discoveryExpiry time.Time
	discoveryDoc    *oidcDiscoveryDocument
}

type oidcDiscoveryDocument struct {
	Issuer                           string   `json:"issuer"`
	AuthorizationEndpoint            string   `json:"authorization_endpoint"`
	TokenEndpoint                    string   `json:"token_endpoint"`
	UserInfoEndpoint                 string   `json:"userinfo_endpoint"`
	JWKSURI                          string   `json:"jwks_uri"`
	IDTokenSigningAlgValuesSupported []string `json:"id_token_signing_alg_values_supported"`
}

type oidcIDTokenClaims struct {
	Email             string          `json:"email"`
	EmailVerified     *bool           `json:"email_verified,omitempty"`
	PreferredUsername string          `json:"preferred_username"`
	Name              string          `json:"name"`
	GivenName         string          `json:"given_name"`
	FamilyName        string          `json:"family_name"`
	Groups            json.RawMessage `json:"groups"`
}

func NewOIDCProvider(info *social.OAuthInfo, cfg *setting.Cfg, orgRoleMapper *OrgRoleMapper, ssoSettings ssosettings.Service, features featuremgmt.FeatureToggles, cache remotecache.CacheStorage) *SocialOIDC {
	ensureOIDCScopes(info)
	ensureDefaultAttributeClaims(info)

	provider := &SocialOIDC{
		SocialBase:         newSocialBase(social.OIDCProviderName, orgRoleMapper, info, features, cfg),
		cache:              cache,
		discoveryURL:       info.Extra[serverDiscoveryURLKey],
		nameAttributePath:  info.Extra[nameAttributePathKey],
		loginAttributePath: info.Extra[loginAttributePathKey],
	}

	appendUniqueScope(provider.Config, "openid")
	if info.UseRefreshToken {
		appendUniqueScope(provider.Config, social.OfflineAccessScope)
	}

	ssoSettings.RegisterReloadable(social.OIDCProviderName, provider)

	return provider
}

func (s *SocialOIDC) AuthCodeURL(state string, opts ...oauth2.AuthCodeOption) string {
	s.prefetchDiscovery(context.Background())
	return s.SocialBase.AuthCodeURL(state, opts...)
}

func (s *SocialOIDC) Exchange(ctx context.Context, code string, opts ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
	s.prefetchDiscovery(ctx)
	return s.SocialBase.Exchange(ctx, code, opts...)
}

func (s *SocialOIDC) Validate(ctx context.Context, newSettings ssoModels.SSOSettings, oldSettings ssoModels.SSOSettings, requester identity.Requester) error {
	info, err := CreateOAuthInfoFromKeyValues(newSettings.Settings)
	if err != nil {
		return ssosettings.ErrInvalidSettings.Errorf("SSO settings map cannot be converted to OAuthInfo: %v", err)
	}

	oldInfo, err := CreateOAuthInfoFromKeyValues(oldSettings.Settings)
	if err != nil {
		oldInfo = &social.OAuthInfo{}
	}

	if err := validateInfo(info, oldInfo, requester); err != nil {
		return err
	}

	if err := validateStandardAttributeClaims(info); err != nil {
		return err
	}

	if err := validateOIDCEmailClaim(info); err != nil {
		return err
	}

	return validateOIDCEndpoints(info, newSettings.Settings, requester)
}

func (s *SocialOIDC) Reload(ctx context.Context, settings ssoModels.SSOSettings) error {
	newInfo, err := CreateOAuthInfoFromKeyValuesWithLogging(s.log, social.OIDCProviderName, settings.Settings)
	if err != nil {
		return ssosettings.ErrInvalidSettings.Errorf("SSO settings map cannot be converted to OAuthInfo: %v", err)
	}

	ensureOIDCScopes(newInfo)
	ensureDefaultAttributeClaims(newInfo)

	s.reloadMutex.Lock()
	defer s.reloadMutex.Unlock()

	s.updateInfo(ctx, social.OIDCProviderName, newInfo)

	s.discoveryURL = newInfo.Extra[serverDiscoveryURLKey]
	s.nameAttributePath = newInfo.Extra[nameAttributePathKey]
	s.loginAttributePath = newInfo.Extra[loginAttributePathKey]

	appendUniqueScope(s.Config, "openid")
	if newInfo.UseRefreshToken {
		appendUniqueScope(s.Config, social.OfflineAccessScope)
	}

	s.resetDiscoveryMetadata()

	return nil
}

func (s *SocialOIDC) UserInfo(ctx context.Context, client *http.Client, token *oauth2.Token) (*social.BasicUserInfo, error) {
	s.reloadMutex.RLock()
	defer s.reloadMutex.RUnlock()

	discoveryDoc, err := s.ensureDiscovery(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("oidc discovery failed: %w", err)
	}

	idToken := token.Extra("id_token")
	if idToken == nil {
		return nil, ErrIDTokenNotFound
	}

	idTokenStr, ok := idToken.(string)
	if !ok || idTokenStr == "" {
		return nil, ErrIDTokenNotFound
	}

	algs := defaultOIDCAlgorithms(discoveryDoc.IDTokenSigningAlgValuesSupported)
	parsedToken, err := jwt.ParseSigned(idTokenStr, algs)
	if err != nil {
		return nil, fmt.Errorf("oidc: failed to parse id token: %w", err)
	}

	rawIDToken, err := s.retrieveRawJWTPayload(idTokenStr)
	if err != nil {
		return nil, fmt.Errorf("oidc: failed to extract id token payload: %w", err)
	}

	claims, customClaims, err := s.verifyIDToken(ctx, client, parsedToken, discoveryDoc)
	if err != nil {
		return nil, err
	}

	rawSources := [][]byte{rawIDToken}
	userInfoRaw := s.fetchUserInfo(ctx, client, discoveryDoc.UserInfoEndpoint)
	if len(userInfoRaw) > 0 {
		rawSources = append(rawSources, userInfoRaw)
	}

	email := customClaims.Email
	if email == "" && customClaims.PreferredUsername != "" && strings.Contains(customClaims.PreferredUsername, "@") {
		email = customClaims.PreferredUsername
	}
	if email == "" {
		return nil, ErrEmailNotFound
	}

	name := ""
	if s.nameAttributePath != "" {
		name = s.searchStringAttr(rawSources, s.nameAttributePath)
	}
	if name == "" {
		name = customClaims.Name
	}
	if name == "" {
		parts := strings.TrimSpace(strings.Join([]string{customClaims.GivenName, customClaims.FamilyName}, " "))
		name = strings.TrimSpace(parts)
	}
	if name == "" {
		name = email
	}

	login := ""
	if s.loginAttributePath != "" {
		login = s.searchStringAttr(rawSources, s.loginAttributePath)
	}
	if login == "" {
		login = customClaims.PreferredUsername
	}
	if login == "" {
		login = email
	}

	groups := []string{}
	if s.info.GroupsAttributePath != "" {
		groups = s.searchStringSliceAttr(rawSources, s.info.GroupsAttributePath)
	}
	if len(groups) == 0 {
		groups = parseGroupsClaim(customClaims.Groups)
	}
	groups = uniqueStrings(groups)

	if !s.isGroupMember(groups) {
		return nil, errMissingGroupMembership
	}

	role, grafanaAdmin, roleErr := s.extractRoleAndAdminOptional(rawIDToken, groups)
	if roleErr != nil {
		s.log.Warn("OIDC: failed to extract role from id token", "err", roleErr)
	}
	if role == "" && len(userInfoRaw) > 0 {
		if newRole, newAdmin, err := s.extractRoleAndAdminOptional(userInfoRaw, groups); err == nil {
			role = newRole
			grafanaAdmin = newAdmin
		} else {
			s.log.Warn("OIDC: failed to extract role from userinfo", "err", err)
		}
	}

	externalOrgs, err := s.extractOrgs(rawIDToken)
	if err != nil {
		s.log.Warn("OIDC: failed to extract organizations from id token", "err", err)
	}
	if len(externalOrgs) == 0 && len(userInfoRaw) > 0 {
		if orgs, err := s.extractOrgs(userInfoRaw); err == nil {
			externalOrgs = orgs
		} else {
			s.log.Warn("OIDC: failed to extract organizations from userinfo", "err", err)
		}
	}

	userInfo := &social.BasicUserInfo{
		Id:     claims.Subject,
		Name:   name,
		Email:  email,
		Login:  login,
		Role:   role,
		Groups: groups,
	}

	if !s.info.SkipOrgRoleSync {
		userInfo.OrgRoles = s.orgRoleMapper.MapOrgRoles(s.orgMappingCfg, externalOrgs, role)
		if s.info.RoleAttributeStrict && len(userInfo.OrgRoles) == 0 {
			return nil, errRoleAttributeStrictViolation.Errorf("could not evaluate any valid roles using IdP provided data")
		}
	}

	if s.info.AllowAssignGrafanaAdmin {
		userInfo.IsGrafanaAdmin = &grafanaAdmin
	}

	if s.info.AllowAssignGrafanaAdmin && s.info.SkipOrgRoleSync {
		s.log.Debug("AllowAssignGrafanaAdmin and skipOrgRoleSync are both set, Grafana Admin role will not be synced, consider setting one or the other")
	}

	return userInfo, nil
}

func ensureOIDCScopes(info *social.OAuthInfo) {
	info.Scopes = ensureScope(info.Scopes, "openid")
	if info.UseRefreshToken {
		info.Scopes = ensureScope(info.Scopes, social.OfflineAccessScope)
	}
}

func (s *SocialOIDC) prefetchDiscovery(ctx context.Context) {
	if s.discoveryURL == "" {
		return
	}
	if _, err := s.ensureDiscovery(ctx, http.DefaultClient); err != nil {
		s.log.Warn("OIDC: failed to update endpoints from discovery", "err", err)
	}
}

func ensureScope(scopes []string, scope string) []string {
	if scope == "" {
		return scopes
	}
	if slices.Contains(scopes, scope) {
		return scopes
	}
	return append(scopes, scope)
}

func validateOIDCEmailClaim(info *social.OAuthInfo) error {
	if info.EmailAttributePath != "" {
		return ssosettings.ErrInvalidOAuthConfig("Email attribute path is not supported for OIDC. Remove this setting to use the email claim from the ID token.")
	}

	return nil
}

func validateOIDCEndpoints(info *social.OAuthInfo, settings map[string]any, requester identity.Requester) error {
	discoveryURL := info.Extra[serverDiscoveryURLKey]

	if discoveryURL == "" {
		return ssosettings.ErrInvalidOAuthConfig("Server discovery URL is required for OIDC configuration.")
	}

	if err := validation.UrlValidator(discoveryURL, "Server discovery URL")(info, requester); err != nil {
		return err
	}

	if info.AuthUrl != "" || info.TokenUrl != "" {
		return ssosettings.ErrInvalidOAuthConfig("Auth URL and Token URL are managed via discovery and must be empty.")
	}

	if hasNonEmptyStringSetting(settings, "api_url") || hasNonEmptyStringSetting(settings, "apiUrl") {
		return ssosettings.ErrInvalidOAuthConfig("API URL is managed via discovery and must be empty.")
	}

	return nil
}

func hasNonEmptyStringSetting(settings map[string]any, key string) bool {
	if settings == nil {
		return false
	}

	value, ok := settings[key]
	if !ok {
		return false
	}

	switch v := value.(type) {
	case string:
		return v != ""
	case *string:
		return v != nil && *v != ""
	default:
		return value != nil
	}
}

func defaultOIDCAlgorithms(values []string) []jose.SignatureAlgorithm {
	if len(values) == 0 {
		return []jose.SignatureAlgorithm{
			jose.RS256, jose.RS384, jose.RS512,
			jose.ES256, jose.ES384, jose.ES512,
			jose.PS256, jose.PS384, jose.PS512,
			jose.EdDSA, jose.HS256, jose.HS384, jose.HS512,
		}
	}
	algs := make([]jose.SignatureAlgorithm, 0, len(values))
	for _, value := range values {
		alg := jose.SignatureAlgorithm(value)
		switch alg {
		case jose.RS256, jose.RS384, jose.RS512,
			jose.ES256, jose.ES384, jose.ES512,
			jose.PS256, jose.PS384, jose.PS512,
			jose.EdDSA, jose.HS256, jose.HS384, jose.HS512:
			algs = append(algs, alg)
		}
	}
	if len(algs) == 0 {
		return defaultOIDCAlgorithms(nil)
	}
	return algs
}

func (s *SocialOIDC) ensureDiscovery(ctx context.Context, client *http.Client) (oidcDiscoveryDocument, error) {
	s.discoveryMu.Lock()
	defer s.discoveryMu.Unlock()

	if s.discoveryURL == "" {
		return oidcDiscoveryDocument{}, fmt.Errorf("oidc: discovery URL not configured")
	}

	if s.discoveryDoc != nil && time.Now().Before(s.discoveryExpiry) {
		doc := *s.discoveryDoc
		s.info.AuthUrl = doc.AuthorizationEndpoint
		s.info.TokenUrl = doc.TokenEndpoint
		s.Config.Endpoint.AuthURL = s.info.AuthUrl
		s.Config.Endpoint.TokenURL = s.info.TokenUrl
		return doc, nil
	}

	doc, ttl, err := s.fetchDiscovery(ctx, client)
	if err != nil {
		return oidcDiscoveryDocument{}, err
	}

	discoveryDoc := s.getDiscovery(doc)
	if discoveryDoc == nil {
		return oidcDiscoveryDocument{}, fmt.Errorf("oidc: discovery metadata unavailable")
	}
	if ttl <= 0 {
		ttl = oidcDefaultCacheExpiration
	}
	s.discoveryExpiry = time.Now().Add(ttl)

	s.info.AuthUrl = discoveryDoc.AuthorizationEndpoint
	s.info.TokenUrl = discoveryDoc.TokenEndpoint
	s.Config.Endpoint.AuthURL = s.info.AuthUrl
	s.Config.Endpoint.TokenURL = s.info.TokenUrl

	return *discoveryDoc, nil
}

func (s *SocialOIDC) fetchDiscovery(ctx context.Context, client *http.Client) (*oidcDiscoveryDocument, time.Duration, error) {
	cacheKey, err := s.discoveryCacheKey()
	if err != nil {
		return nil, 0, err
	}

	if s.cache != nil && cacheKey != "" {
		if cached, err := s.cache.Get(ctx, cacheKey); err == nil && len(cached) > 0 {
			var doc oidcDiscoveryDocument
			if err := json.Unmarshal(cached, &doc); err == nil {
				return &doc, 0, nil
			}
			s.log.Warn("OIDC: failed to unmarshal cached discovery document", "err", err)
		}
	}

	resp, err := s.httpGet(ctx, client, s.discoveryURL)
	if err != nil {
		return nil, 0, fmt.Errorf("oidc: failed to fetch discovery document: %w", err)
	}

	var doc oidcDiscoveryDocument
	if err := json.Unmarshal(resp.Body, &doc); err != nil {
		return nil, 0, fmt.Errorf("oidc: failed to decode discovery document: %w", err)
	}

	if doc.Issuer == "" || doc.JWKSURI == "" {
		return nil, 0, fmt.Errorf("oidc: discovery document missing required fields")
	}

	ttl := cacheExpirationFromHeader(resp.Headers.Get("cache-control"), oidcDefaultCacheExpiration)

	if s.cache != nil && cacheKey != "" {
		if err := s.cache.Set(ctx, cacheKey, resp.Body, ttl); err != nil {
			s.log.Warn("OIDC: failed to cache discovery document", "err", err)
		}
	}

	return &doc, ttl, nil
}

func (s *SocialOIDC) discoveryCacheKey() (string, error) {
	if s.discoveryURL == "" {
		return "", nil
	}
	value, err := util.Md5SumString(s.discoveryURL)
	if err != nil {
		return "", err
	}
	return oidcDiscoveryCachePrefix + value, nil
}

func (s *SocialOIDC) jwksCacheKey(jwksURI string) (string, error) {
	if jwksURI == "" {
		return "", fmt.Errorf("oidc: jwks uri not available")
	}
	value, err := util.Md5SumString(jwksURI)
	if err != nil {
		return "", err
	}
	return oidcJWKSCachePrefix + value, nil
}

func (s *SocialOIDC) getDiscovery(doc *oidcDiscoveryDocument) *oidcDiscoveryDocument {
	if doc == nil {
		return nil
	}
	copyDoc := &oidcDiscoveryDocument{
		Issuer:                           doc.Issuer,
		AuthorizationEndpoint:            doc.AuthorizationEndpoint,
		TokenEndpoint:                    doc.TokenEndpoint,
		UserInfoEndpoint:                 doc.UserInfoEndpoint,
		JWKSURI:                          doc.JWKSURI,
		IDTokenSigningAlgValuesSupported: slices.Clone(doc.IDTokenSigningAlgValuesSupported),
	}
	s.discoveryDoc = copyDoc
	return copyDoc
}

func (s *SocialOIDC) loadJWKS(ctx context.Context, client *http.Client, jwksURI string) (*jose.JSONWebKeySet, time.Duration, error) {
	if jwksURI == "" {
		return nil, 0, fmt.Errorf("oidc: jwks uri not available")
	}

	cacheKey, err := s.jwksCacheKey(jwksURI)
	if err != nil {
		return nil, 0, err
	}

	if s.cache != nil && cacheKey != "" {
		if cached, err := s.cache.Get(ctx, cacheKey); err == nil && len(cached) > 0 {
			var set jose.JSONWebKeySet
			if err := json.Unmarshal(cached, &set); err == nil {
				return &set, 0, nil
			}
			s.log.Warn("OIDC: failed to unmarshal cached JWKS", "err", err)
		}
	}

	resp, err := s.httpGet(ctx, client, jwksURI)
	if err != nil {
		return nil, 0, fmt.Errorf("oidc: failed to fetch jwks: %w", err)
	}

	var set jose.JSONWebKeySet
	if err := json.Unmarshal(resp.Body, &set); err != nil {
		return nil, 0, fmt.Errorf("oidc: failed to decode jwks: %w", err)
	}

	ttl := cacheExpirationFromHeader(resp.Headers.Get("cache-control"), oidcDefaultCacheExpiration)

	if s.cache != nil && cacheKey != "" {
		if err := s.cache.Set(ctx, cacheKey, resp.Body, ttl); err != nil {
			s.log.Warn("OIDC: failed to cache JWKS", "err", err)
		}
	}

	return &set, ttl, nil
}

func (s *SocialOIDC) verifyIDToken(ctx context.Context, client *http.Client, parsedToken *jwt.JSONWebToken, doc oidcDiscoveryDocument) (jwt.Claims, oidcIDTokenClaims, error) {
	var claims jwt.Claims
	var custom oidcIDTokenClaims

	if len(parsedToken.Headers) == 0 {
		return claims, custom, &SocialError{"OIDC: id token header missing"}
	}

	header := parsedToken.Headers[0]
	keyID := header.KeyID

	switch jose.SignatureAlgorithm(header.Algorithm) {
	case jose.HS256, jose.HS384, jose.HS512:
		if s.ClientSecret == "" {
			return claims, custom, &SocialError{fmt.Sprintf("OIDC: client secret required for %s signatures", header.Algorithm)}
		}
		if err := parsedToken.Claims([]byte(s.ClientSecret), &claims, &custom); err != nil {
			return claims, custom, &SocialError{fmt.Sprintf("OIDC: invalid id token signature: %v", err)}
		}
	default:
		jwks, _, err := s.loadJWKS(ctx, client, doc.JWKSURI)
		if err != nil {
			return claims, custom, err
		}

		keys := jwks.Key(keyID)
		if len(keys) == 0 {
			keys = jwks.Keys
		}

		var parseErr error
		for _, key := range keys {
			var tmpClaims jwt.Claims
			var tmpCustom oidcIDTokenClaims
			if err := parsedToken.Claims(key, &tmpClaims, &tmpCustom); err != nil {
				parseErr = err
				continue
			}
			claims = tmpClaims
			custom = tmpCustom
			parseErr = nil
			break
		}

		if parseErr != nil {
			return claims, custom, &SocialError{fmt.Sprintf("OIDC: unable to verify id token signature: %v", parseErr)}
		}
	}

	if doc.Issuer != "" {
		expected := jwt.Expected{
			Issuer:      doc.Issuer,
			AnyAudience: jwt.Audience{s.ClientID},
			Time:        time.Now(),
		}
		if err := claims.Validate(expected); err != nil {
			return claims, custom, &SocialError{fmt.Sprintf("OIDC: ID token validation failed: %v", err)}
		}
	}

	if claims.Subject == "" {
		return claims, custom, &SocialError{"OIDC: subject claim is missing"}
	}

	return claims, custom, nil
}

func (s *SocialOIDC) fetchUserInfo(ctx context.Context, client *http.Client, url string) []byte {
	if url == "" {
		return nil
	}

	resp, err := s.httpGet(ctx, client, url)
	if err != nil {
		s.log.Debug("OIDC: failed to fetch userinfo", "url", url, "err", err)
		return nil
	}

	return resp.Body
}

func (s *SocialOIDC) searchStringAttr(rawSources [][]byte, path string) string {
	for _, raw := range rawSources {
		if len(raw) == 0 {
			continue
		}
		value, err := util.SearchJSONForStringAttr(path, raw)
		if err != nil {
			s.log.Debug("OIDC: failed to resolve string attribute", "path", path, "err", err)
			continue
		}
		if value != "" {
			return value
		}
	}
	return ""
}

func (s *SocialOIDC) searchStringSliceAttr(rawSources [][]byte, path string) []string {
	for _, raw := range rawSources {
		if len(raw) == 0 {
			continue
		}
		values, err := util.SearchJSONForStringSliceAttr(path, raw)
		if err != nil {
			s.log.Debug("OIDC: failed to resolve slice attribute", "path", path, "err", err)
			continue
		}
		if len(values) > 0 {
			return values
		}
	}
	return nil
}

func parseGroupsClaim(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}

	var arr []string
	if err := json.Unmarshal(raw, &arr); err == nil {
		return arr
	}

	var generic []any
	if err := json.Unmarshal(raw, &generic); err == nil {
		result := make([]string, 0, len(generic))
		for _, item := range generic {
			if str, ok := item.(string); ok && str != "" {
				result = append(result, str)
			}
		}
		return result
	}

	var single string
	if err := json.Unmarshal(raw, &single); err == nil && single != "" {
		return []string{single}
	}

	return nil
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func cacheExpirationFromHeader(header string, fallback time.Duration) time.Duration {
	if header == "" {
		return fallback
	}

	directives := strings.Split(header, ",")
	for _, directive := range directives {
		directive = strings.TrimSpace(strings.ToLower(directive))
		if strings.HasPrefix(directive, "max-age=") {
			parts := strings.SplitN(directive, "=", 2)
			if len(parts) != 2 {
				continue
			}
			seconds, err := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err != nil || seconds <= 0 {
				continue
			}
			return time.Duration(seconds) * time.Second
		}
	}

	return fallback
}

func (s *SocialOIDC) resetDiscoveryMetadata() {
	s.discoveryMu.Lock()
	defer s.discoveryMu.Unlock()

	s.discoveryExpiry = time.Time{}
	s.discoveryDoc = nil
}
