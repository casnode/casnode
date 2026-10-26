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

// GetChatChannels
// @Title GetChatChannels
// @Description the channels the user follows and their direct messages, with unread counts
// @router /get-chat-channels [get]
func (c *ApiController) GetChatChannels() {
	user := c.RequireSignedInUser()
	if user == nil {
		return
	}
	list, err := object.GetMyChatChannels(user)
	if c.responseErr(err) {
		return
	}
	c.ResponseOk(list)
}

// BrowseChatChannels
// @Title BrowseChatChannels
// @Description the category channels the user may see: status open|read_only|closed|archived|all
// @router /browse-chat-channels [get]
func (c *ApiController) BrowseChatChannels() {
	if !object.IsChatEnabled() {
		c.ResponseError("Chat is turned off")
		return
	}
	channels, err := object.BrowseChatChannels(c.GetSessionUser(), c.GetString("status"), c.GetString("term"))
	if c.responseErr(err) {
		return
	}
	c.ResponseOk(channels)
}

// GetChatChannel
// @Title GetChatChannel
// @router /get-chat-channel [get]
func (c *ApiController) GetChatChannel() {
	channel, err := object.GetChatChannelForUser(c.GetSessionUser(), c.getInt64("id"))
	if c.responseErr(err) {
		return
	}
	c.ResponseOk(channel)
}

// GetChatMessages
// @Title GetChatMessages
// @Description messages of a channel: the latest, or before / after / around a message id
// @router /get-chat-messages [get]
func (c *ApiController) GetChatMessages() {
	list, err := object.GetChatMessages(c.GetSessionUser(), c.getInt64("channelId"), c.getInt64("before"), c.getInt64("after"), c.getInt64("around"))
	if c.responseErr(err) {
		return
	}
	c.ResponseOk(list)
}

type chatMessageInput struct {
	Id          int64  `json:"id"`
	ChannelId   int64  `json:"channelId"`
	Raw         string `json:"raw"`
	InReplyToId int64  `json:"inReplyToId"`
}

// SendChatMessage
// @Title SendChatMessage
// @router /send-chat-message [post]
func (c *ApiController) SendChatMessage() {
	user := c.RequireSignedInUser()
	if user == nil {
		return
	}
	input := &chatMessageInput{}
	if !c.readJsonBody(input) {
		return
	}
	message, err := object.SendChatMessage(user, input.ChannelId, input.Raw, input.InReplyToId)
	if c.responseErr(err) {
		return
	}
	c.ResponseOk(message)
}

// EditChatMessage
// @Title EditChatMessage
// @router /edit-chat-message [post]
func (c *ApiController) EditChatMessage() {
	user := c.RequireSignedInUser()
	if user == nil {
		return
	}
	input := &chatMessageInput{}
	if !c.readJsonBody(input) {
		return
	}
	message, err := object.EditChatMessage(user, input.Id, input.Raw)
	if c.responseErr(err) {
		return
	}
	c.ResponseOk(message)
}

// GetChatMessage
// @Title GetChatMessage
// @Description a message with its Markdown, for editing or quoting it
// @router /get-chat-message [get]
func (c *ApiController) GetChatMessage() {
	user := c.RequireSignedInUser()
	if user == nil {
		return
	}
	message, err := object.GetChatMessageRaw(user, c.getInt64("id"))
	if c.responseErr(err) {
		return
	}
	c.ResponseOk(message)
}

// DeleteChatMessage
// @Title DeleteChatMessage
// @Description deletes a message, or restores it with restore=true (staff)
// @router /delete-chat-message [post]
func (c *ApiController) DeleteChatMessage() {
	user := c.RequireSignedInUser()
	if user == nil {
		return
	}
	if c.responseErr(object.SetChatMessageDeleted(user, c.getInt64("id"), c.GetString("restore") != "true")) {
		return
	}
	c.ResponseOk()
}

// ReactChatMessage
// @Title ReactChatMessage
// @Description adds an emoji reaction to a message, or takes it back with undo=true
// @router /react-chat-message [post]
func (c *ApiController) ReactChatMessage() {
	user := c.RequireSignedInUser()
	if user == nil {
		return
	}
	reactions, err := object.ReactChatMessage(user, c.getInt64("id"), c.GetString("emoji"), c.GetString("undo") != "true")
	if c.responseErr(err) {
		return
	}
	c.ResponseOk(reactions)
}

// FollowChatChannel
// @Title FollowChatChannel
// @Description joins a channel, or leaves it with follow=false
// @router /follow-chat-channel [post]
func (c *ApiController) FollowChatChannel() {
	user := c.RequireSignedInUser()
	if user == nil {
		return
	}
	if c.responseErr(object.FollowChatChannel(user, c.getInt64("id"), c.GetString("follow") != "false")) {
		return
	}
	c.ResponseOk()
}

// MarkChatRead
// @Title MarkChatRead
// @router /mark-chat-read [post]
func (c *ApiController) MarkChatRead() {
	user := c.RequireSignedInUser()
	if user == nil {
		return
	}
	if c.responseErr(object.MarkChatRead(user, c.getInt64("channelId"), c.getInt64("messageId"))) {
		return
	}
	c.ResponseOk()
}

// CreateChatChannel
// @Title CreateChatChannel
// @router /create-chat-channel [post]
func (c *ApiController) CreateChatChannel() {
	user := c.RequireStaff()
	if user == nil {
		return
	}
	input := &object.ChatChannelInput{}
	if !c.readJsonBody(input) {
		return
	}
	channel, err := object.CreateChatChannel(user, input)
	if c.responseErr(err) {
		return
	}
	c.ResponseOk(channel)
}

// UpdateChatChannel
// @Title UpdateChatChannel
// @router /update-chat-channel [post]
func (c *ApiController) UpdateChatChannel() {
	user := c.RequireStaff()
	if user == nil {
		return
	}
	input := &object.ChatChannelInput{}
	if !c.readJsonBody(input) {
		return
	}
	channel, err := object.UpdateChatChannel(user, input)
	if c.responseErr(err) {
		return
	}
	c.ResponseOk(channel)
}

// DeleteChatChannel
// @Title DeleteChatChannel
// @router /delete-chat-channel [post]
func (c *ApiController) DeleteChatChannel() {
	user := c.RequireAdmin()
	if user == nil {
		return
	}
	if c.responseErr(object.DeleteChatChannel(user, c.getInt64("id"))) {
		return
	}
	c.ResponseOk()
}

type directChannelInput struct {
	Usernames []string `json:"usernames"`
}

// CreateDirectChannel
// @Title CreateDirectChannel
// @Description the direct message channel with these users, made when missing
// @router /create-direct-channel [post]
func (c *ApiController) CreateDirectChannel() {
	user := c.RequireSignedInUser()
	if user == nil {
		return
	}
	input := &directChannelInput{}
	if !c.readJsonBody(input) {
		return
	}
	channel, err := object.GetOrCreateDirectChannel(user, input.Usernames)
	if c.responseErr(err) {
		return
	}
	c.ResponseOk(channel)
}
