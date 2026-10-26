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
	"strings"

	"github.com/casnode/casnode/object"
)

func (c *ApiController) loadGroup(user *object.User, name string) *object.Group {
	group, err := object.GetGroupForUser(user, name)
	if c.responseErr(err) {
		return nil
	}
	return group
}

// GetGroups
// @Title GetGroups
// @Description the groups the user may see, with whether they are in them
// @router /get-groups [get]
func (c *ApiController) GetGroups() {
	groups, err := object.GetGroups(c.GetSessionUser(), c.GetString("term"))
	if c.responseErr(err) {
		return
	}
	c.ResponseOk(groups)
}

// GetGroup
// @Title GetGroup
// @router /get-group [get]
func (c *ApiController) GetGroup() {
	group := c.loadGroup(c.GetSessionUser(), c.GetString("name"))
	if group == nil {
		return
	}
	c.ResponseOk(group)
}

// GetGroupMembers
// @Title GetGroupMembers
// @router /get-group-members [get]
func (c *ApiController) GetGroupMembers() {
	group := c.loadGroup(c.GetSessionUser(), c.GetString("name"))
	if group == nil {
		return
	}
	members, more, err := object.GetGroupMembers(group, c.GetString("term"), c.getInt("page"))
	if c.responseErr(err) {
		return
	}
	c.ResponseOk(members, more)
}

// GetGroupActivity
// @Title GetGroupActivity
// @Description the latest posts of the group's members
// @router /get-group-activity [get]
func (c *ApiController) GetGroupActivity() {
	user := c.GetSessionUser()
	group := c.loadGroup(user, c.GetString("name"))
	if group == nil {
		return
	}
	posts, more, err := object.GetGroupPosts(user, group, c.getInt("page"))
	if c.responseErr(err) {
		return
	}
	userIds := []int64{}
	for _, item := range posts {
		userIds = append(userIds, item.Post.UserId)
	}
	users, err := object.GetUserMap(userIds)
	if c.responseErr(err) {
		return
	}
	c.ResponseOk(map[string]interface{}{"posts": posts, "users": users, "more": more})
}

// GetGroupMessages
// @Title GetGroupMessages
// @Description the messages sent to a group, for its members
// @router /get-group-messages [get]
func (c *ApiController) GetGroupMessages() {
	user := c.RequireSignedInUser()
	if user == nil {
		return
	}
	group := c.loadGroup(user, c.GetString("name"))
	if group == nil {
		return
	}
	if !group.IsMember && !user.Admin {
		c.ResponseError("Only the members can read the group's messages")
		return
	}
	list, err := object.GetTopicList(user, &object.TopicListQuery{Filter: "private_messages_group", GroupId: group.Id, Page: c.getInt("page")})
	if c.responseErr(err) {
		return
	}
	c.ResponseOk(list)
}

// AddGroup
// @Title AddGroup
// @router /add-group [post]
func (c *ApiController) AddGroup() {
	if c.RequireAdmin() == nil {
		return
	}
	group := &object.Group{}
	if !c.readJsonBody(group) {
		return
	}
	if c.responseErr(object.AddGroup(group)) {
		return
	}
	c.ResponseOk(group)
}

// UpdateGroup
// @Title UpdateGroup
// @Description changes a group, for admins and the group's owners
// @router /update-group [post]
func (c *ApiController) UpdateGroup() {
	user := c.RequireSignedInUser()
	if user == nil {
		return
	}
	group := &object.Group{}
	if !c.readJsonBody(group) {
		return
	}
	if c.responseErr(object.UpdateGroup(user, group)) {
		return
	}
	c.ResponseOk(group)
}

// DeleteGroup
// @Title DeleteGroup
// @router /delete-group [post]
func (c *ApiController) DeleteGroup() {
	if c.RequireAdmin() == nil {
		return
	}
	if c.responseErr(object.DeleteGroup(c.getInt64("id"))) {
		return
	}
	c.ResponseOk()
}

