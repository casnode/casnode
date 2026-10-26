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
	"sync"
	"time"
)

const (
	PostTypeRegular     = 1
	PostTypeSmallAction = 3
)

const PostChunkSize = 20

type Post struct {
	Id                int64     `xorm:"pk autoincr" json:"id"`
	TopicId           int64     `xorm:"index unique(topic_post)" json:"topicId"`
	PostNumber        int       `xorm:"unique(topic_post)" json:"postNumber"`
	UserId            int64     `xorm:"index" json:"userId"`
	Raw               string    `xorm:"mediumtext" json:"raw,omitempty"`
	Cooked            string    `xorm:"mediumtext" json:"cooked"`
	PostType          int       `json:"postType"`
	ActionCode        string    `xorm:"varchar(100)" json:"actionCode,omitempty"`
	ReplyToPostNumber int       `json:"replyToPostNumber"`
	ReplyToUserId     int64     `json:"replyToUserId"`
	ReplyCount        int       `json:"replyCount"`
	LikeCount         int       `json:"likeCount"`
	Version           int       `json:"version"`
	EditReason        string    `xorm:"varchar(500)" json:"editReason,omitempty"`
	LastEditorId      int64     `json:"lastEditorId,omitempty"`
	Deleted           bool      `xorm:"index" json:"deleted"`
	DeletedById       int64     `json:"deletedById,omitempty"`
	Hidden            bool      `json:"hidden"`
	CreatedAt         time.Time `xorm:"created index" json:"createdAt"`
	UpdatedAt         time.Time `xorm:"updated" json:"updatedAt"`

	// what the signed-in user may do with the post
	Liked      bool `xorm:"-" json:"liked"`
	Bookmarked bool `xorm:"-" json:"bookmarked"`
	Flagged    bool `xorm:"-" json:"flagged"`
	CanEdit    bool `xorm:"-" json:"canEdit"`
	CanDelete  bool `xorm:"-" json:"canDelete"`
	CanRecover bool `xorm:"-" json:"canRecover"`
}

// PostRevision keeps what a post looked like before each edit.
type PostRevision struct {
	Id        int64     `xorm:"pk autoincr" json:"id"`
	PostId    int64     `xorm:"index" json:"postId"`
	Number    int       `json:"number"`
	UserId    int64     `json:"userId"`
	RawBefore string    `xorm:"mediumtext" json:"rawBefore"`
	RawAfter  string    `xorm:"mediumtext" json:"rawAfter"`
	Title     string    `xorm:"varchar(300)" json:"title,omitempty"`
	Reason    string    `xorm:"varchar(500)" json:"reason,omitempty"`
	CreatedAt time.Time `xorm:"created" json:"createdAt"`
}

// postLock numbers the posts of a topic one at a time.
var postLock sync.Mutex

func GetPost(id int64) (*Post, error) {
	post := &Post{}
	has, err := adapter.Engine.ID(id).Get(post)
	if err != nil || !has {
		return nil, err
	}
	return post, nil
}

func GetPostByNumber(topicId int64, postNumber int) (*Post, error) {
	post := &Post{}
	has, err := adapter.Engine.Where("topic_id = ? AND post_number = ?", topicId, postNumber).Get(post)
	if err != nil || !has {
		return nil, err
	}
	return post, nil
}

// GetPostStream is the numbers of every post of the topic the user may see, in order.
func GetPostStream(user *User, topicId int64) ([]int, error) {
	numbers := []int{}
	session := adapter.Engine.Table("post").Cols("post_number").Where("topic_id = ?", topicId)
	if !user.IsStaff() {
		session = session.And("deleted = ?", false)
	}
	err := session.Asc("post_number").Find(&numbers)
	return numbers, err
}

func GetPostsByNumbers(user *User, topic *Topic, numbers []int) ([]*Post, error) {
	posts := []*Post{}
	if len(numbers) == 0 {
		return posts, nil
	}
	session := adapter.Engine.Where("topic_id = ?", topic.Id).In("post_number", numbers)
	if !user.IsStaff() {
		session = session.And("deleted = ?", false)
	}
	err := session.Omit("raw").Asc("post_number").Find(&posts)
	if err != nil {
		return nil, err
	}
	err = FillPostUserState(user, topic, posts)
	return posts, err
}

func CanEditPost(user *User, topic *Topic, post *Post) bool {
	if user == nil || post.PostType != PostTypeRegular {
		return false
	}
	if user.IsStaff() {
		return true
	}
	if post.UserId != user.Id || post.Deleted || topic.Archived || user.IsSilenced() || user.IsSuspended() {
		return false
	}
	if limit := GetSiteSettingInt("post_edit_time_limit_minutes"); limit > 0 && time.Since(post.CreatedAt) > time.Duration(limit)*time.Minute {
		return false
	}
	return true
}

