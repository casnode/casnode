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

package controllers

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/beego/beego/v2/core/logs"
	"github.com/casnode/casnode/conf"
	"github.com/casnode/casnode/object"
)

var (
	failedLoginsByIp    = object.NewRateLimiter(20, 15*time.Minute)
	failedLoginsByLogin = object.NewRateLimiter(10, 15*time.Minute)
	signupsByIp         = object.NewRateLimiter(5, time.Hour)
	resetEmailsByIp     = object.NewRateLimiter(5, time.Hour)
	resetEmailsByLogin  = object.NewRateLimiter(3, time.Hour)
)

const (
	tooManyAttempts = "Too many attempts; wait a few minutes and try again"
	onlyWayToLogIn  = "This is the only way left to log in; turn on enable_local_logins first"
)

// siteOrigin is the address members reach the forum at: site_url, or what the request came to.
func (c *ApiController) siteOrigin() string {
	siteUrl := strings.TrimSuffix(object.GetSiteSetting("site_url"), "/")
	if siteUrl != "" {
		return siteUrl
	}
	scheme := "http"
	if c.Ctx.Request.TLS != nil {
		scheme = "https"
	}
	host := c.Ctx.Request.Host
	if conf.GetConfigBool("trustProxyHeader") {
		if proto := c.Ctx.Request.Header.Get("X-Forwarded-Proto"); proto == "http" || proto == "https" {
			scheme = proto
		}
		if forwardedHost := c.Ctx.Request.Header.Get("X-Forwarded-Host"); forwardedHost != "" {
			host = forwardedHost
		}
	}
	return scheme + "://" + host
}

// signIn starts the session of a user who proved who they are.
func (c *ApiController) signIn(user *object.User, how string) error {
	if user.IsSuspended() {
		return fmt.Errorf("This account is suspended until %s", user.SuspendedTill.Format("2006-01-02 15:04"))
	}
	err := c.SetSessionUser(user)
	if err != nil {
		return err
	}
	logs.Info("signin (%s): %s from %s", how, user.Username, c.getClientIp())
	return nil
}

type signupForm struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

// Signup
// @Title Signup
// @Description creates a local account with a password and signs it in
// @router /signup [post]
func (c *ApiController) Signup() {
	var form signupForm
	if !c.readJsonBody(&form) {
		return
	}
	ip := c.getClientIp()
	if !signupsByIp.Allowed(ip) {
		c.ResponseError(tooManyAttempts)
		return
	}
	user, err := object.SignUp(form.Username, form.Email, form.Name, form.Password)
	if c.responseErr(err) {
		return
	}
	signupsByIp.Add(ip)
	if c.responseErr(c.signIn(user, "signup")) {
		return
	}
	c.ResponseOk(user)
}

type loginForm struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// Login
// @Title Login
// @Description logs in with a user name or email and a password
// @router /login [post]
func (c *ApiController) Login() {
	var form loginForm
	if !c.readJsonBody(&form) {
		return
	}
	ip := c.getClientIp()
	loginKey := strings.ToLower(strings.TrimSpace(form.Login))
	if !failedLoginsByIp.Allowed(ip) || !failedLoginsByLogin.Allowed(loginKey) {
		c.ResponseError(tooManyAttempts)
		return
	}
	user, err := object.LogIn(form.Login, form.Password)
	if err != nil {
		failedLoginsByIp.Add(ip)
		failedLoginsByLogin.Add(loginKey)
		c.ResponseError(err.Error())
		return
	}
	if c.responseErr(c.signIn(user, "password")) {
		return
	}
	c.ResponseOk(user)
}

