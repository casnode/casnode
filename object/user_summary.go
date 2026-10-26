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

type UserCount struct {
	UserId int64 `json:"userId"`
	Count  int   `json:"count"`
}

type CategoryCount struct {
	CategoryId int64 `json:"categoryId"`
	TopicCount int   `json:"topicCount"`
	PostCount  int   `json:"postCount"`
}

// UserSummary is the Summary tab of a profile.
type UserSummary struct {
	TopReplies    []*UserPost      `json:"topReplies"`
	TopTopics     []*Topic         `json:"topTopics"`
	MostLikedBy   []*UserCount     `json:"mostLikedBy"`
	MostLiked     []*UserCount     `json:"mostLiked"`
	MostRepliedTo []*UserCount     `json:"mostRepliedTo"`
	TopCategories []*CategoryCount `json:"topCategories"`
	Bookmarks     int64            `json:"bookmarkCount"`
	Users         map[int64]*User  `json:"users"`
}

func GetUserSummary(viewer *User, user *User) (*UserSummary, error) {
	res := &UserSummary{}
	hidden := GetHiddenCategoryIds(viewer)

	replies := []*Post{}
	session := adapter.Engine.Table("post").Select("post.*").Join("INNER", "topic", "topic.id = post.topic_id").
		Where("post.user_id = ? AND post.post_number > ? AND post.deleted = ? AND post.like_count > ? AND topic.deleted = ? AND topic.archetype = ?",
			user.Id, 1, false, 0, false, ArchetypeRegular)
	if len(hidden) > 0 {
		session = session.NotIn("topic.category_id", hidden)
	}
	err := session.Desc("post.like_count").Desc("post.id").Limit(6).Find(&replies)
	if err != nil {
		return nil, err
	}
	res.TopReplies, err = attachTopics(replies)
	if err != nil {
		return nil, err
	}

	topics := []*Topic{}
	topicSession := adapter.Engine.Where("user_id = ? AND deleted = ? AND archetype = ? AND visible = ?", user.Id, false, ArchetypeRegular, true)
	if len(hidden) > 0 {
		topicSession = topicSession.NotIn("category_id", hidden)
	}
	err = topicSession.Desc("like_count").Desc("posts_count").Limit(6).Find(&topics)
	if err != nil {
		return nil, err
	}
	res.TopTopics = topics

	// who liked this user's posts most, and whose posts this user liked most
	res.MostLikedBy = []*UserCount{}
	err = adapter.Engine.Table("post_action").Select("post_action.user_id AS user_id, COUNT(*) AS count").
		Join("INNER", "post", "post.id = post_action.post_id").
		Where("post.user_id = ? AND post_action.action_type = ? AND post_action.deleted = ?", user.Id, PostActionLike, false).
		GroupBy("post_action.user_id").OrderBy("count DESC").Limit(6).Find(&res.MostLikedBy)
	if err != nil {
		return nil, err
	}
	res.MostLiked = []*UserCount{}
	err = adapter.Engine.Table("post_action").Select("post.user_id AS user_id, COUNT(*) AS count").
		Join("INNER", "post", "post.id = post_action.post_id").
		Where("post_action.user_id = ? AND post_action.action_type = ? AND post_action.deleted = ?", user.Id, PostActionLike, false).
		GroupBy("post.user_id").OrderBy("count DESC").Limit(6).Find(&res.MostLiked)
	if err != nil {
		return nil, err
	}
	res.MostRepliedTo = []*UserCount{}
	err = adapter.Engine.Table("post").Select("reply_to_user_id AS user_id, COUNT(*) AS count").
		Where("user_id = ? AND reply_to_user_id > ? AND reply_to_user_id <> ? AND deleted = ?", user.Id, 0, user.Id, false).
		GroupBy("reply_to_user_id").OrderBy("count DESC").Limit(6).Find(&res.MostRepliedTo)
	if err != nil {
		return nil, err
	}

	res.TopCategories = []*CategoryCount{}
	err = adapter.Engine.Table("post").Select("topic.category_id AS category_id, COUNT(DISTINCT topic.id) AS topic_count, COUNT(*) AS post_count").
		Join("INNER", "topic", "topic.id = post.topic_id").
		Where("post.user_id = ? AND post.deleted = ? AND topic.deleted = ? AND topic.archetype = ?", user.Id, false, false, ArchetypeRegular).
		GroupBy("topic.category_id").OrderBy("post_count DESC").Limit(5).Find(&res.TopCategories)
	if err != nil {
		return nil, err
	}
	visible := []*CategoryCount{}
	for _, item := range res.TopCategories {
		if !containsId(hidden, item.CategoryId) {
			visible = append(visible, item)
		}
	}
	res.TopCategories = visible

	if viewer != nil && viewer.Id == user.Id {
		res.Bookmarks, err = adapter.Engine.Where("user_id = ? AND action_type = ? AND deleted = ?", user.Id, PostActionBookmark, false).Count(&PostAction{})
		if err != nil {
			return nil, err
		}
	}

	userIds := []int64{}
	for _, list := range [][]*UserCount{res.MostLikedBy, res.MostLiked, res.MostRepliedTo} {
		for _, item := range list {
			userIds = append(userIds, item.UserId)
		}
	}
	res.Users, err = GetUserMap(userIds)
	return res, err
}