// FillPostUserState sets whether the user liked, bookmarked or flagged each post and what they may do with it.
func FillPostUserState(user *User, topic *Topic, posts []*Post) error {
	if user == nil || len(posts) == 0 {
		return nil
	}
	ids := []int64{}
	for _, post := range posts {
		ids = append(ids, post.Id)
	}
	actions := []*PostAction{}
	err := adapter.Engine.Where("user_id = ? AND deleted = ?", user.Id, false).In("post_id", ids).Find(&actions)
	if err != nil {
		return err
	}

	for _, post := range posts {
		for _, action := range actions {
			if action.PostId != post.Id {
				continue
			}
			switch {
			case action.ActionType == PostActionLike:
				post.Liked = true
			case action.ActionType == PostActionBookmark:
				post.Bookmarked = true
			case IsFlagType(action.ActionType):
				post.Flagged = true
			}
		}
		post.CanEdit = CanEditPost(user, topic, post)
		post.CanDelete = !post.Deleted && post.PostType == PostTypeRegular && (user.IsStaff() || (post.UserId == user.Id && post.PostNumber > 1 && !topic.Archived))
		post.CanRecover = post.Deleted && user.IsStaff()
	}
	return nil
}

func checkCanReply(user *User, topic *Topic) error {
	err := checkCanPost(user)
	if err != nil {
		return err
	}
	canSee, err := CanSeeTopic(user, topic)
	if err != nil {
		return err
	}
	if !canSee {
		return fmt.Errorf("The topic does not exist")
	}
	if topic.Archived {
		return fmt.Errorf("The topic is archived")
	}
	if topic.Closed && !user.IsStaff() {
		return fmt.Errorf("The topic is closed")
	}
	return nil
}

func CanReply(user *User, topic *Topic) bool {
	return checkCanReply(user, topic) == nil
}

func AddPost(user *User, topicId int64, raw string, replyToPostNumber int) (*Post, error) {
	topic, err := GetTopic(topicId)
	if err != nil {
		return nil, err
	}
	if topic == nil {
		return nil, fmt.Errorf("The topic does not exist")
	}
	err = checkCanReply(user, topic)
	if err != nil {
		return nil, err
	}
	minSetting := "min_post_length"
	if topic.IsPrivateMessage() {
		minSetting = "min_personal_message_post_length"
	}
	err = validateRaw(raw, minSetting)
	if err != nil {
		return nil, err
	}

	var replyTo *Post
	if replyToPostNumber > 0 {
		replyTo, err = GetPostByNumber(topic.Id, replyToPostNumber)
		if err != nil {
			return nil, err
		}
		if replyTo == nil || replyTo.Deleted {
			replyToPostNumber = 0
			replyTo = nil
		}
	}

	postLock.Lock()
	defer postLock.Unlock()

	topic, err = GetTopic(topicId)
	if err != nil {
		return nil, err
	}
	cooked := CookMarkdown(raw)
	now := time.Now()
	post := &Post{
		TopicId:           topic.Id,
		PostNumber:        topic.HighestPostNumber + 1,
		UserId:            user.Id,
		Raw:               raw,
		Cooked:            cooked.Html,
		PostType:          PostTypeRegular,
		ReplyToPostNumber: replyToPostNumber,
		Version:           1,
	}
	if replyTo != nil {
		post.ReplyToUserId = replyTo.UserId
	}
	_, err = adapter.Engine.Insert(post)
	if err != nil {
		return nil, err
	}

	topic.HighestPostNumber = post.PostNumber
	topic.PostsCount++
	topic.ReplyCount++
	topic.LastPostUserId = user.Id
	topic.LastPostedAt = now
	topic.BumpedAt = now
	topic.WordCount += countWords(raw)
	_, err = adapter.Engine.ID(topic.Id).Cols("highest_post_number", "posts_count", "reply_count", "last_post_user_id", "last_posted_at", "bumped_at", "word_count").Update(topic)
	if err != nil {
		return nil, err
	}
	err = refreshTopicPosters(topic)
	if err != nil {
		return nil, err
	}

	if replyTo != nil {
		_, _ = adapter.Engine.ID(replyTo.Id).Incr("reply_count").Update(&Post{})
	}
	if !topic.IsPrivateMessage() {
		incrCategory(topic.CategoryId, "post_count", 1)
	}
	incrUser(user.Id, "post_count", 1)
	_, _ = adapter.Engine.ID(user.Id).Cols("last_posted_at").Update(&User{LastPostedAt: now})

	err = SetTopicUser(user.Id, topic.Id, func(tu *TopicUser) {
		tu.Posted = true
		tu.LastReadPostNumber = post.PostNumber
		if tu.NotificationLevel < NotificationLevelTracking {
			tu.NotificationLevel = NotificationLevelTracking
		}
	})
	if err != nil {
		return nil, err
	}

	notifyNewPost(user, topic, post, cooked)
	return post, nil
}

