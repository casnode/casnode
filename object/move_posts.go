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
	"html"
	"net/url"
	"sort"
	"strings"
	"time"
)

// MovePostsInput moves posts of a topic to a new topic (title, category and tags) or to the end of an existing one;
// moving every post of a topic merges it into the other, and the emptied topic is deleted and points to it.
type MovePostsInput struct {
	TopicId            int64    `json:"topicId"`
	PostIds            []int64  `json:"postIds"`
	Title              string   `json:"title"`
	CategoryId         int64    `json:"categoryId"`
	Tags               []string `json:"tags"`
	DestinationTopicId int64    `json:"destinationTopicId"`
}

func MovePosts(user *User, input *MovePostsInput) (*Topic, error) {
	if !user.IsStaff() {
		return nil, fmt.Errorf("Only staff can move posts")
	}
	source, err := GetTopic(input.TopicId)
	if err != nil {
		return nil, err
	}
	if source == nil || source.Deleted {
		return nil, fmt.Errorf("The topic does not exist")
	}

	posts := []*Post{}
	if len(input.PostIds) > 0 {
		err = adapter.Engine.Where("topic_id = ? AND post_type = ?", source.Id, PostTypeRegular).In("id", input.PostIds).Asc("post_number").Find(&posts)
		if err != nil {
			return nil, err
		}
	}
	if len(posts) == 0 {
		return nil, fmt.Errorf("Select the posts to move")
	}
	remaining, err := adapter.Engine.Where("topic_id = ? AND post_type = ? AND deleted = ?", source.Id, PostTypeRegular, false).
		NotIn("id", input.PostIds).Count(&Post{})
	if err != nil {
		return nil, err
	}
	merging := remaining == 0
	if posts[0].PostNumber == 1 && !merging {
		return nil, fmt.Errorf("The first post can only be moved with all the others, to merge the topic into another")
	}

	var destination *Topic
	if input.DestinationTopicId != 0 {
		destination, err = GetTopic(input.DestinationTopicId)
		if err != nil {
			return nil, err
		}
		if destination == nil || destination.Deleted {
			return nil, fmt.Errorf("The destination topic does not exist")
		}
		if destination.Id == source.Id {
			return nil, fmt.Errorf("Choose another topic")
		}
		if destination.Archetype != source.Archetype {
			return nil, fmt.Errorf("Posts can't move between messages and topics")
		}
	} else {
		if merging {
			return nil, fmt.Errorf("Moving every post to a new topic is just renaming the topic")
		}
		destination, err = createTopicForMove(user, source, posts[0], input)
		if err != nil {
			return nil, err
		}
	}

	postLock.Lock()
	destination, err = GetTopic(destination.Id)
	if err != nil {
		postLock.Unlock()
		return nil, err
	}
	next := destination.HighestPostNumber
	numbers := map[int]int{}
	for _, post := range posts {
		next++
		numbers[post.PostNumber] = next
	}
	for _, post := range posts {
		oldNumber := post.PostNumber
		replyTo, replyToUser := numbers[post.ReplyToPostNumber], post.ReplyToUserId
		if replyTo == 0 {
			replyToUser = 0
		}
		_, err = adapter.Engine.ID(post.Id).Cols("topic_id", "post_number", "reply_to_post_number", "reply_to_user_id").
			Update(&Post{TopicId: destination.Id, PostNumber: numbers[oldNumber], ReplyToPostNumber: replyTo, ReplyToUserId: replyToUser})
		if err == nil {
			_, err = adapter.Engine.Where("topic_id = ? AND post_number = ?", source.Id, oldNumber).Cols("topic_id", "post_number", "topic_title", "slug").
				Update(&Notification{TopicId: destination.Id, PostNumber: numbers[oldNumber], TopicTitle: destination.Title, Slug: destination.Slug})
		}
		if err == nil {
			_, err = adapter.Engine.Where("post_id = ?", post.Id).Cols("topic_id").Update(&Reviewable{TopicId: destination.Id})
		}
		if err != nil {
			postLock.Unlock()
			return nil, err
		}
	}
	// the replies left behind to posts that moved now reply to nothing here
	movedNumbers := []int{}
	for old := range numbers {
		movedNumbers = append(movedNumbers, old)
	}
	_, err = adapter.Engine.Where("topic_id = ?", source.Id).In("reply_to_post_number", movedNumbers).
		Cols("reply_to_post_number", "reply_to_user_id").Update(&Post{})
	postLock.Unlock()
	if err != nil {
		return nil, err
	}

	for _, topic := range []*Topic{source, destination} {
		err = refreshTopicAfterMove(topic.Id)
		if err != nil {
			return nil, err
		}
	}
	authors := []int64{}
	for _, post := range posts {
		authors = append(authors, post.UserId)
	}
	for _, author := range uniqueIds(authors) {
		err = SetTopicUser(author, destination.Id, func(tu *TopicUser) {
			tu.Posted = true
			if tu.NotificationLevel < NotificationLevelTracking {
				tu.NotificationLevel = NotificationLevelTracking
			}
		})
		if err != nil {
			return nil, err
		}
	}
	if source.IsPrivateMessage() && input.DestinationTopicId == 0 {
		ids, err := GetTopicAllowedUserIds(source.Id)
		if err != nil {
			return nil, err
		}
		for _, id := range ids {
			_, err = adapter.Engine.Insert(&TopicAllowedUser{TopicId: destination.Id, UserId: id})
			if err != nil {
				return nil, err
			}
		}
	}

	destination, err = GetTopic(destination.Id)
	if err != nil {
		return nil, err
	}
	link := fmt.Sprintf("<a href=\"/t/%s/%d\">%s</a>", url.PathEscape(destination.Slug), destination.Id, html.EscapeString(destination.Title))
	if merging {
		source.Deleted = true
		source.DeletedById = user.Id
		source.MergedIntoId = destination.Id
		_, err = adapter.Engine.ID(source.Id).Cols("deleted", "deleted_by_id", "merged_into_id").Update(source)
		if err != nil {
			return nil, err
		}
		err = addSmallAction(user, source, "merged", link)
	} else {
		err = addSmallAction(user, source, "split_topic", link)
	}
	if err != nil {
		return nil, err
	}
	if input.DestinationTopicId != 0 {
		sourceLink := fmt.Sprintf("<a href=\"/t/%s/%d\">%s</a>", url.PathEscape(source.Slug), source.Id, html.EscapeString(source.Title))
		err = addSmallAction(user, destination, "moved_posts", sourceLink)
		if err != nil {
			return nil, err
		}
	}
	err = RefreshCategoryCounts()
	if err != nil {
		return nil, err
	}

	for _, author := range uniqueIds(authors) {
		if author == user.Id {
			continue
		}
		_ = addNotification(&Notification{
			UserId:           author,
			NotificationType: NotificationMovedPost,
			TopicId:          destination.Id,
			PostNumber:       numbers[posts[0].PostNumber],
			ActingUserId:     user.Id,
		}, destination)
	}
	PublishToTopic(source, map[string]interface{}{"type": "moved", "destinationId": destination.Id})
	PublishToTopic(destination, map[string]interface{}{"type": "status"})
	PublishLatest(destination, "bumped")
	return destination, nil
}

