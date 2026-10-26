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

import (
	"fmt"
	"time"
)

// The action types follow Discourse's numbering.
const (
	PostActionBookmark      = 1
	PostActionLike          = 2
	PostActionOffTopic      = 3
	PostActionInappropriate = 4
	PostActionNotifyMods    = 7
	PostActionSpam          = 8
)

type PostAction struct {
	Id         int64     `xorm:"pk autoincr" json:"id"`
	PostId     int64     `xorm:"index unique(post_user_type)" json:"postId"`
	UserId     int64     `xorm:"index unique(post_user_type)" json:"userId"`
	ActionType int       `xorm:"unique(post_user_type)" json:"actionType"`
	Message    string    `xorm:"varchar(1000)" json:"message,omitempty"`
	Deleted    bool      `json:"deleted"`
	CreatedAt  time.Time `xorm:"created" json:"createdAt"`
}

func IsFlagType(actionType int) bool {
	return actionType == PostActionOffTopic || actionType == PostActionInappropriate || actionType == PostActionNotifyMods || actionType == PostActionSpam
}

func getPostAction(postId int64, userId int64, actionType int) (*PostAction, error) {
	action := &PostAction{}
	has, err := adapter.Engine.Where("post_id = ? AND user_id = ? AND action_type = ?", postId, userId, actionType).Get(action)
	if err != nil || !has {
		return nil, err
	}
	return action, nil
}

// setPostAction adds or removes the user's action on the post and tells whether anything changed.
func setPostAction(postId int64, userId int64, actionType int, enabled bool, message string) (bool, error) {
	action, err := getPostAction(postId, userId, actionType)
	if err != nil {
		return false, err
	}
	if action == nil {
		if !enabled {
			return false, nil
		}
		_, err = adapter.Engine.Insert(&PostAction{PostId: postId, UserId: userId, ActionType: actionType, Message: message})
		return err == nil, err
	}
	if action.Deleted == !enabled {
		return false, nil
	}
	action.Deleted = !enabled
	action.Message = message
	_, err = adapter.Engine.ID(action.Id).Cols("deleted", "message").Update(action)
	return err == nil, err
}

func getActionablePost(user *User, postId int64) (*Post, *Topic, error) {
	if user == nil {
		return nil, nil, fmt.Errorf("Please sign in first")
	}
	post, err := GetPost(postId)
	if err != nil {
		return nil, nil, err
	}
	if post == nil || post.Deleted {
		return nil, nil, fmt.Errorf("The post does not exist")
	}
	topic, err := GetTopic(post.TopicId)
	if err != nil {
		return nil, nil, err
	}
	canSee, err := CanSeeTopic(user, topic)
	if err != nil {
		return nil, nil, err
	}
	if !canSee {
		return nil, nil, fmt.Errorf("The post does not exist")
	}
	return post, topic, nil
}

func LikePost(user *User, postId int64, enabled bool) (*Post, error) {
	post, topic, err := getActionablePost(user, postId)
	if err != nil {
		return nil, err
	}
	if post.UserId == user.Id {
		return nil, fmt.Errorf("You can't like your own post")
	}
	if user.IsSuspended() {
		return nil, fmt.Errorf("Your account is suspended")
	}

	changed, err := setPostAction(post.Id, user.Id, PostActionLike, enabled, "")
	if err != nil || !changed {
		return post, err
	}

	delta := 1
	if !enabled {
		delta = -1
	}
	post.LikeCount += delta
	_, err = adapter.Engine.ID(post.Id).Incr("like_count", delta).Update(&Post{})
	if err != nil {
		return nil, err
	}
	_, err = adapter.Engine.ID(topic.Id).Incr("like_count", delta).Update(&Topic{})
	if err != nil {
		return nil, err
	}
	incrUser(user.Id, "likes_given", delta)
	incrUser(post.UserId, "likes_received", delta)
	PublishToTopic(topic, map[string]interface{}{"type": "liked", "postId": post.Id, "postNumber": post.PostNumber, "likeCount": post.LikeCount})
	TriggerBadgeGrant()
	if enabled {
		notifyLike(user, topic, post)
	}
	post.Liked = enabled
	return post, nil
}

func GetPostLikers(postId int64) ([]*User, error) {
	actions := []*PostAction{}
	err := adapter.Engine.Where("post_id = ? AND action_type = ? AND deleted = ?", postId, PostActionLike, false).Asc("created_at").Find(&actions)
	if err != nil {
		return nil, err
	}
	ids := []int64{}
	for _, action := range actions {
		ids = append(ids, action.UserId)
	}
	userMap, err := GetUserMap(ids)
	if err != nil {
		return nil, err
	}
	res := []*User{}
	for _, id := range ids {
		if user := userMap[id]; user != nil {
			res = append(res, user)
		}
	}
	return res, nil
}

func BookmarkPost(user *User, postId int64, enabled bool) error {
	post, _, err := getActionablePost(user, postId)
	if err != nil {
		return err
	}
	_, err = setPostAction(post.Id, user.Id, PostActionBookmark, enabled, "")
	return err
}

func GetBookmarks(user *User, page int) ([]*UserPost, bool, error) {
	const pageSize = 30
	posts := []*Post{}
	err := adapter.Engine.Table("post").Select("post.*").
		Join("INNER", "post_action", "post_action.post_id = post.id").
		Join("INNER", "topic", "topic.id = post.topic_id").
		Where("post_action.user_id = ? AND post_action.action_type = ? AND post_action.deleted = ? AND post.deleted = ? AND topic.deleted = ?",
			user.Id, PostActionBookmark, false, false, false).
		Desc("post_action.created_at").Limit(pageSize+1, page*pageSize).Find(&posts)
	if err != nil {
		return nil, false, err
	}
	more := len(posts) > pageSize
	if more {
		posts = posts[:pageSize]
	}
	res, err := attachTopics(posts)
	return res, more, err
}

func GetUserLikes(viewer *User, userId int64, page int) ([]*UserPost, bool, error) {
	const pageSize = 30
	posts := []*Post{}
	session := adapter.Engine.Table("post").Select("post.*").
		Join("INNER", "post_action", "post_action.post_id = post.id").
		Join("INNER", "topic", "topic.id = post.topic_id").
		Where("post_action.user_id = ? AND post_action.action_type = ? AND post_action.deleted = ? AND post.deleted = ? AND topic.deleted = ? AND topic.archetype = ?",
			userId, PostActionLike, false, false, false, ArchetypeRegular)
	hidden := GetHiddenCategoryIds(viewer)
	if len(hidden) > 0 {
		session = session.NotIn("topic.category_id", hidden)
	}
	err := session.Desc("post_action.created_at").Limit(pageSize+1, page*pageSize).Find(&posts)
	if err != nil {
		return nil, false, err
	}
	more := len(posts) > pageSize
	if more {
		posts = posts[:pageSize]
	}
	res, err := attachTopics(posts)
	return res, more, err
}

// FlagPost records the flag and puts the post in the review queue.
func FlagPost(user *User, postId int64, actionType int, message string) error {
	if !IsFlagType(actionType) {
		return fmt.Errorf("Unknown flag type")
	}
	post, topic, err := getActionablePost(user, postId)
	if err != nil {
		return err
	}
	if post.UserId == user.Id {
		return fmt.Errorf("You can't flag your own post")
	}
	changed, err := setPostAction(post.Id, user.Id, actionType, true, message)
	if err != nil {
		return err
	}
	if !changed {
		return fmt.Errorf("You have already flagged this post")
	}
	return addFlagToReview(user, topic, post, actionType, message)
}