func addSmallAction(user *User, topic *Topic, actionCode string) error {
	postLock.Lock()
	defer postLock.Unlock()

	topic, err := GetTopic(topic.Id)
	if err != nil {
		return err
	}
	post := &Post{
		TopicId:    topic.Id,
		PostNumber: topic.HighestPostNumber + 1,
		UserId:     user.Id,
		PostType:   PostTypeSmallAction,
		ActionCode: actionCode,
		Version:    1,
	}
	_, err = adapter.Engine.Insert(post)
	if err != nil {
		return err
	}
	_, err = adapter.Engine.ID(topic.Id).Cols("highest_post_number").Update(&Topic{HighestPostNumber: post.PostNumber})
	return err
}

type PostUpdate struct {
	Id         int64  `json:"id"`
	Raw        string `json:"raw"`
	EditReason string `json:"editReason"`
}

func UpdatePost(user *User, input *PostUpdate) (*Post, error) {
	post, err := GetPost(input.Id)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, fmt.Errorf("The post does not exist")
	}
	topic, err := GetTopic(post.TopicId)
	if err != nil {
		return nil, err
	}
	if topic == nil || !CanEditPost(user, topic, post) {
		return nil, fmt.Errorf("You can't edit this post")
	}
	minSetting := "min_post_length"
	if post.PostNumber == 1 {
		minSetting = "min_first_post_length"
	}
	if topic.IsPrivateMessage() {
		minSetting = "min_personal_message_post_length"
	}
	err = validateRaw(input.Raw, minSetting)
	if err != nil {
		return nil, err
	}
	if input.Raw == post.Raw {
		return post, nil
	}

	revision := &PostRevision{
		PostId:    post.Id,
		Number:    post.Version + 1,
		UserId:    user.Id,
		RawBefore: post.Raw,
		RawAfter:  input.Raw,
		Reason:    input.EditReason,
	}
	_, err = adapter.Engine.Insert(revision)
	if err != nil {
		return nil, err
	}

	cooked := CookMarkdown(input.Raw)
	post.Raw = input.Raw
	post.Cooked = cooked.Html
	post.Version++
	post.EditReason = input.EditReason
	post.LastEditorId = user.Id
	_, err = adapter.Engine.ID(post.Id).Cols("raw", "cooked", "version", "edit_reason", "last_editor_id").Update(post)
	if err != nil {
		return nil, err
	}

	if post.PostNumber == 1 {
		topic.Excerpt = cooked.Excerpt
		topic.ImageUrl = cooked.ImageUrl
		_, err = adapter.Engine.ID(topic.Id).Cols("excerpt", "image_url").Update(topic)
		if err != nil {
			return nil, err
		}
	}
	notifyMentions(user, topic, post, cooked, true)
	return post, nil
}

func GetPostRevisions(postId int64) ([]*PostRevision, error) {
	revisions := []*PostRevision{}
	err := adapter.Engine.Where("post_id = ?", postId).Asc("number").Find(&revisions)
	return revisions, err
}

// DeletePost hides a reply for everyone but staff; deleting the first post deletes the topic.
func DeletePost(user *User, postId int64) error {
	post, err := GetPost(postId)
	if err != nil {
		return err
	}
	if post == nil || post.Deleted {
		return fmt.Errorf("The post does not exist")
	}
	topic, err := GetTopic(post.TopicId)
	if err != nil {
		return err
	}
	if topic == nil {
		return fmt.Errorf("The topic does not exist")
	}
	if post.PostNumber == 1 {
		return DeleteTopic(user, topic.Id)
	}
	if !user.IsStaff() && (post.UserId != user.Id || topic.Archived) {
		return fmt.Errorf("You can't delete this post")
	}
	return setPostDeleted(user, topic, post, true)
}

func RecoverPost(user *User, postId int64) error {
	post, err := GetPost(postId)
	if err != nil {
		return err
	}
	if post == nil || !post.Deleted {
		return fmt.Errorf("The post is not deleted")
	}
	if !user.IsStaff() {
		return fmt.Errorf("Only staff can do this")
	}
	topic, err := GetTopic(post.TopicId)
	if err != nil {
		return err
	}
	if post.PostNumber == 1 {
		return RecoverTopic(user, topic.Id)
	}
	return setPostDeleted(user, topic, post, false)
}

