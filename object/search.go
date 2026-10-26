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
	"strings"

	"github.com/xorm-io/builder"
)

type SearchResult struct {
	Posts      []*SearchPost    `json:"posts"`
	Topics     map[int64]*Topic `json:"topics"`
	Users      map[int64]*User  `json:"users"`
	Categories []*Category      `json:"categories"`
	Tags       []*Tag           `json:"tags"`
	MatchUsers []*User          `json:"matchUsers"`
	More       bool             `json:"more"`
}

type SearchPost struct {
	Id         int64  `json:"id"`
	TopicId    int64  `json:"topicId"`
	PostNumber int    `json:"postNumber"`
	UserId     int64  `json:"userId"`
	Blurb      string `json:"blurb"`
	LikeCount  int    `json:"likeCount"`
	CreatedAt  string `json:"createdAt"`
}

type searchQuery struct {
	terms      []string
	username   string
	category   string
	tag        string
	inTitle    bool
	order      string
	status     string
	inMessages bool
}

// parseSearch splits the words from the filters Discourse's search understands: @user, #category, tags:x,
// in:title, in:messages, status:open|closed, order:latest|likes|views.
func parseSearch(q string) *searchQuery {
	res := &searchQuery{}
	for _, word := range strings.Fields(q) {
		lower := strings.ToLower(word)
		switch {
		case strings.HasPrefix(word, "@") && len(word) > 1:
			res.username = word[1:]
		case strings.HasPrefix(word, "#") && len(word) > 1:
			res.category = word[1:]
		case strings.HasPrefix(lower, "tags:") || strings.HasPrefix(lower, "tag:"):
			res.tag = word[strings.Index(word, ":")+1:]
		case lower == "in:title":
			res.inTitle = true
		case lower == "in:messages" || lower == "in:personal":
			res.inMessages = true
		case strings.HasPrefix(lower, "order:"):
			res.order = strings.TrimPrefix(lower, "order:")
		case strings.HasPrefix(lower, "status:"):
			res.status = strings.TrimPrefix(lower, "status:")
		default:
			res.terms = append(res.terms, lower)
		}
	}
	return res
}

func Search(user *User, q string, page int, limit int) (*SearchResult, error) {
	query := parseSearch(q)
	res := &SearchResult{Posts: []*SearchPost{}, Topics: map[int64]*Topic{}, Users: map[int64]*User{}, Categories: []*Category{}, Tags: []*Tag{}, MatchUsers: []*User{}}
	if len(query.terms) == 0 && query.username == "" && query.category == "" && query.tag == "" {
		return res, nil
	}

	session := adapter.Engine.Table("post").Select("post.*").Join("INNER", "topic", "topic.id = post.topic_id").
		Where("post.deleted = ? AND post.hidden = ? AND post.post_type = ? AND topic.deleted = ?", false, false, PostTypeRegular, false)
	if !user.IsStaff() {
		session = session.And("topic.visible = ?", true)
	}
	if query.inMessages && user != nil {
		session = session.And("topic.archetype = ?", ArchetypePrivateMessage).
			And(builder.In("topic.id", builder.Select("topic_id").From("topic_allowed_user").Where(builder.Eq{"user_id": user.Id})))
	} else {
		session = session.And("topic.archetype = ?", ArchetypeRegular)
		hidden := GetHiddenCategoryIds(user)
		if len(hidden) > 0 {
			session = session.NotIn("topic.category_id", hidden)
		}
	}

	for _, term := range query.terms {
		like := "%" + escapeLike(term) + "%"
		if query.inTitle {
			session = session.And(builder.Expr("LOWER(topic.title) LIKE ?", like))
		} else {
			session = session.And(builder.Or(builder.Expr("LOWER(topic.title) LIKE ?", like), builder.Expr("LOWER(post.raw) LIKE ?", like)))
		}
	}
	if query.inTitle {
		session = session.And("post.post_number = ?", 1)
	}
	if query.username != "" {
		author, err := GetUserByUsername(query.username)
		if err != nil {
			return nil, err
		}
		authorId := int64(-1)
		if author != nil {
			authorId = author.Id
		}
		session = session.And("post.user_id = ?", authorId)
	}
	if query.category != "" {
		category, err := GetCategoryBySlug(query.category)
		if err != nil {
			return nil, err
		}
		ids := []int64{-1}
		if category != nil {
			ids = append([]int64{category.Id}, category.SubIds...)
		}
		session = session.In("topic.category_id", ids)
	}
	if query.tag != "" {
		tag, err := GetTag(query.tag)
		if err != nil {
			return nil, err
		}
		tagId := int64(-1)
		if tag != nil {
			tagId = tag.Id
		}
		session = session.And(builder.In("topic.id", builder.Select("topic_id").From("topic_tag").Where(builder.Eq{"tag_id": tagId})))
	}
	switch query.status {
	case "open":
		session = session.And("topic.closed = ?", false)
	case "closed":
		session = session.And("topic.closed = ?", true)
	}
	switch query.order {
	case "likes":
		session = session.Desc("post.like_count")
	case "views":
		session = session.Desc("topic.views")
	default:
		session = session.Desc("post.created_at")
	}

	posts := []*Post{}
	err := session.Limit(limit+1, page*limit).Find(&posts)
	if err != nil {
		return nil, err
	}
	res.More = len(posts) > limit
	if res.More {
		posts = posts[:limit]
	}

	topicIds := []int64{}
	userIds := []int64{}
	for _, post := range posts {
		topicIds = append(topicIds, post.TopicId)
		userIds = append(userIds, post.UserId)
		res.Posts = append(res.Posts, &SearchPost{
			Id:         post.Id,
			TopicId:    post.TopicId,
			PostNumber: post.PostNumber,
			UserId:     post.UserId,
			Blurb:      makeBlurb(post.Cooked, query.terms),
			LikeCount:  post.LikeCount,
			CreatedAt:  post.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	if len(topicIds) > 0 {
		topics := []*Topic{}
		err = adapter.Engine.In("id", uniqueIds(topicIds)).Find(&topics)
		if err != nil {
			return nil, err
		}
		for _, topic := range topics {
			res.Topics[topic.Id] = topic
		}
	}
	res.Users, err = GetUserMap(userIds)
	if err != nil {
		return nil, err
	}

	if page == 0 && len(query.terms) > 0 {
		term := strings.Join(query.terms, " ")
		categories, err := GetCategories(user)
		if err != nil {
			return nil, err
		}
		for _, category := range categories {
			if strings.Contains(strings.ToLower(category.Name), term) && len(res.Categories) < 5 {
				res.Categories = append(res.Categories, category)
			}
		}
		res.Tags, err = SearchTags(term, 5)
		if err != nil {
			return nil, err
		}
		res.MatchUsers, err = SearchUsers(term, 5)
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

// makeBlurb is the part of the post's text around the first search term.
func makeBlurb(cooked string, terms []string) string {
	text, _ := makeExcerptFull(cooked)
	lower := strings.ToLower(text)
	runes := []rune(text)
	for _, term := range terms {
		index := strings.Index(lower, term)
		if index < 0 {
			continue
		}
		start := len([]rune(lower[:index])) - 80
		prefix := "…"
		if start <= 0 {
			start = 0
			prefix = ""
		}
		end := start + 220
		suffix := "…"
		if end >= len(runes) {
			end = len(runes)
			suffix = ""
		}
		return prefix + string(runes[start:end]) + suffix
	}
	return truncateRunes(text, 220)
}

func makeExcerptFull(cooked string) (string, string) {
	excerpt, image := makeExcerpt(cooked)
	return strings.TrimSuffix(excerpt, "…"), image
}
