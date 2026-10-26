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
	"sort"
	"strconv"
	"strings"

	"github.com/casnode/casnode/object"
)

// GetTopics
// @Title GetTopics
// @Description lists topics: filter latest|new|unread|top|hot|posted|created|private_messages|private_messages_sent, in a category or with a tag
// @router /get-topics [get]
func (c *ApiController) GetTopics() {
	user := c.GetSessionUser()
	query := &object.TopicListQuery{
		Filter:     c.GetString("filter", "latest"),
		CategoryId: c.getInt64("categoryId"),
		NoSubs:     c.GetString("noSubs") == "true",
		Tag:        c.GetString("tag"),
		Period:     c.GetString("period", "all"),
		Order:      c.GetString("order"),
		Ascending:  c.GetString("ascending") == "true",
		Page:       c.getInt("page"),
	}
	if username := c.GetString("username"); username != "" {
		target, err := object.GetUserByUsername(username)
		if c.responseErr(err) {
			return
		}
		if target == nil {
			c.ResponseError("The user does not exist")
			return
		}
		if strings.HasPrefix(query.Filter, "private_messages") && (user == nil || (user.Id != target.Id && !user.Admin)) {
			c.ResponseError("You can't see these messages")
			return
		}
		query.UserId = target.Id
	}
	if query.Page < 0 {
		query.Page = 0
	}

	list, err := object.GetTopicList(user, query)
	if c.responseErr(err) {
		return
	}
	c.ResponseOk(list)
}

type participant struct {
	UserId    int64 `json:"userId"`
	PostCount int   `json:"postCount"`
}

type topicDetails struct {
	CreatedBy      int64          `json:"createdBy"`
	LastPoster     int64          `json:"lastPoster"`
	Participants   []*participant `json:"participants"`
	AllowedUserIds []int64        `json:"allowedUserIds"`
	CanEdit        bool           `json:"canEdit"`
	CanReply       bool           `json:"canReply"`
	CanModerate    bool           `json:"canModerate"`
	CanDelete      bool           `json:"canDelete"`
	CanRecover     bool           `json:"canRecover"`
	CanClose       bool           `json:"canClose"`
	Bookmarked     bool           `json:"bookmarked"`
}

type topicView struct {
	Topic          *object.Topic          `json:"topic"`
	Category       *object.Category       `json:"category"`
	Posts          []*object.Post         `json:"posts"`
	Stream         []int                  `json:"stream"`
	Users          map[int64]*object.User `json:"users"`
	Details        *topicDetails          `json:"details"`
	Suggested      []*object.Topic        `json:"suggested"`
	SuggestedUsers map[int64]*object.User `json:"suggestedUsers"`
	ReadingTime    int                    `json:"readingTime"`
}

func (c *ApiController) loadVisibleTopic(user *object.User, id int64) *object.Topic {
	topic, err := object.GetTopic(id)
	if c.responseErr(err) {
		return nil
	}
	canSee, err := object.CanSeeTopic(user, topic)
	if c.responseErr(err) {
		return nil
	}
	if !canSee {
		c.ResponseError("The topic does not exist or is private")
		return nil
	}
	return topic
}

