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
	"github.com/casnode/casnode/object"
)

// GetBadges
// @Title GetBadges
// @Description the enabled badges; all=true lists the disabled ones too, for admins
// @router /get-badges [get]
func (c *ApiController) GetBadges() {
	user := c.GetSessionUser()
	all := c.GetString("all") == "true" && user != nil && user.Admin
	badges, err := object.GetBadges(user, all)
	if c.responseErr(err) {
		return
	}
	c.ResponseOk(badges)
}

// GetBadge
// @Title GetBadge
// @Description a badge with who was granted it; username= shows only that user's grants
// @router /get-badge [get]
func (c *ApiController) GetBadge() {
	user := c.GetSessionUser()
	badge, err := object.GetBadge(c.getInt64("id"))
	if c.responseErr(err) {
		return
	}
	if badge == nil || (!badge.Enabled && !(user != nil && user.Admin)) {
		c.ResponseError("The badge does not exist")
		return
	}
	userId := int64(0)
	if username := c.GetString("username"); username != "" {
		target := c.loadUser(username)
		if target == nil {
			return
		}
		userId = target.Id
	}
	grants, more, err := object.GetBadgeGrants(user, badge.Id, userId, c.getInt("page"))
	if c.responseErr(err) {
		return
	}
	c.ResponseOk(badge, map[string]interface{}{"grants": grants, "more": more})
}

// GetUserBadges
// @Title GetUserBadges
// @router /get-user-badges [get]
func (c *ApiController) GetUserBadges() {
	user := c.loadUser(c.GetString("username"))
	if user == nil {
		return
	}
	badges, err := object.GetUserBadges(user.Id)
	if c.responseErr(err) {
		return
	}
	c.ResponseOk(badges)
}

// AddBadge
// @Title AddBadge
// @router /add-badge [post]
func (c *ApiController) AddBadge() {
	if c.RequireAdmin() == nil {
		return
	}
	badge := &object.Badge{}
	if !c.readJsonBody(badge) {
		return
	}
	if c.responseErr(object.AddBadge(badge)) {
		return
	}
	c.ResponseOk(badge)
}

// UpdateBadge
// @Title UpdateBadge
// @router /update-badge [post]
func (c *ApiController) UpdateBadge() {
	if c.RequireAdmin() == nil {
		return
	}
	badge := &object.Badge{}
	if !c.readJsonBody(badge) {
		return
	}
	if c.responseErr(object.UpdateBadge(badge)) {
		return
	}
	c.ResponseOk(badge)
}

// DeleteBadge
// @Title DeleteBadge
// @router /delete-badge [post]
func (c *ApiController) DeleteBadge() {
	if c.RequireAdmin() == nil {
		return
	}
	if c.responseErr(object.DeleteBadge(c.getInt64("id"))) {
		return
	}
	c.ResponseOk()
}

type grantBadgeInput struct {
	BadgeId  int64  `json:"badgeId"`
	Username string `json:"username"`
	Reason   string `json:"reason"`
}

// GrantBadge
// @Title GrantBadge
// @router /grant-badge [post]
func (c *ApiController) GrantBadge() {
	admin := c.RequireAdmin()
	if admin == nil {
		return
	}
	input := &grantBadgeInput{}
	if !c.readJsonBody(input) {
		return
	}
	if c.responseErr(object.GrantBadgeManually(admin, input.BadgeId, input.Username, input.Reason)) {
		return
	}
	c.ResponseOk()
}

// RevokeBadge
// @Title RevokeBadge
// @Description takes back one grant of a badge (the id of the grant)
// @router /revoke-badge [post]
func (c *ApiController) RevokeBadge() {
	if c.RequireAdmin() == nil {
		return
	}
	if c.responseErr(object.RevokeBadge(c.getInt64("id"))) {
		return
	}
	c.ResponseOk()
}