func setPostDeleted(user *User, topic *Topic, post *Post, deleted bool) error {
	post.Deleted = deleted
	post.DeletedById = 0
	if deleted {
		post.DeletedById = user.Id
	}
	_, err := adapter.Engine.ID(post.Id).Cols("deleted", "deleted_by_id").Update(post)
	if err != nil {
		return err
	}
	if post.PostType != PostTypeRegular {
		return nil
	}

	delta := 1
	if deleted {
		delta = -1
	}
	_, err = adapter.Engine.ID(topic.Id).Incr("posts_count", delta).Incr("reply_count", delta).Update(&Topic{})
	if err != nil {
		return err
	}
	if !topic.IsPrivateMessage() {
		incrCategory(topic.CategoryId, "post_count", delta)
	}
	incrUser(post.UserId, "post_count", delta)
	if post.ReplyToPostNumber > 0 {
		_, _ = adapter.Engine.Where("topic_id = ? AND post_number = ?", topic.Id, post.ReplyToPostNumber).Incr("reply_count", delta).Update(&Post{})
	}

	// the last poster and the bump follow the newest post that is left
	last := &Post{}
	has, err := adapter.Engine.Where("topic_id = ? AND deleted = ? AND post_type = ?", topic.Id, false, PostTypeRegular).Desc("post_number").Get(last)
	if err != nil {
		return err
	}
	if has {
		topic, err = GetTopic(topic.Id)
		if err != nil {
			return err
		}
		topic.LastPostUserId = last.UserId
		topic.LastPostedAt = last.CreatedAt
		_, err = adapter.Engine.ID(topic.Id).Cols("last_post_user_id", "last_posted_at").Update(topic)
		if err != nil {
			return err
		}
		return refreshTopicPosters(topic)
	}
	return nil
}

// GetPostReplies are the direct replies to a post, for its "2 Replies" button.
func GetPostReplies(user *User, topic *Topic, post *Post) ([]*Post, error) {
	posts := []*Post{}
	err := adapter.Engine.Where("topic_id = ? AND reply_to_post_number = ? AND deleted = ?", topic.Id, post.PostNumber, false).
		Omit("raw").Asc("post_number").Find(&posts)
	if err != nil {
		return nil, err
	}
	err = FillPostUserState(user, topic, posts)
	return posts, err
}

type UserPost struct {
	Post  *Post  `json:"post"`
	Topic *Topic `json:"topic"`
}

// GetUserPosts lists a user's posts with their topics, newest first: "replies" leaves out the first posts of topics.
func GetUserPosts(viewer *User, userId int64, filter string, page int) ([]*UserPost, bool, error) {
	const pageSize = 30
	session := adapter.Engine.Table("post").Select("post.*").Join("INNER", "topic", "topic.id = post.topic_id").
		Where("post.user_id = ? AND post.deleted = ? AND post.post_type = ? AND topic.deleted = ? AND topic.archetype = ?",
			userId, false, PostTypeRegular, false, ArchetypeRegular)
	if !viewer.IsStaff() {
		session = session.And("topic.visible = ?", true)
	}
	hidden := GetHiddenCategoryIds(viewer)
	if len(hidden) > 0 {
		session = session.NotIn("topic.category_id", hidden)
	}
	if filter == "replies" {
		session = session.And("post.post_number > ?", 1)
	}

	posts := []*Post{}
	err := session.Desc("post.created_at").Limit(pageSize+1, page*pageSize).Find(&posts)
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

func attachTopics(posts []*Post) ([]*UserPost, error) {
	topicIds := []int64{}
	for _, post := range posts {
		topicIds = append(topicIds, post.TopicId)
	}
	topics := []*Topic{}
	if len(topicIds) > 0 {
		err := adapter.Engine.In("id", uniqueIds(topicIds)).Find(&topics)
		if err != nil {
			return nil, err
		}
	}
	topicMap := map[int64]*Topic{}
	for _, topic := range topics {
		topicMap[topic.Id] = topic
	}

	res := []*UserPost{}
	for _, post := range posts {
		post.Raw = ""
		res = append(res, &UserPost{Post: post, Topic: topicMap[post.TopicId]})
	}
	return res, nil
}

func CountPosts(since time.Time) (int64, error) {
	session := adapter.Engine.Where("deleted = ? AND post_type = ?", false, PostTypeRegular)
	if !since.IsZero() {
		session = session.And("created_at > ?", dbTime(since))
	}
	return session.Count(&Post{})
}

func PreviewMarkdown(raw string) string {
	return CookMarkdown(raw).Html
}