// GetTopic
// @Title GetTopic
// @Description returns a topic with a window of its posts: around postNumber, or page (20 posts each) for crawlers
// @router /get-topic [get]
func (c *ApiController) GetTopic() {
	user := c.GetSessionUser()
	topic := c.loadVisibleTopic(user, c.getInt64("id"))
	if topic == nil {
		return
	}

	stream, err := object.GetPostStream(user, topic.Id)
	if c.responseErr(err) {
		return
	}

	// the window: a page for crawlers, else from a few posts before the one asked for
	start := 0
	if page := c.getInt("page"); page > 1 {
		start = (page - 1) * object.PostChunkSize
	} else if postNumber := c.getInt("postNumber"); postNumber > 1 {
		index := sort.SearchInts(stream, postNumber)
		start = index - 5
		if index >= len(stream)-object.PostChunkSize+5 {
			start = len(stream) - object.PostChunkSize
		}
	}
	if start > len(stream)-1 {
		start = len(stream) - 1
	}
	if start < 0 {
		start = 0
	}
	end := start + object.PostChunkSize
	if end > len(stream) {
		end = len(stream)
	}

	posts, err := object.GetPostsByNumbers(user, topic, stream[start:end])
	if c.responseErr(err) {
		return
	}

	if c.GetString("track") != "false" {
		object.RecordTopicView(user, topic, c.visitorKey(user))
	}
	err = object.FillTopicUserState(user, []*object.Topic{topic})
	if c.responseErr(err) {
		return
	}

	view := &topicView{Topic: topic, Posts: posts, Stream: stream}
	if !topic.IsPrivateMessage() {
		view.Category, err = object.GetCategory(topic.CategoryId)
		if c.responseErr(err) {
			return
		}
	}

	details, userIds, err := getTopicDetails(user, topic)
	if c.responseErr(err) {
		return
	}
	view.Details = details
	for _, post := range posts {
		userIds = append(userIds, post.UserId, post.ReplyToUserId, post.LastEditorId)
	}
	view.Users, err = object.GetUserMap(userIds)
	if c.responseErr(err) {
		return
	}

	view.Suggested, view.SuggestedUsers, err = object.GetSuggestedTopics(user, topic)
	if c.responseErr(err) {
		return
	}
	view.ReadingTime = (topic.WordCount + 499) / 500
	if view.ReadingTime < 1 {
		view.ReadingTime = 1
	}
	c.ResponseOk(view)
}

func getTopicDetails(user *object.User, topic *object.Topic) (*topicDetails, []int64, error) {
	details := &topicDetails{
		CreatedBy:   topic.UserId,
		LastPoster:  topic.LastPostUserId,
		CanEdit:     object.CanEditTopic(user, topic),
		CanReply:    object.CanReply(user, topic),
		CanModerate: user.IsStaff(),
		CanDelete:   user.IsStaff() || (user != nil && topic.UserId == user.Id && topic.PostsCount <= 1),
		CanRecover:  user.IsStaff() && topic.Deleted,
		CanClose:    user.IsStaff() || (user != nil && topic.UserId == user.Id),
	}
	participants, err := object.GetTopicParticipants(topic.Id, 24)
	if err != nil {
		return nil, nil, err
	}
	userIds := []int64{topic.UserId, topic.LastPostUserId}
	for _, p := range participants {
		details.Participants = append(details.Participants, &participant{UserId: p.UserId, PostCount: p.Count})
		userIds = append(userIds, p.UserId)
	}
	if topic.IsPrivateMessage() {
		details.AllowedUserIds, err = object.GetTopicAllowedUserIds(topic.Id)
		if err != nil {
			return nil, nil, err
		}
		userIds = append(userIds, details.AllowedUserIds...)
	}
	return details, userIds, nil
}

// GetPosts
// @Title GetPosts
// @Description returns posts of a topic by their numbers (comma-separated), for loading more while scrolling
// @router /get-posts [get]
func (c *ApiController) GetPosts() {
	user := c.GetSessionUser()
	topic := c.loadVisibleTopic(user, c.getInt64("topicId"))
	if topic == nil {
		return
	}
	numbers := []int{}
	for _, item := range strings.Split(c.GetString("postNumbers"), ",") {
		n, err := strconv.Atoi(strings.TrimSpace(item))
		if err == nil && len(numbers) < 100 {
			numbers = append(numbers, n)
		}
	}
	posts, err := object.GetPostsByNumbers(user, topic, numbers)
	if c.responseErr(err) {
		return
	}
	userIds := []int64{}
	for _, post := range posts {
		userIds = append(userIds, post.UserId, post.ReplyToUserId)
	}
	users, err := object.GetUserMap(userIds)
	if c.responseErr(err) {
		return
	}
	c.ResponseOk(posts, users)
}

// AddTopic
// @Title AddTopic
// @Description creates a topic or a personal message with its first post
// @router /add-topic [post]
func (c *ApiController) AddTopic() {
	user := c.RequireSignedInUser()
	if user == nil {
		return
	}
	input := &object.NewTopic{}
	if !c.readJsonBody(input) {
		return
	}
	topic, err := object.CreateTopic(user, input)
	if c.responseErr(err) {
		return
	}
	c.ResponseOk(topic)
}