// ForgotPassword
// @Title ForgotPassword
// @Description emails a link for setting a new password; the answer is the same whether the account exists or not
// @Param login query string true "user name or email"
// @router /forgot-password [post]
func (c *ApiController) ForgotPassword() {
	login := strings.TrimSpace(c.GetString("login"))
	if login == "" {
		c.ResponseError("Enter your user name or email")
		return
	}
	ip := c.getClientIp()
	loginKey := strings.ToLower(login)
	if !resetEmailsByIp.Allowed(ip) || !resetEmailsByLogin.Allowed(loginKey) {
		c.ResponseError(tooManyAttempts)
		return
	}
	resetEmailsByIp.Add(ip)
	resetEmailsByLogin.Add(loginKey)
	err := object.SendPasswordResetEmail(login, c.siteOrigin())
	if err != nil {
		logs.Warn("password reset email for %s: %v", login, err)
		if !object.CanResetPassword() {
			c.ResponseError(err.Error())
			return
		}
	}
	c.ResponseOk()
}

type resetPasswordForm struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

// ResetPassword
// @Title ResetPassword
// @Description sets a new password through the emailed link and signs in
// @router /reset-password [post]
func (c *ApiController) ResetPassword() {
	var form resetPasswordForm
	if !c.readJsonBody(&form) {
		return
	}
	user, err := object.ResetPassword(form.Token, form.Password)
	if c.responseErr(err) {
		return
	}
	if c.responseErr(c.signIn(user, "password reset")) {
		return
	}
	c.ResponseOk(user)
}

type changePasswordForm struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

// ChangePassword
// @Title ChangePassword
// @Description changes the password of the signed-in local account
// @router /change-password [post]
func (c *ApiController) ChangePassword() {
	user := c.RequireSignedInUser()
	if user == nil {
		return
	}
	var form changePasswordForm
	if !c.readJsonBody(&form) {
		return
	}
	key := fmt.Sprintf("u%d", user.Id)
	if !failedLoginsByLogin.Allowed(key) {
		c.ResponseError(tooManyAttempts)
		return
	}
	err := object.ChangePassword(user, form.CurrentPassword, form.NewPassword)
	if err != nil {
		failedLoginsByLogin.Add(key)
		c.ResponseError(err.Error())
		return
	}
	c.ResponseOk()
}

func (c *ApiController) providerCallbackUrl(name string) string {
	return c.siteOrigin() + "/api/auth/callback/" + name
}

// safeReturnTo keeps the browser on this site after logging in.
func safeReturnTo(returnTo string) string {
	if !strings.HasPrefix(returnTo, "/") || strings.HasPrefix(returnTo, "//") || strings.HasPrefix(returnTo, "/\\") ||
		strings.HasPrefix(returnTo, "/login") || strings.HasPrefix(returnTo, "/signup") {
		return "/"
	}
	return returnTo
}

// AuthStart
// @Title AuthStart
// @Description sends the browser to a single sign-on provider
// @Param provider query string true "provider name"
// @Param return_to query string false "the page to come back to"
// @router /auth/start [get]
func (c *ApiController) AuthStart() {
	provider, err := object.GetAuthProvider(c.GetString("provider"))
	if err != nil || provider == nil || !provider.Enabled {
		c.authFailed("No such login provider")
		return
	}
	state := object.NewRandomToken()
	verifier := object.NewRandomToken() + object.NewRandomToken()
	authorizeUrl, err := provider.GetAuthorizeUrl(c.providerCallbackUrl(provider.Name), state, verifier)
	if err != nil {
		logs.Warn("auth start %s: %v", provider.Name, err)
		c.authFailed(err.Error())
		return
	}
	err = c.SetSession("oauthState", provider.Name+":"+state)
	if err == nil {
		err = c.SetSession("oauthVerifier", verifier)
	}
	if err == nil {
		err = c.SetSession("oauthReturnTo", safeReturnTo(c.GetString("return_to")))
	}
	if err != nil {
		c.authFailed(err.Error())
		return
	}
	c.Redirect(authorizeUrl, 302)
}

// authFailed goes back to the login page, which shows the error.
func (c *ApiController) authFailed(message string) {
	c.Redirect("/login?error="+url.QueryEscape(message), 302)
}