func createTopicForMove(user *User, source *Topic, first *Post, input *MovePostsInput) (*Topic, error) {
	title, err := validateTitle(input.Title)
	if err != nil {
		return nil, err
	}
	topic := &Topic{
		Title:     title,
		Slug:      Slugify(title),
		UserId:    first.UserId,
		Archetype: source.Archetype,
		Visible:   true,
	}
	if !source.IsPrivateMessage() {
		categoryId := input.CategoryId
		if categoryId == 0 {
			categoryId = source.CategoryId
		}
		category, err := GetCategory(categoryId)
		if err != nil {
			return nil, err
		}
		if category == nil {
			return nil, fmt.Errorf("The category does not exist")
		}
		topic.CategoryId = category.Id
		topic.Tags, err = normalizeTags(user, input.Tags)
		if err != nil {
			return nil, err
		}
	}
	_, err = adapter.Engine.Insert(topic)
	if err != nil {
		return nil, err
	}
	// the new topic starts when its first post was written
	_, err = adapter.Engine.ID(topic.Id).Cols("created_at").Update(&Topic{CreatedAt: first.CreatedAt})
	if err != nil {
		return nil, err
	}
	if !topic.IsPrivateMessage() {
		err = setTopicTags(topic.Id, topic.Tags)
		if err != nil {
			return nil, err
		}
	}
	incrUser(first.UserId, "topic_count", 1)
	return topic, nil
}

