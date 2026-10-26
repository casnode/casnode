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
	"time"

	"github.com/casnode/casnode/object"
)

// GetDashboard
// @Title GetDashboard
// @Description the numbers on the admin dashboard
// @router /get-dashboard [get]
func (c *ApiController) GetDashboard() {
	if c.RequireStaff() == nil {
		return
	}
	res := map[string]int64{}
	periods := map[string]time.Time{
		"":      {},
		"Today": time.Now().AddDate(0, 0, -1),
		"Week":  time.Now().AddDate(0, 0, -7),
		"Month": time.Now().AddDate(0, -1, 0),
	}
	var err error
	for suffix, since := range periods {
		res["topics"+suffix], err = object.CountTopics(since)
		if c.responseErr(err) {
			return
		}
		res["posts"+suffix], err = object.CountPosts(since)
		if c.responseErr(err) {
			return
		}
	}
	res["users"], err = object.CountUsers()
	if c.responseErr(err) {
		return
	}
	res["reviewables"], err = object.CountPendingReviewables()
	if c.responseErr(err) {
		return
	}
	c.ResponseOk(res)
}

// GetSiteSettings
// @Title GetSiteSettings
// @router /get-site-settings [get]
func (c *ApiController) GetSiteSettings() {
	if c.RequireAdmin() == nil {
		return
	}
	c.ResponseOk(object.GetAllSettings())
}

type settingInput struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// UpdateSiteSetting
// @Title UpdateSiteSetting
// @router /update-site-setting [post]
func (c *ApiController) UpdateSiteSetting() {
	if c.RequireAdmin() == nil {
		return
	}
	input := &settingInput{}
	if !c.readJsonBody(input) {
		return
	}
	if c.responseErr(object.SetSiteSetting(input.Name, input.Value)) {
		return
	}
	c.ResponseOk()
}

// GetAdminUsers
// @Title GetAdminUsers
// @Description users for admins: filter active|new|staff|suspended|silenced
// @router /get-admin-users [get]
func (c *ApiController) GetAdminUsers() {
	if c.RequireStaff() == nil {
		return
	}
	users, more, err := object.GetAdminUsers(c.GetString("filter"), c.GetString("name"), c.getInt("page"))
	if c.responseErr(err) {
		return
	}
	c.ResponseOk(users, more)
}

type staffUserInput struct {
	Id            int64     `json:"id"`
	Admin         bool      `json:"admin"`
	Moderator     bool      `json:"moderator"`
	TrustLevel    int       `json:"trustLevel"`
	Title         string    `json:"title"`
	SuspendedTill time.Time `json:"suspendedTill"`
	SuspendReason string    `json:"suspendReason"`
	SilencedTill  time.Time `json:"silencedTill"`
}

// UpdateAdminUser
// @Title UpdateAdminUser
// @Description grants or revokes admin and moderator, suspends or silences a user
// @router /update-admin-user [post]
func (c *ApiController) UpdateAdminUser() {
	viewer := c.RequireStaff()
	if viewer == nil {
		return
	}
	input := &staffUserInput{}
	if !c.readJsonBody(input) {
		return
	}
	user, err := object.GetUser(input.Id)
	if c.responseErr(err) {
		return
	}
	if user == nil {
		c.ResponseError("The user does not exist")
		return
	}
	if (input.Admin != user.Admin || input.Moderator != user.Moderator) && !viewer.Admin {
		c.ResponseError("Only administrators can change roles")
		return
	}
	if user.Admin && !viewer.Admin {
		c.ResponseError("Moderators can't change administrators")
		return
	}
	if user.Id == viewer.Id && !input.Admin && viewer.Admin {
		c.ResponseError("You can't revoke your own admin")
		return
	}
	user.Admin = input.Admin
	user.Moderator = input.Moderator
	user.TrustLevel = input.TrustLevel
	user.Title = input.Title
	user.SuspendedTill = input.SuspendedTill
	user.SuspendReason = input.SuspendReason
	user.SilencedTill = input.SilencedTill
	if c.responseErr(object.UpdateUserStaffFields(user)) {
		return
	}
	c.ResponseOk(user)
}

// GetReviewables
// @Title GetReviewables
// @Description the review queue: status 0 pending, 1 approved, 2 rejected, 3 ignored, 4 deleted
// @router /get-reviewables [get]
func (c *ApiController) GetReviewables() {
	if c.RequireStaff() == nil {
		return
	}
	list, err := object.GetReviewables(c.getInt("status"))
	if c.responseErr(err) {
		return
	}
	c.ResponseOk(list)
}

// PerformReviewable
// @Title PerformReviewable
// @Description acts on a flagged post: agree (hide it), disagree, delete, ignore
// @router /perform-reviewable [post]
func (c *ApiController) PerformReviewable() {
	user := c.RequireStaff()
	if user == nil {
		return
	}
	if c.responseErr(object.PerformReviewable(user, c.getInt64("id"), c.GetString("action"))) {
		return
	}
	c.ResponseOk()
}

// SendTestEmail
// @Title SendTestEmail
// @Description sends a test email with the mail settings, to check them
// @router /send-test-email [post]
func (c *ApiController) SendTestEmail() {
	if c.RequireAdmin() == nil {
		return
	}
	if c.responseErr(object.SendTestEmail(c.GetString("to"))) {
		return
	}
	c.ResponseOk()
}