// AuthCallback
// @Title AuthCallback
// @Description where a single sign-on provider sends the browser back to, with the code
// @router /auth/callback/:provider [get]
func (c *ApiController) AuthCallback() {
	name := c.Ctx.Input.Param(":provider")
	state, _ := c.GetSession("oauthState").(string)
	verifier, _ := c.GetSession("oauthVerifier").(string)
	returnTo, _ := c.GetSession("oauthReturnTo").(string)
	_ = c.DelSession("oauthState")
	_ = c.DelSession("oauthVerifier")
	_ = c.DelSession("oauthReturnTo")

	if message := c.GetString("error_description"); message != "" || c.GetString("error") != "" {
		if message == "" {
			message = c.GetString("error")
		}
		c.authFailed(message)
		return
	}
	if state == "" || state != name+":"+c.GetString("state") {
		c.authFailed("The login took too long or was started in another browser; try again")
		return
	}
	provider, err := object.GetAuthProvider(name)
	if err != nil || provider == nil || !provider.Enabled {
		c.authFailed("No such login provider")
		return
	}
	identity, err := provider.GetIdentity(c.GetString("code"), c.providerCallbackUrl(name), verifier)
	if err != nil {
		logs.Warn("auth callback %s: %v", name, err)
		c.authFailed("Logging in with " + provider.DisplayName + " failed: " + err.Error())
		return
	}
	user, err := object.SignInWithProvider(provider, identity)
	if err == nil {
		err = c.signIn(user, name)
	}
	if err != nil {
		c.authFailed(err.Error())
		return
	}
	c.Redirect(safeReturnTo(returnTo), 302)
}

type authProvidersInfo struct {
	Providers   []*object.AuthProvider `json:"providers"`
	CallbackUrl string                 `json:"callbackUrl"`
}

// GetAuthProviders
// @Title GetAuthProviders
// @Description lists the single sign-on providers for /admin/logins, without their secrets
// @router /get-auth-providers [get]
func (c *ApiController) GetAuthProviders() {
	if c.RequireAdmin() == nil {
		return
	}
	providers, err := object.GetAuthProviders()
	if c.responseErr(err) {
		return
	}
	masked := []*object.AuthProvider{}
	for _, provider := range providers {
		masked = append(masked, provider.MaskSecret())
	}
	c.ResponseOk(&authProvidersInfo{Providers: masked, CallbackUrl: c.providerCallbackUrl("")})
}

type saveAuthProviderForm struct {
	OriginalName string               `json:"originalName"`
	Provider     *object.AuthProvider `json:"provider"`
}

// SaveAuthProvider
// @Title SaveAuthProvider
// @Description adds or changes a single sign-on provider
// @router /save-auth-provider [post]
func (c *ApiController) SaveAuthProvider() {
	if c.RequireAdmin() == nil {
		return
	}
	var form saveAuthProviderForm
	if !c.readJsonBody(&form) {
		return
	}
	if form.Provider == nil {
		c.ResponseError("Missing the provider")
		return
	}
	if form.OriginalName != "" && !form.Provider.Enabled && !object.CanTurnOffProvider(form.OriginalName) {
		c.ResponseError(onlyWayToLogIn)
		return
	}
	if c.responseErr(object.SaveAuthProvider(form.OriginalName, form.Provider)) {
		return
	}
	c.ResponseOk(form.Provider.MaskSecret())
}

// DeleteAuthProvider
// @Title DeleteAuthProvider
// @Description removes a single sign-on provider
// @Param name query string true "provider name"
// @router /delete-auth-provider [post]
func (c *ApiController) DeleteAuthProvider() {
	if c.RequireAdmin() == nil {
		return
	}
	name := c.GetString("name")
	if !object.CanTurnOffProvider(name) {
		c.ResponseError(onlyWayToLogIn)
		return
	}
	if c.responseErr(object.DeleteAuthProvider(name)) {
		return
	}
	c.ResponseOk()
}