// refreshTopicAfterMove recounts a topic from the posts it has now.
func refreshTopicAfterMove(topicId int64) error {
	topic, err := GetTopic(topicId)
	if err != nil || topic == nil {
		return err
	}
	posts := []*Post{}
	err = adapter.Engine.Where("topic_id = ?", topic.Id).Asc("post_number").Find(&posts)
	if err != nil {
		return err
	}

	replies := map[int]int{}
	for _, post := range posts {
		if !post.Deleted && post.ReplyToPostNumber > 0 {
			replies[post.ReplyToPostNumber]++
		}
	}
	topic.PostsCount, topic.LikeCount, topic.WordCount, topic.HighestPostNumber = 0, 0, 0, 0
	var last *Post
	for _, post := range posts {
		if post.PostNumber > topic.HighestPostNumber {
			topic.HighestPostNumber = post.PostNumber
		}
		if post.PostType == PostTypeRegular && post.ReplyCount != replies[post.PostNumber] {
			_, err = adapter.Engine.ID(post.Id).Cols("reply_count").Update(&Post{ReplyCount: replies[post.PostNumber]})
			if err != nil {
				return err
			}
		}
		if post.Deleted || post.PostType != PostTypeRegular {
			continue
		}
		topic.PostsCount++
		topic.LikeCount += post.LikeCount
		topic.WordCount += countWords(post.Raw)
		if last == nil || post.CreatedAt.After(last.CreatedAt) {
			last = post
		}
		if topic.PostsCount == 1 {
			topic.UserId = post.UserId
			cooked := CookMarkdown(post.Raw)
			topic.Excerpt = cooked.Excerpt
			topic.ImageUrl = cooked.ImageUrl
		}
	}
	topic.ReplyCount = topic.PostsCount - 1
	if topic.ReplyCount < 0 {
		topic.ReplyCount = 0
	}
	if last != nil {
		topic.LastPostUserId = last.UserId
		topic.LastPostedAt = last.CreatedAt
		if last.CreatedAt.After(topic.BumpedAt) || topic.BumpedAt.IsZero() || topic.BumpedAt.Year() <= 1 {
			topic.BumpedAt = last.CreatedAt
		}
	}
	if topic.BumpedAt.IsZero() || topic.BumpedAt.Year() <= 1 {
		topic.BumpedAt = time.Now()
	}
	_, err = adapter.Engine.ID(topic.Id).Cols("posts_count", "reply_count", "like_count", "word_count", "highest_post_number",
		"last_post_user_id", "last_posted_at", "bumped_at", "user_id", "excerpt", "image_url").Update(topic)
	if err != nil {
		return err
	}
	return refreshTopicPosters(topic)
}

// SearchTopicsForMove finds the topics to move posts into, by title or by id or link.
func SearchTopicsForMove(user *User, term string, archetype string) ([]*Topic, error) {
	topics := []*Topic{}
	session := adapter.Engine.Cols("id", "title", "slug", "category_id", "archetype", "posts_count", "bumped_at").
		Where("deleted = ? AND archetype = ?", false, archetype)
	if id := parseTopicReference(term); id != 0 {
		session = session.And("id = ?", id)
	} else if term != "" {
		session = session.And("LOWER(title) LIKE ?", "%"+escapeLike(strings.ToLower(term))+"%")
	}
	err := session.Desc("bumped_at").Limit(10).Find(&topics)
	if err != nil {
		return nil, err
	}
	res := []*Topic{}
	for _, topic := range topics {
		canSee, err := CanSeeTopic(user, topic)
		if err == nil && canSee {
			res = append(res, topic)
		}
	}
	sort.SliceStable(res, func(i, j int) bool { return res[i].BumpedAt.After(res[j].BumpedAt) })
	return res, nil
}

// parseTopicReference reads a topic id from "123" or a topic's address.
func parseTopicReference(term string) int64 {
	var id int64
	if _, err := fmt.Sscanf(term, "%d", &id); err == nil && fmt.Sprint(id) == term {
		return id
	}
	parsed, err := url.Parse(term)
	if err != nil {
		return 0
	}
	if m := internalTopicRegex.FindStringSubmatch(parsed.Path); m != nil {
		_, _ = fmt.Sscanf(m[1], "%d", &id)
		return id
	}
	return 0
}