type groupMembersInput struct {
	Name      string   `json:"name"`
	Usernames []string `json:"usernames"`
	Owner     bool     `json:"owner"`
}

// AddGroupMembers
// @Title AddGroupMembers
// @router /add-group-members [post]
func (c *ApiController) AddGroupMembers() {
	user := c.RequireSignedInUser()
	if user == nil {
		return
	}
	input := &groupMembersInput{}
	if !c.readJsonBody(input) {
		return
	}
	group := c.loadGroup(user, input.Name)
	if group == nil {
		return
	}
	usernames := []string{}
	for _, name := range input.Usernames {
		for _, part := range strings.FieldsFunc(name, func(r rune) bool { return r == ',' || r == ' ' || r == '\n' }) {
			usernames = append(usernames, strings.TrimPrefix(part, "@"))
		}
	}
	if c.responseErr(object.AddGroupMembers(user, group, usernames, input.Owner)) {
		return
	}
	c.ResponseOk()
}

// RemoveGroupMember
// @Title RemoveGroupMember
// @Description removes a member, or the user themselves (leaving the group)
// @router /remove-group-member [post]
func (c *ApiController) RemoveGroupMember() {
	user := c.RequireSignedInUser()
	if user == nil {
		return
	}
	group := c.loadGroup(user, c.GetString("name"))
	if group == nil {
		return
	}
	userId := c.getInt64("userId")
	if userId == 0 {
		userId = user.Id
	}
	if c.responseErr(object.RemoveGroupMember(user, group, userId)) {
		return
	}
	c.ResponseOk()
}

// SetGroupOwner
// @Title SetGroupOwner
// @router /set-group-owner [post]
func (c *ApiController) SetGroupOwner() {
	user := c.RequireSignedInUser()
	if user == nil {
		return
	}
	group := c.loadGroup(user, c.GetString("name"))
	if group == nil {
		return
	}
	if c.responseErr(object.SetGroupOwner(user, group, c.getInt64("userId"), c.GetString("owner") == "true")) {
		return
	}
	c.ResponseOk()
}

// JoinGroup
// @Title JoinGroup
// @router /join-group [post]
func (c *ApiController) JoinGroup() {
	user := c.RequireSignedInUser()
	if user == nil {
		return
	}
	group := c.loadGroup(user, c.GetString("name"))
	if group == nil {
		return
	}
	if c.responseErr(object.JoinGroup(user, group)) {
		return
	}
	c.ResponseOk()
}

type groupRequestInput struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

// RequestGroupMembership
// @Title RequestGroupMembership
// @router /request-group-membership [post]
func (c *ApiController) RequestGroupMembership() {
	user := c.RequireSignedInUser()
	if user == nil {
		return
	}
	input := &groupRequestInput{}
	if !c.readJsonBody(input) {
		return
	}
	group := c.loadGroup(user, input.Name)
	if group == nil {
		return
	}
	if c.responseErr(object.RequestGroupMembership(user, group, input.Reason)) {
		return
	}
	c.ResponseOk()
}

// GetGroupRequests
// @Title GetGroupRequests
// @router /get-group-requests [get]
func (c *ApiController) GetGroupRequests() {
	user := c.RequireSignedInUser()
	if user == nil {
		return
	}
	group := c.loadGroup(user, c.GetString("name"))
	if group == nil {
		return
	}
	requests, err := object.GetGroupRequests(user, group)
	if c.responseErr(err) {
		return
	}
	c.ResponseOk(requests)
}

// HandleGroupRequest
// @Title HandleGroupRequest
// @Description accepts or denies a membership request
// @router /handle-group-request [post]
func (c *ApiController) HandleGroupRequest() {
	user := c.RequireSignedInUser()
	if user == nil {
		return
	}
	group := c.loadGroup(user, c.GetString("name"))
	if group == nil {
		return
	}
	if c.responseErr(object.HandleGroupRequest(user, group, c.getInt64("userId"), c.GetString("accept") == "true")) {
		return
	}
	c.ResponseOk()
}
