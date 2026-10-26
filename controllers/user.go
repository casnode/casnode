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

func (c *ApiController) loadUser(username string) *object.User {
	user, err := object.GetUserByUsername(username)
	if c.responseErr(err) {
		return nil
	}
	if user == nil {
		c.ResponseError("The user does not exist")
		return nil
	}
	return user
}

// GetUser
// @Title GetUser
// @Description returns a user's profile; summary=true adds the Summary tab
// @router /get-user [get]
func (c *ApiController) GetUser() {
	viewer := c.GetSessionUser()
	user := c.loadUser(c.GetString("username"))
	if user == nil {
		return
	}

	profile := user.Public()
	if viewer != nil && (viewer.Id == user.Id || viewer.IsStaff()) {
		profile = user
	}
	if c.GetString("summary") == "true" {
		summary, err := object.GetUserSummary(viewer, user)
		if c.responseErr(err) {
			return
		}
		c.ResponseOk(profile, summary)
		return
	}
	c.ResponseOk(profile)
}

// GetUserActivity
// @Title GetUserActivity
// @Description a user's posts: filter all|replies|likes|bookmarks
// @router /get-user-activity [get]
func (c *ApiController) GetUserActivity() {
	viewer := c.GetSessionUser()
	user := c.loadUser(c.GetString("username"))
	if user == nil {
		return
	}

	filter := c.GetString("filter", "all")
	page := c.getInt("page")
	var posts []*object.UserPost
	var more bool
	var err error
	switch filter {
	case "likes":
		posts, more, err = object.GetUserLikes(viewer, user.Id, page)
	case "bookmarks":
		if viewer == nil || viewer.Id != user.Id {
			c.ResponseError("Bookmarks are private")
			return
		}
		posts, more, err = object.GetBookmarks(user, page)
	default:
		posts, more, err = object.GetUserPosts(viewer, user.Id, filter, page)
	}
	if c.responseErr(err) {
		return
	}

	userIds := []int64{user.Id}
	for _, item := range posts {
		userIds = append(userIds, item.Post.UserId)
	}
	users, err := object.GetUserMap(userIds)
	if c.responseErr(err) {
		return
	}
	c.ResponseOk(map[string]interface{}{"posts": posts, "users": users, "more": more})
}

type profileInput struct {
	Username string `json:"username"`
	BioRaw   string `json:"bioRaw"`
	Location string `json:"location"`
	Website  string `json:"website"`
	Title    string `json:"title"`
	Locale   string `json:"locale"`
	// "", "away", "always" or "never"; "" keeps the setting
	EmailLevel         string `json:"emailLevel"`
	EmailMessagesLevel string `json:"emailMessagesLevel"`
}

// UpdateProfile
// @Title UpdateProfile
// @Description changes the about me, location, website and interface language of a user (theirs, or any for staff)
// @router /update-profile [post]
func (c *ApiController) UpdateProfile() {
	viewer := c.RequireSignedInUser()
	if viewer == nil {
		return
	}
	input := &profileInput{}
	if !c.readJsonBody(input) {
		return
	}
	user := c.loadUser(input.Username)
	if user == nil {
		return
	}
	if user.Id != viewer.Id && !viewer.IsStaff() {
		c.ResponseError("You can't change this profile")
		return
	}
	user.BioRaw = input.BioRaw
	user.Location = input.Location
	user.Website = input.Website
	user.Locale = input.Locale
	if input.EmailLevel != "" {
		user.EmailLevel = object.EmailLevelValue(input.EmailLevel, false)
	}
	if input.EmailMessagesLevel != "" {
		user.EmailMessagesLevel = object.EmailLevelValue(input.EmailMessagesLevel, true)
	}
	if viewer.IsStaff() {
		user.Title = input.Title
	} else if input.Title != user.Title {
		// users pick their title from their badges and groups
		titles, err := object.GetBadgeTitles(user.Id)
		if c.responseErr(err) {
			return
		}
		allowed := input.Title == ""
		for _, title := range titles {
			allowed = allowed || title == input.Title
		}
		if !allowed {
			c.ResponseError("You can't use this title")
			return
		}
		user.Title = input.Title
	}
	if c.responseErr(object.UpdateUserProfile(user)) {
		return
	}
	c.ResponseOk(user)
}

