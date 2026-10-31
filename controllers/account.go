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
	"net"
	"regexp"
	"strings"

	"github.com/beego/beego/v2/core/logs"
	"github.com/casnode/casnode/conf"
	"github.com/casnode/casnode/object"
)

// Signin
// @Title Signin
// @Description signs in with the code Casdoor sends to /callback (the providers on /admin/logins use /api/auth/callback)
// @Param code query string true "OAuth code"
// @router /signin [post]
func (c *ApiController) Signin() {
	code := c.GetString("code")
	if code == "" {
		c.ResponseError("Missing the sign-in code")
		return
	}

	account, err := object.GetAccountByOAuthCode(code)
	if err != nil {
		logs.Warn("signin failed: %v", err)
		c.ResponseError("Sign-in failed: " + err.Error())
		return
	}
	user, err := object.SyncCasdoorUser(account)
	if c.responseErr(err) {
		return
	}
	err = c.SetSessionUser(user)
	if c.responseErr(err) {
		return
	}

	logs.Info("signin: %s from %s", user.Username, c.getClientIp())
	c.ResponseOk(user)
}

var devUsernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_.\-]{1,30}$`)

// DevSignin
// @Title DevSignin
// @Description signs in as any user name without Casdoor, when devSignin = true and the request comes from this computer
// @router /dev-signin [post]
func (c *ApiController) DevSignin() {
	// the address of the connection itself, not a forwarded header anyone can set; and only loopback addresses
	// forwarded, since a reverse proxy on this computer (CasWAF, Nginx) connects from 127.0.0.1 too, while `yarn dev`
	// passes on the browser's 127.0.0.1
	host, _, _ := net.SplitHostPort(c.Ctx.Request.RemoteAddr)
	if !conf.GetConfigBool("devSignin") || !isLoopback(host) || !onlyLoopbackForwarded(c.Ctx.Request.Header.Get("X-Forwarded-For")) || c.Ctx.Request.Header.Get("X-Real-IP") != "" {
		c.ResponseError("Sign-in without Casdoor is turned off")
		return
	}

	username := c.GetString("username")
	if username == "" {
		username = "admin"
	}
	if !devUsernameRegex.MatchString(username) || username == object.SystemUsername {
		c.ResponseError("Invalid user name")
		return
	}
	user, err := object.EnsureLocalUser(username)
	if c.responseErr(err) {
		return
	}
	err = c.SetSessionUser(user)
	if c.responseErr(err) {
		return
	}
	c.ResponseOk(user)
}

func isLoopback(host string) bool {
	ip := net.ParseIP(strings.TrimSpace(host))
	return ip != nil && ip.IsLoopback()
}

func onlyLoopbackForwarded(forwardedFor string) bool {
	if forwardedFor == "" {
		return true
	}
	for _, item := range strings.Split(forwardedFor, ",") {
		if !isLoopback(item) {
			return false
		}
	}
	return true
}

// Signout
// @Title Signout
// @Description signs out
// @router /signout [post]
func (c *ApiController) Signout() {
	err := c.SetSessionUser(nil)
	if c.responseErr(err) {
		return
	}
	c.ResponseOk()
}

// GetAccount
// @Title GetAccount
// @Description returns the signed-in user, or null
// @router /get-account [get]
func (c *ApiController) GetAccount() {
	c.ResponseOk(c.GetSessionUser())
}

type siteInfo struct {
	Settings      map[string]string            `json:"settings"`
	Casdoor       *object.CasdoorConfig        `json:"casdoor"`
	AuthProviders []*object.PublicAuthProvider `json:"authProviders"`
	PasswordReset bool                         `json:"passwordReset"`
	DevSignin     bool                         `json:"devSignin"`
	Categories    []*object.Category           `json:"categories"`
	TopTags       []*object.Tag                `json:"topTags"`
	CurrentUser   *object.User                 `json:"currentUser"`
	Counts        map[string]int64             `json:"counts"`
	DefaultLocale string                       `json:"defaultLocale"`
}

// GetSite
// @Title GetSite
// @Description returns what every page needs: the site settings, the categories, the top tags, and the signed-in user with their counts
// @router /get-site [get]
func (c *ApiController) GetSite() {
	user := c.GetSessionUser()
	categories, err := object.GetCategories(user)
	if c.responseErr(err) {
		return
	}
	tags, err := object.GetTags(30)
	if c.responseErr(err) {
		return
	}

	counts := map[string]int64{}
	if user != nil {
		counts["new"], counts["unread"], err = object.CountNewAndUnread(user)
		if c.responseErr(err) {
			return
		}
		counts["notifications"], counts["highPriorityNotifications"], err = object.CountUnreadNotifications(user)
		if c.responseErr(err) {
			return
		}
		counts["unreadMessages"], err = object.CountUnreadPrivateMessages(user)
		if c.responseErr(err) {
			return
		}
		counts["chatChannels"], counts["chatDirect"], err = object.CountUnreadChat(user)
		if c.responseErr(err) {
			return
		}
		if user.IsStaff() {
			counts["reviewables"], err = object.CountPendingReviewables()
			if c.responseErr(err) {
				return
			}
		}
	}

	c.ResponseOk(&siteInfo{
		Settings:      object.GetPublicSettings(),
		Casdoor:       object.GetCasdoorConfig(),
		AuthProviders: object.GetPublicAuthProviders(),
		PasswordReset: object.CanResetPassword(),
		DevSignin:     conf.GetConfigBool("devSignin"),
		Categories:    object.SortCategoriesForDisplay(categories),
		TopTags:       tags,
		CurrentUser:   user,
		Counts:        counts,
		DefaultLocale: object.GetSiteSetting("default_locale"),
	})
}
