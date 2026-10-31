// Copyright 2026 The Casnode Authors. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package object

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// AuthProvider is a single sign-on service members can log in with, set up on /admin/logins: any OpenID Connect
// provider (Keycloak, Okta, Microsoft Entra ID, Google, Authentik, Casdoor...) by its issuer, or any OAuth 2.0 provider
// (GitHub, Discord, Gitee...) by its endpoints and the fields of its user info.
type AuthProvider struct {
	Name         string `xorm:"varchar(50) pk" json:"name"`
	DisplayName  string `xorm:"varchar(100)" json:"displayName"`
	Type         string `xorm:"varchar(20)" json:"type"`
	Icon         string `xorm:"varchar(50)" json:"icon"`
	ClientId     string `xorm:"varchar(500)" json:"clientId"`
	ClientSecret string `xorm:"varchar(1000)" json:"clientSecret"`
	// OpenID Connect: the endpoints come from <issuer>/.well-known/openid-configuration
	Issuer string `xorm:"varchar(500)" json:"issuer"`
	// OAuth 2.0, or to override what the issuer says
	AuthUrl     string `xorm:"varchar(500)" json:"authUrl"`
	TokenUrl    string `xorm:"varchar(500)" json:"tokenUrl"`
	UserInfoUrl string `xorm:"varchar(500)" json:"userInfoUrl"`
	Scopes      string `xorm:"varchar(500)" json:"scopes"`
	// where the user info keeps each value; empty means the OpenID Connect claim (sub, preferred_username, ...)
	IdField       string    `xorm:"varchar(100)" json:"idField"`
	UsernameField string    `xorm:"varchar(100)" json:"usernameField"`
	NameField     string    `xorm:"varchar(100)" json:"nameField"`
	EmailField    string    `xorm:"varchar(100)" json:"emailField"`
	AvatarField   string    `xorm:"varchar(100)" json:"avatarField"`
	AllowSignup   bool      `json:"allowSignup"`
	Enabled       bool      `json:"enabled"`
	SortOrder     int       `json:"sortOrder"`
	CreatedAt     time.Time `xorm:"created" json:"createdAt"`
	UpdatedAt     time.Time `xorm:"updated" json:"updatedAt"`
}

// UserAuth links a user to an account at a single sign-on provider.
type UserAuth struct {
	Id        int64     `xorm:"pk autoincr" json:"id"`
	Provider  string    `xorm:"varchar(50) unique(provider_subject)" json:"provider"`
	Subject   string    `xorm:"varchar(191) unique(provider_subject)" json:"subject"`
	UserId    int64     `xorm:"index" json:"userId"`
	CreatedAt time.Time `xorm:"created" json:"createdAt"`
}

// PublicAuthProvider is what the login page shows of a provider.
type PublicAuthProvider struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Icon        string `json:"icon"`
}

// OAuthIdentity is who a provider says signed in.
type OAuthIdentity struct {
	Subject  string
	Username string
	Name     string
	Email    string
	Avatar   string
}

const casdoorProviderName = "casdoor"

var providerNameRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9\-]{0,49}$`)

var oauthHttpClient = &http.Client{Timeout: 15 * time.Second}

func GetAuthProviders() ([]*AuthProvider, error) {
	providers := []*AuthProvider{}
	err := adapter.Engine.Asc("sort_order", "name").Find(&providers)
	return providers, err
}

func GetAuthProvider(name string) (*AuthProvider, error) {
	provider := &AuthProvider{}
	has, err := adapter.Engine.ID(name).Get(provider)
	if err != nil || !has {
		return nil, err
	}
	return provider, nil
}

func GetPublicAuthProviders() []*PublicAuthProvider {
	res := []*PublicAuthProvider{}
	providers, err := GetAuthProviders()
	if err != nil {
		return res
	}
	for _, provider := range providers {
		if provider.Enabled {
			res = append(res, &PublicAuthProvider{Name: provider.Name, DisplayName: provider.DisplayName, Icon: provider.Icon})
		}
	}
	return res
}

// MaskSecret replaces a set client secret by the placeholder the admin page sends back unchanged.
func (p *AuthProvider) MaskSecret() *AuthProvider {
	res := *p
	if res.ClientSecret != "" {
		res.ClientSecret = SecretPlaceholder
	}
	return &res
}

// SaveAuthProvider adds a provider, or changes the one called originalName.
func SaveAuthProvider(originalName string, provider *AuthProvider) error {
	provider.Name = strings.TrimSpace(provider.Name)
	provider.DisplayName = strings.TrimSpace(provider.DisplayName)
	provider.Issuer = strings.TrimSuffix(strings.TrimSpace(provider.Issuer), "/")
	if !providerNameRegex.MatchString(provider.Name) {
		return fmt.Errorf("The name is lowercase letters, digits and dashes; it goes into the callback URL")
	}
	if provider.Name == casdoorProviderName {
		return fmt.Errorf("The name casdoor is kept for the Casdoor set up in conf/app.conf")
	}
	if provider.DisplayName == "" {
		return fmt.Errorf("Give the provider a name to show on the login page")
	}
	if provider.ClientId == "" {
		return fmt.Errorf("The client ID is required")
	}
	switch provider.Type {
	case "oidc":
		if provider.Issuer == "" && (provider.AuthUrl == "" || provider.TokenUrl == "" || provider.UserInfoUrl == "") {
			return fmt.Errorf("Give the issuer URL, or all three endpoints")
		}
	case "oauth2":
		if provider.AuthUrl == "" || provider.TokenUrl == "" || provider.UserInfoUrl == "" {
			return fmt.Errorf("An OAuth 2.0 provider needs the authorization, token and user info URLs")
		}
		if provider.IdField == "" {
			return fmt.Errorf("An OAuth 2.0 provider needs the field of the user info that holds the user ID")
		}
	default:
		return fmt.Errorf("The type is oidc or oauth2")
	}
	for _, value := range []string{provider.Issuer, provider.AuthUrl, provider.TokenUrl, provider.UserInfoUrl} {
		if value != "" && !strings.HasPrefix(value, "https://") && !strings.HasPrefix(value, "http://") {
			return fmt.Errorf("%s is not an http(s) URL", value)
		}
	}

	var old *AuthProvider
	var err error
	if originalName != "" {
		old, err = GetAuthProvider(originalName)
		if err != nil {
			return err
		}
		if old == nil {
			return fmt.Errorf("No provider %s", originalName)
		}
		// the name is in the callback URL registered at the provider and in the links to accounts
		if old.Name != provider.Name {
			return fmt.Errorf("The name of a provider cannot be changed")
		}
	} else {
		taken, err := GetAuthProvider(provider.Name)
		if err != nil {
			return err
		}
		if taken != nil {
			return fmt.Errorf("There is already a provider called %s", provider.Name)
		}
	}
	if provider.ClientSecret == SecretPlaceholder {
		provider.ClientSecret = ""
		if old != nil {
			provider.ClientSecret = old.ClientSecret
		}
	}

	if old == nil {
		_, err = adapter.Engine.Insert(provider)
		return err
	}
	_, err = adapter.Engine.ID(old.Name).AllCols().Omit("name", "created_at").Update(provider)
	discoveryCache.Delete(old.Issuer)
	return err
}

// DeleteAuthProvider removes a provider; its members keep their accounts but can no longer log in with it.
func DeleteAuthProvider(name string) error {
	_, err := adapter.Engine.ID(name).Delete(&AuthProvider{})
	return err
}

// CanTurnOffProvider tells whether members could still log in without the provider called name.
func CanTurnOffProvider(name string) bool {
	if GetSiteSettingBool("enable_local_logins") {
		return true
	}
	config := GetCasdoorConfig()
	if config.ServerUrl != "" && config.ClientId != "" {
		return true
	}
	for _, provider := range GetPublicAuthProviders() {
		if provider.Name != name {
			return true
		}
	}
	return false
}

// HasOtherWayToLogIn tells whether members could still log in without a password: Casdoor or another provider.
func HasOtherWayToLogIn() bool {
	config := GetCasdoorConfig()
	return (config.ServerUrl != "" && config.ClientId != "") || len(GetPublicAuthProviders()) > 0
}

type oidcDiscovery struct {
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserinfoEndpoint      string `json:"userinfo_endpoint"`
	fetchedAt             time.Time
}

var discoveryCache sync.Map

func discover(issuer string) (*oidcDiscovery, error) {
	if cached, ok := discoveryCache.Load(issuer); ok && time.Since(cached.(*oidcDiscovery).fetchedAt) < time.Hour {
		return cached.(*oidcDiscovery), nil
	}
	body, err := oauthGet(issuer+"/.well-known/openid-configuration", "")
	if err != nil {
		return nil, fmt.Errorf("reading the OpenID configuration of %s: %v", issuer, err)
	}
	res := &oidcDiscovery{}
	err = json.Unmarshal(body, res)
	if err != nil || res.AuthorizationEndpoint == "" || res.TokenEndpoint == "" {
		return nil, fmt.Errorf("%s/.well-known/openid-configuration is not an OpenID configuration", issuer)
	}
	res.fetchedAt = time.Now()
	discoveryCache.Store(issuer, res)
	return res, nil
}

// endpoints are the authorization, token and user info URLs, from the issuer unless set by hand.
func (p *AuthProvider) endpoints() (string, string, string, error) {
	authUrl, tokenUrl, userInfoUrl := p.AuthUrl, p.TokenUrl, p.UserInfoUrl
	if p.Type == "oidc" && p.Issuer != "" && (authUrl == "" || tokenUrl == "" || userInfoUrl == "") {
		config, err := discover(p.Issuer)
		if err != nil {
			return "", "", "", err
		}
		if authUrl == "" {
			authUrl = config.AuthorizationEndpoint
		}
		if tokenUrl == "" {
			tokenUrl = config.TokenEndpoint
		}
		if userInfoUrl == "" {
			userInfoUrl = config.UserinfoEndpoint
		}
	}
	if userInfoUrl == "" {
		return "", "", "", fmt.Errorf("%s has no user info endpoint", p.Name)
	}
	return authUrl, tokenUrl, userInfoUrl, nil
}

func (p *AuthProvider) scopes() string {
	if strings.TrimSpace(p.Scopes) != "" {
		return strings.Join(strings.Fields(strings.ReplaceAll(p.Scopes, ",", " ")), " ")
	}
	if p.Type == "oidc" {
		return "openid profile email"
	}
	return ""
}

func codeChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// GetAuthorizeUrl is where to send the browser to log in, with PKCE.
func (p *AuthProvider) GetAuthorizeUrl(redirectUri string, state string, verifier string) (string, error) {
	authUrl, _, _, err := p.endpoints()
	if err != nil {
		return "", err
	}
	params := url.Values{
		"client_id":             {p.ClientId},
		"redirect_uri":          {redirectUri},
		"response_type":         {"code"},
		"state":                 {state},
		"code_challenge":        {codeChallenge(verifier)},
		"code_challenge_method": {"S256"},
	}
	if scopes := p.scopes(); scopes != "" {
		params.Set("scope", scopes)
	}
	separator := "?"
	if strings.Contains(authUrl, "?") {
		separator = "&"
	}
	return authUrl + separator + params.Encode(), nil
}

func oauthGet(target string, accessToken string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	// GitHub refuses requests without one
	req.Header.Set("User-Agent", "Casnode")
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}
	resp, err := oauthHttpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s returned %d: %s", target, resp.StatusCode, truncate(string(body), 300))
	}
	return body, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// GetIdentity trades the code from the callback for an access token, then asks the provider who it belongs to. The
// answer comes straight from the provider over HTTPS, so no ID token signature needs checking.
func (p *AuthProvider) GetIdentity(code string, redirectUri string, verifier string) (*OAuthIdentity, error) {
	_, tokenUrl, userInfoUrl, err := p.endpoints()
	if err != nil {
		return nil, err
	}

	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectUri},
		"client_id":     {p.ClientId},
		"client_secret": {p.ClientSecret},
		"code_verifier": {verifier},
	}
	req, err := http.NewRequest(http.MethodPost, tokenUrl, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Casnode")
	resp, err := oauthHttpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}

	var token struct {
		AccessToken      string `json:"access_token"`
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	mediaType, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if mediaType == "application/x-www-form-urlencoded" || mediaType == "text/plain" {
		values, _ := url.ParseQuery(string(body))
		token.AccessToken, token.Error, token.ErrorDescription = values.Get("access_token"), values.Get("error"), values.Get("error_description")
	} else if err = json.Unmarshal(body, &token); err != nil {
		return nil, fmt.Errorf("%s returned %d: %s", tokenUrl, resp.StatusCode, truncate(string(body), 300))
	}
	if token.AccessToken == "" {
		return nil, fmt.Errorf("%s: %s %s", p.DisplayName, token.Error, token.ErrorDescription)
	}

	body, err = oauthGet(userInfoUrl, token.AccessToken)
	if err != nil {
		return nil, err
	}
	info := map[string]interface{}{}
	err = json.Unmarshal(body, &info)
	if err != nil {
		return nil, fmt.Errorf("the user info of %s is not JSON: %s", p.DisplayName, truncate(string(body), 300))
	}

	identity := &OAuthIdentity{
		Subject:  fieldString(info, p.IdField, "sub"),
		Username: fieldString(info, p.UsernameField, "preferred_username"),
		Name:     fieldString(info, p.NameField, "name"),
		Email:    fieldString(info, p.EmailField, "email"),
		Avatar:   fieldString(info, p.AvatarField, "picture"),
	}
	if identity.Subject == "" {
		return nil, fmt.Errorf("the user info of %s has no user ID", p.DisplayName)
	}
	// GitHub leaves out an email the user keeps private; the user:email scope can still read it
	if identity.Email == "" && strings.HasPrefix(userInfoUrl, "https://api.github.com/") {
		identity.Email = githubPrimaryEmail(token.AccessToken)
	}
	return identity, nil
}

// fieldString reads a value of the user info, by a dotted path such as "data.user.id".
func fieldString(info map[string]interface{}, field string, fallback string) string {
	if field == "" {
		field = fallback
	}
	var value interface{} = info
	for _, key := range strings.Split(field, ".") {
		m, ok := value.(map[string]interface{})
		if !ok {
			return ""
		}
		value = m[key]
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case json.Number:
		return v.String()
	}
	return ""
}

func githubPrimaryEmail(accessToken string) string {
	body, err := oauthGet("https://api.github.com/user/emails", accessToken)
	if err != nil {
		return ""
	}
	emails := []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}{}
	if json.Unmarshal(body, &emails) != nil {
		return ""
	}
	for _, email := range emails {
		if email.Primary && email.Verified {
			return email.Email
		}
	}
	return ""
}

var usernameCleanRegex = regexp.MustCompile(`[^a-zA-Z0-9_.\-]+`)

// freeUsername turns what a provider calls the user into a user name nobody here has.
func freeUsername(candidates ...string) (string, error) {
	base := ""
	for _, candidate := range candidates {
		candidate = strings.Trim(usernameCleanRegex.ReplaceAllString(candidate, ""), "._-")
		if len(candidate) >= 3 {
			base = candidate
			break
		}
	}
	if base == "" {
		base = "user"
	}
	if len(base) > 17 {
		base = base[:17]
	}
	for i := 0; i < 1000; i++ {
		username := base
		if i > 0 {
			username = fmt.Sprintf("%s%d", base, i)
		}
		if validateUsername(username) != nil {
			continue
		}
		user, err := GetUserByUsername(username)
		if err != nil {
			return "", err
		}
		if user == nil {
			return username, nil
		}
	}
	return "", fmt.Errorf("no free user name for %s", base)
}

func getLinkedUser(provider string, subject string) (*User, error) {
	link := &UserAuth{}
	has, err := adapter.Engine.Where("provider = ? AND subject = ?", provider, subject).Get(link)
	if err != nil || !has {
		return nil, err
	}
	return GetUser(link.UserId)
}

func linkUser(provider string, subject string, userId int64) error {
	_, err := adapter.Engine.Insert(&UserAuth{Provider: provider, Subject: subject, UserId: userId})
	return err
}

// SignInWithProvider finds the user linked to the provider account, or creates one. An account is never linked to an
// existing user by email: a local account's email is not verified, so that would hand it to whoever typed it.
func SignInWithProvider(provider *AuthProvider, identity *OAuthIdentity) (*User, error) {
	user, err := getLinkedUser(provider.Name, identity.Subject)
	if err != nil {
		return nil, err
	}
	if user != nil {
		if identity.Avatar != "" && identity.Avatar != user.Avatar {
			user.Avatar = identity.Avatar
			_, err = adapter.Engine.ID(user.Id).Cols("avatar").Update(user)
		}
		return user, err
	}

	if !provider.AllowSignup {
		return nil, fmt.Errorf("There is no account here for this %s user, and new accounts cannot be created with it", provider.DisplayName)
	}
	emailLocal := ""
	if at := strings.Index(identity.Email, "@"); at > 0 {
		emailLocal = identity.Email[:at]
	}
	username, err := freeUsername(identity.Username, emailLocal, identity.Name)
	if err != nil {
		return nil, err
	}
	name := identity.Name
	if name == "" {
		name = username
	}
	user = &User{
		Username:  username,
		Name:      truncate(name, 90),
		Avatar:    identity.Avatar,
		Email:     identity.Email,
		LoginType: LoginTypeOAuth,
	}
	err = insertNewUser(user)
	if err != nil {
		return nil, err
	}
	return user, linkUser(provider.Name, identity.Subject, user.Id)
}