// GetDirectory
// @Title GetDirectory
// @Description the Users page: order likes_received|likes_given|topic_count|post_count|topics_entered|posts_read|days_visited
// @router /get-directory [get]
func (c *ApiController) GetDirectory() {
	items, more, err := object.GetDirectory(c.GetString("order"), c.GetString("asc") == "true", c.GetString("name"), c.getInt("page"))
	if c.responseErr(err) {
		return
	}
	c.ResponseOk(items, more)
}

// SearchUsers
// @Title SearchUsers
// @Description user names starting with or containing term, for @mentions and message recipients
// @router /search-users [get]
func (c *ApiController) SearchUsers() {
	if c.RequireSignedInUser() == nil {
		return
	}
	users, err := object.SearchUsers(c.GetString("term"), 8)
	if c.responseErr(err) {
		return
	}
	// groups too, for @mentions (mentionable ones) and message recipients (messageable ones)
	if include := c.GetString("groups"); include == "mention" || include == "message" {
		groups, err := object.SearchGroupsForMention(c.GetSessionUser(), c.GetString("term"), include == "message")
		if c.responseErr(err) {
			return
		}
		c.ResponseOk(users, groups)
		return
	}
	c.ResponseOk(users)
}

// GetNotifications
// @Title GetNotifications
// @Description the signed-in user's notifications: filter all|unread|replies|likes|messages
// @router /get-notifications [get]
func (c *ApiController) GetNotifications() {
	user := c.RequireSignedInUser()
	if user == nil {
		return
	}
	pageSize := c.getInt("limit")
	if pageSize <= 0 || pageSize > 60 {
		pageSize = 60
	}
	list, err := object.GetNotifications(user, c.GetString("filter"), c.getInt("page"), pageSize)
	if c.responseErr(err) {
		return
	}
	c.ResponseOk(list)
}

// ReadNotification
// @Title ReadNotification
// @Description marks one notification read, or all of them without id
// @router /read-notification [post]
func (c *ApiController) ReadNotification() {
	user := c.RequireSignedInUser()
	if user == nil {
		return
	}
	if c.responseErr(object.MarkNotificationRead(user, c.getInt64("id"))) {
		return
	}
	c.ResponseOk()
}

type userCard struct {
	User       *object.User               `json:"user"`
	Badges     []*object.UserBadgeSummary `json:"badges"`
	BadgeCount int                        `json:"badgeCount"`
	Groups     []*object.Group            `json:"groups"`
	CanMessage bool                       `json:"canMessage"`
	CanChat    bool                       `json:"canChat"`
}

// GetUserCard
// @Title GetUserCard
// @Description what the card shown on clicking a user's avatar or name needs
// @router /get-user-card [get]
func (c *ApiController) GetUserCard() {
	viewer := c.GetSessionUser()
	user := c.loadUser(c.GetString("username"))
	if user == nil {
		return
	}
	badges, err := object.GetUserBadges(user.Id)
	if c.responseErr(err) {
		return
	}
	groups, err := object.GetUserGroups(viewer, user.Id)
	if c.responseErr(err) {
		return
	}
	card := &userCard{User: user.Public(), Badges: badges, BadgeCount: len(badges), Groups: groups}
	if len(card.Badges) > 3 {
		card.Badges = card.Badges[:3]
	}
	if viewer != nil && viewer.Id != user.Id && user.Username != object.SystemUsername {
		card.CanMessage = object.GetSiteSettingBool("enable_personal_messages") || viewer.IsStaff()
		card.CanChat = object.IsChatEnabled() && !viewer.IsSuspended()
	}
	c.ResponseOk(card)
}

// GetUserPreferences
// @Title GetUserPreferences
// @Description the settings only the user and staff see: email levels and the titles they may pick
// @router /get-user-preferences [get]
func (c *ApiController) GetUserPreferences() {
	viewer := c.RequireSignedInUser()
	if viewer == nil {
		return
	}
	user := c.loadUser(c.GetString("username"))
	if user == nil {
		return
	}
	if user.Id != viewer.Id && !viewer.IsStaff() {
		c.ResponseError("You can't see these settings")
		return
	}
	titles, err := object.GetBadgeTitles(user.Id)
	if c.responseErr(err) {
		return
	}
	groups, err := object.GetUserGroups(viewer, user.Id)
	if c.responseErr(err) {
		return
	}
	c.ResponseOk(map[string]interface{}{
		"emailLevel":         object.EmailLevelName(user.EmailLevel, false),
		"emailMessagesLevel": object.EmailLevelName(user.EmailMessagesLevel, true),
		"emailEnabled":       object.IsEmailConfigured(),
		"titles":             titles,
		"groups":             groups,
	})
}
