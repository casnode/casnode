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

type newPost struct {
	TopicId           int64  `json:"topicId"`
	Raw               string `json:"raw"`
	ReplyToPostNumber int    `json:"replyToPostNumber"`
}

// AddPost
// @Title AddPost
// @Description replies to a topic, or to one of its posts
// @router /add-post [post]
func (c *ApiController) AddPost() {
	user := c.RequireSignedInUser()
	if user == nil {
		return
	}
	input := &newPost{}
	if !c.readJsonBody(input) {
		return
	}
	post, err := object.AddPost(user, input.TopicId, input.Raw, input.ReplyToPostNumber)
	if c.responseErr(err) {
		return
	}
	post.Raw = ""
	c.ResponseOk(post)
}

// GetPost
// @Title GetPost
// @Description returns a post with its Markdown, for editing or quoting it
// @router /get-post [get]
func (c *ApiController) GetPost() {
	user := c.GetSessionUser()
	post, err := object.GetPost(c.getInt64("id"))
	if c.responseErr(err) {
		return
	}
	if post == nil {
		c.ResponseError("The post does not exist")
		return
	}
	topic := c.loadVisibleTopic(user, post.TopicId)
	if topic == nil {
		return
	}
	if post.Deleted && !user.IsStaff() {
		c.ResponseError("The post does not exist")
		return
	}
	err = object.FillPostUserState(user, topic, []*object.Post{post})
	if c.responseErr(err) {
		return
	}
	author, err := object.GetUser(post.UserId)
	if c.responseErr(err) {
		return
	}
	c.ResponseOk(post, author.Basic())
}

// UpdatePost
// @Title UpdatePost
// @Description edits a post, keeping the previous version as a revision
// @router /update-post [post]
func (c *ApiController) UpdatePost() {
	user := c.RequireSignedInUser()
	if user == nil {
		return
	}
	input := &object.PostUpdate{}
	if !c.readJsonBody(input) {
		return
	}
	post, err := object.UpdatePost(user, input)
	if c.responseErr(err) {
		return
	}
	c.ResponseOk(post)
}

// DeletePost
// @Title DeletePost
// @router /delete-post [post]
func (c *ApiController) DeletePost() {
	user := c.RequireSignedInUser()
	if user == nil {
		return
	}
	if c.responseErr(object.DeletePost(user, c.getInt64("id"))) {
		return
	}
	c.ResponseOk()
}

// RecoverPost
// @Title RecoverPost
// @router /recover-post [post]
func (c *ApiController) RecoverPost() {
	user := c.RequireStaff()
	if user == nil {
		return
	}
	if c.responseErr(object.RecoverPost(user, c.getInt64("id"))) {
		return
	}
	c.ResponseOk()
}

// GetPostReplies
// @Title GetPostReplies
// @Description returns the direct replies to a post
// @router /get-post-replies [get]
func (c *ApiController) GetPostReplies() {
	user := c.GetSessionUser()
	post, err := object.GetPost(c.getInt64("id"))
	if c.responseErr(err) {
		return
	}
	if post == nil {
		c.ResponseError("The post does not exist")
		return
	}
	topic := c.loadVisibleTopic(user, post.TopicId)
	if topic == nil {
		return
	}
	replies, err := object.GetPostReplies(user, topic, post)
	if c.responseErr(err) {
		return
	}
	userIds := []int64{}
	for _, reply := range replies {
		userIds = append(userIds, reply.UserId)
	}
	users, err := object.GetUserMap(userIds)
	if c.responseErr(err) {
		return
	}
	c.ResponseOk(replies, users)
}

// GetPostRevisions
// @Title GetPostRevisions
// @Description returns the edit history of a post
// @router /get-post-revisions [get]
func (c *ApiController) GetPostRevisions() {
	user := c.GetSessionUser()
	post, err := object.GetPost(c.getInt64("id"))
	if c.responseErr(err) {
		return
	}
	if post == nil {
		c.ResponseError("The post does not exist")
		return
	}
	if c.loadVisibleTopic(user, post.TopicId) == nil {
		return
	}
	revisions, err := object.GetPostRevisions(post.Id)
	if c.responseErr(err) {
		return
	}
	userIds := []int64{}
	for _, revision := range revisions {
		userIds = append(userIds, revision.UserId)
	}
	users, err := object.GetUserMap(userIds)
	if c.responseErr(err) {
		return
	}
	c.ResponseOk(revisions, users)
}

// GetPostLikers
// @Title GetPostLikers
// @router /get-post-likers [get]
func (c *ApiController) GetPostLikers() {
	user := c.GetSessionUser()
	post, err := object.GetPost(c.getInt64("id"))
	if c.responseErr(err) {
		return
	}
	if post == nil || c.loadVisibleTopic(user, post.TopicId) == nil {
		if post == nil {
			c.ResponseError("The post does not exist")
		}
		return
	}
	likers, err := object.GetPostLikers(post.Id)
	if c.responseErr(err) {
		return
	}
	c.ResponseOk(likers)
}

// LikePost
// @Title LikePost
// @Description likes a post, or takes the like back with undo=true
// @router /like-post [post]
func (c *ApiController) LikePost() {
	user := c.RequireSignedInUser()
	if user == nil {
		return
	}
	post, err := object.LikePost(user, c.getInt64("id"), c.GetString("undo") != "true")
	if c.responseErr(err) {
		return
	}
	c.ResponseOk(map[string]interface{}{"likeCount": post.LikeCount, "liked": post.Liked})
}

// BookmarkPost
// @Title BookmarkPost
// @router /bookmark-post [post]
func (c *ApiController) BookmarkPost() {
	user := c.RequireSignedInUser()
	if user == nil {
		return
	}
	if c.responseErr(object.BookmarkPost(user, c.getInt64("id"), c.GetString("undo") != "true")) {
		return
	}
	c.ResponseOk()
}

type flagInput struct {
	Id         int64  `json:"id"`
	ActionType int    `json:"actionType"`
	Message    string `json:"message"`
}

// FlagPost
// @Title FlagPost
// @Description flags a post for staff: 3 off-topic, 4 inappropriate, 7 something else, 8 spam
// @router /flag-post [post]
func (c *ApiController) FlagPost() {
	user := c.RequireSignedInUser()
	if user == nil {
		return
	}
	input := &flagInput{}
	if !c.readJsonBody(input) {
		return
	}
	if c.responseErr(object.FlagPost(user, input.Id, input.ActionType, input.Message)) {
		return
	}
	c.ResponseOk()
}

type previewInput struct {
	Raw string `json:"raw"`
}

// Preview
// @Title Preview
// @Description renders Markdown the way a post will look
// @router /preview [post]
func (c *ApiController) Preview() {
	if c.RequireSignedInUser() == nil {
		return
	}
	input := &previewInput{}
	if !c.readJsonBody(input) {
		return
	}
	c.ResponseOk(object.PreviewMarkdown(input.Raw))
}