// UpdateTopic
// @Title UpdateTopic
// @Description changes a topic's title, category and tags
// @router /update-topic [post]
func (c *ApiController) UpdateTopic() {
	user := c.RequireSignedInUser()
	if user == nil {
		return
	}
	input := &object.TopicUpdate{}
	if !c.readJsonBody(input) {
		return
	}
	topic, err := object.UpdateTopic(user, input)
	if c.responseErr(err) {
		return
	}
	c.ResponseOk(topic)
}

// SetTopicStatus
// @Title SetTopicStatus
// @Description closes, archives, unlists or pins a topic (status closed|archived|visible|pinned|pinned_globally)
// @router /set-topic-status [post]
func (c *ApiController) SetTopicStatus() {
	user := c.RequireSignedInUser()
	if user == nil {
		return
	}
	err := object.SetTopicStatus(user, c.getInt64("id"), c.GetString("status"), c.GetString("enabled") == "true")
	if c.responseErr(err) {
		return
	}
	c.ResponseOk()
}

// DeleteTopic
// @Title DeleteTopic
// @router /delete-topic [post]
func (c *ApiController) DeleteTopic() {
	user := c.RequireSignedInUser()
	if user == nil {
		return
	}
	if c.responseErr(object.DeleteTopic(user, c.getInt64("id"))) {
		return
	}
	c.ResponseOk()
}

// RecoverTopic
// @Title RecoverTopic
// @router /recover-topic [post]
func (c *ApiController) RecoverTopic() {
	user := c.RequireStaff()
	if user == nil {
		return
	}
	if c.responseErr(object.RecoverTopic(user, c.getInt64("id"))) {
		return
	}
	c.ResponseOk()
}

// SetTopicNotificationLevel
// @Title SetTopicNotificationLevel
// @Description 0 muted, 1 normal, 2 tracking, 3 watching
// @router /set-topic-notification-level [post]
func (c *ApiController) SetTopicNotificationLevel() {
	user := c.RequireSignedInUser()
	if user == nil {
		return
	}
	topic := c.loadVisibleTopic(user, c.getInt64("id"))
	if topic == nil {
		return
	}
	if c.responseErr(object.SetTopicNotificationLevel(user, topic.Id, c.getInt("level"))) {
		return
	}
	c.ResponseOk()
}

// RecordTopicTiming
// @Title RecordTopicTiming
// @Description moves the user's read position in a topic forward
// @router /record-topic-timing [post]
func (c *ApiController) RecordTopicTiming() {
	user := c.RequireSignedInUser()
	if user == nil {
		return
	}
	topic := c.loadVisibleTopic(user, c.getInt64("id"))
	if topic == nil {
		return
	}
	if c.responseErr(object.RecordTopicTiming(user, topic, c.getInt("lastRead"))) {
		return
	}
	c.ResponseOk()
}

// DismissTopics
// @Title DismissTopics
// @Description marks the new or unread topics read
// @router /dismiss-topics [post]
func (c *ApiController) DismissTopics() {
	user := c.RequireSignedInUser()
	if user == nil {
		return
	}
	if c.responseErr(object.DismissTopics(user, c.GetString("filter"), c.getInt64("categoryId"))) {
		return
	}
	c.ResponseOk()
}

// InviteToMessage
// @Title InviteToMessage
// @Description adds a participant to a personal message
// @router /invite-to-message [post]
func (c *ApiController) InviteToMessage() {
	user := c.RequireSignedInUser()
	if user == nil {
		return
	}
	topic := c.loadVisibleTopic(user, c.getInt64("id"))
	if topic == nil {
		return
	}
	if !topic.IsPrivateMessage() {
		c.ResponseError("Only messages have participants")
		return
	}
	if c.responseErr(object.AddTopicAllowedUser(user, topic, c.GetString("username"))) {
		return
	}
	c.ResponseOk()
}
