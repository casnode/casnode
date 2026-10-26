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
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/xorm-io/builder"
	"github.com/xorm-io/xorm"
)

const (
	ArchetypeRegular        = "regular"
	ArchetypePrivateMessage = "private_message"
)

const TopicPageSize = 30

type Topic struct {
	Id                int64     `xorm:"pk autoincr" json:"id"`
	Title             string    `xorm:"varchar(300) notnull" json:"title"`
	Slug              string    `xorm:"varchar(300)" json:"slug"`
	CategoryId        int64     `xorm:"index" json:"categoryId"`
	UserId            int64     `xorm:"index" json:"userId"`
	Archetype         string    `xorm:"varchar(30) index" json:"archetype"`
	Excerpt           string    `xorm:"varchar(1000)" json:"excerpt,omitempty"`
	ImageUrl          string    `xorm:"varchar(500)" json:"imageUrl,omitempty"`
	PinnedAt          time.Time `json:"pinnedAt"`
	PinnedGlobally    bool      `json:"pinnedGlobally"`
	Closed            bool      `json:"closed"`
	Archived          bool      `json:"archived"`
	Visible           bool      `json:"visible"`
	Deleted           bool      `xorm:"index" json:"deleted"`
	DeletedById       int64     `json:"deletedById,omitempty"`
	Views             int       `json:"views"`
	PostsCount        int       `json:"postsCount"`
	ReplyCount        int       `json:"replyCount"`
	LikeCount         int       `json:"likeCount"`
	HighestPostNumber int       `json:"highestPostNumber"`
	ParticipantCount  int       `json:"participantCount"`
	WordCount         int       `json:"wordCount"`
	LastPostUserId    int64     `json:"lastPostUserId"`
	LastPostedAt      time.Time `json:"lastPostedAt"`
	BumpedAt          time.Time `xorm:"index" json:"bumpedAt"`
	CreatedAt         time.Time `xorm:"created index" json:"createdAt"`
	UpdatedAt         time.Time `xorm:"updated" json:"updatedAt"`
	PosterIds         []int64   `xorm:"varchar(500)" json:"posterIds"`
	Tags              []string  `xorm:"varchar(1000)" json:"tags"`
	HotScore          float64   `xorm:"index" json:"-"`
	// the topic this one was merged into, where its address now leads
	MergedIntoId int64 `json:"mergedIntoId,omitempty"`

	// what the signed-in user has done with the topic
	LastReadPostNumber int  `xorm:"-" json:"lastReadPostNumber"`
	UnreadPosts        int  `xorm:"-" json:"unreadPosts"`
	Unseen             bool `xorm:"-" json:"unseen"`
	NotificationLevel  int  `xorm:"-" json:"notificationLevel"`
}

func (t *Topic) IsPinned() bool {
	return !t.PinnedAt.IsZero() && t.PinnedAt.Year() > 1
}

func (t *Topic) IsPrivateMessage() bool {
	return t.Archetype == ArchetypePrivateMessage
}

func GetTopic(id int64) (*Topic, error) {
	topic := &Topic{}
	has, err := adapter.Engine.ID(id).Get(topic)
	if err != nil || !has {
		return nil, err
	}
	return topic, nil
}

// CanSee tells whether the user may read the topic: deleted and unlisted ones only for staff, messages only for
// their participants, and the category's restriction.
func CanSeeTopic(user *User, topic *Topic) (bool, error) {
	if topic == nil {
		return false, nil
	}
	if (topic.Deleted || !topic.Visible) && !user.IsStaff() {
		return false, nil
	}
	if topic.IsPrivateMessage() {
		if user == nil {
			return false, nil
		}
		if user.Admin {
			return true, nil
		}
		allowed, err := IsTopicAllowedUser(topic.Id, user.Id)
		if err != nil || allowed {
			return allowed, err
		}
		return isTopicAllowedGroupMember(topic.Id, user), nil
	}
	category, err := GetCategory(topic.CategoryId)
	if err != nil {
		return false, err
	}
	return category == nil || category.CanSee(user), nil
}

type TopicListQuery struct {
	Filter     string
	CategoryId int64
	NoSubs     bool
	Tag        string
	Period     string
	Order      string
	Ascending  bool
	Page       int
	UserId     int64 // topics of this user (created / posted / messages)
	GroupId    int64 // the messages of this group
	PageSize   int
}

type TopicList struct {
	Topics []*Topic         `json:"topics"`
	Users  map[int64]*User  `json:"users"`
	More   bool             `json:"more"`
	Page   int              `json:"page"`
	Extra  map[string]int64 `json:"extra,omitempty"`
}

var topicOrders = map[string]string{
	"activity": "topic.bumped_at",
	"created":  "topic.created_at",
	"views":    "topic.views",
	"posts":    "topic.posts_count",
	"likes":    "topic.like_count",
}

func PeriodStart(period string) time.Time {
	now := time.Now()
	switch period {
	case "daily":
		return now.AddDate(0, 0, -1)
	case "weekly":
		return now.AddDate(0, 0, -7)
	case "monthly":
		return now.AddDate(0, -1, 0)
	case "quarterly":
		return now.AddDate(0, -3, 0)
	case "yearly":
		return now.AddDate(-1, 0, 0)
	default:
		return time.Time{}
	}
}

func newTopicSince(user *User) time.Time {
	days := GetSiteSettingInt("new_topic_duration_days")
	if days <= 0 {
		days = 2
	}
	since := time.Now().AddDate(0, 0, -days)
	if user != nil && user.CreatedAt.After(since) {
		// a new user's "new" starts a little before they joined, like Discourse
		since = user.CreatedAt.AddDate(0, 0, -2)
	}
	return since
}

func applyCategoryFilter(session *xorm.Session, query *TopicListQuery) (*xorm.Session, error) {
	if query.CategoryId == 0 {
		return session, nil
	}
	ids := []int64{query.CategoryId}
	if !query.NoSubs {
		category, err := GetCategory(query.CategoryId)
		if err != nil {
			return nil, err
		}
		if category != nil {
			ids = append(ids, category.SubIds...)
		}
	}
	return session.In("topic.category_id", ids), nil
}

func newTopicListSession(user *User, query *TopicListQuery) (*xorm.Session, error) {
	session := adapter.Engine.Table("topic").Where("topic.deleted = ?", false)
	if !user.IsStaff() {
		session = session.And("topic.visible = ?", true)
	}

	if strings.HasPrefix(query.Filter, "private_messages") {
		if user == nil {
			return nil, fmt.Errorf("Please sign in first")
		}
		userId := user.Id
		if query.UserId != 0 {
			userId = query.UserId
		}
		session = session.And("topic.archetype = ?", ArchetypePrivateMessage)
		switch query.Filter {
		case "private_messages_sent":
			session = session.And("topic.user_id = ?", userId)
		case "private_messages_group":
			session = session.And(builder.In("topic.id", builder.Select("topic_id").From("topic_allowed_group").Where(builder.Eq{"group_id": query.GroupId})))
		default:
			session = session.And(builder.In("topic.id", builder.Select("topic_id").From("topic_allowed_user").Where(builder.Eq{"user_id": userId})))
		}
		return session, nil
	}

	session = session.And("topic.archetype = ?", ArchetypeRegular)
	hidden := GetHiddenCategoryIds(user)
	if len(hidden) > 0 {
		session = session.NotIn("topic.category_id", hidden)
	}
	session, err := applyCategoryFilter(session, query)
	if err != nil {
		return nil, err
	}

	if query.Tag != "" {
		tag, err := GetTag(query.Tag)
		if err != nil {
			return nil, err
		}
		tagId := int64(-1)
		if tag != nil {
			tagId = tag.Id
		}
		session = session.And(builder.In("topic.id", builder.Select("topic_id").From("topic_tag").Where(builder.Eq{"tag_id": tagId})))
	}

	switch query.Filter {
	case "new":
		if user == nil {
			return nil, fmt.Errorf("Please sign in first")
		}
		session = session.Join("LEFT", "topic_user", "topic_user.topic_id = topic.id AND topic_user.user_id = ?", user.Id).
			And("topic_user.id IS NULL").And("topic.created_at > ?", dbTime(newTopicSince(user)))
	case "unread":
		if user == nil {
			return nil, fmt.Errorf("Please sign in first")
		}
		session = session.Join("INNER", "topic_user", "topic_user.topic_id = topic.id AND topic_user.user_id = ?", user.Id).
			And("topic_user.notification_level >= ?", NotificationLevelTracking).
			And("topic.highest_post_number > topic_user.last_read_post_number")
	case "posted":
		if user == nil {
			return nil, fmt.Errorf("Please sign in first")
		}
		session = session.Join("INNER", "topic_user", "topic_user.topic_id = topic.id AND topic_user.user_id = ?", user.Id).
			And("topic_user.posted = ?", true)
	case "created":
		session = session.And("topic.user_id = ?", query.UserId)
	case "top":
		start := PeriodStart(query.Period)
		if !start.IsZero() {
			session = session.And("topic.created_at > ?", dbTime(start))
		}
	case "hot":
		session = session.And("topic.bumped_at > ?", dbTime(time.Now().AddDate(0, 0, -30)))
	}
	return session, nil
}

func GetTopicList(user *User, query *TopicListQuery) (*TopicList, error) {
	pageSize := query.PageSize
	if pageSize <= 0 {
		pageSize = TopicPageSize
	}

	session, err := newTopicListSession(user, query)
	if err != nil {
		return nil, err
	}
	session = session.Select("topic.*")

	// pinned topics go first on the first page of "latest": the globally pinned ones everywhere, the others in their category
	pinned := []*Topic{}
	showPinned := query.Filter == "latest" && query.Order == "" && query.Tag == ""
	if showPinned {
		session = session.And(builder.Or(builder.IsNull{"topic.pinned_at"}, builder.Expr("topic.pinned_at < ?", "1900-01-01")))
		if query.Page == 0 {
			pinnedSession, err := newTopicListSession(user, query)
			if err != nil {
				return nil, err
			}
			pinnedSession = pinnedSession.Select("topic.*").And("topic.pinned_at > ?", "1900-01-01")
			if query.CategoryId == 0 {
				pinnedSession = pinnedSession.And("topic.pinned_globally = ?", true)
			}
			err = pinnedSession.Desc("topic.pinned_at").Find(&pinned)
			if err != nil {
				return nil, err
			}
		}
	}

	switch query.Filter {
	case "top":
		session = session.Desc("topic.like_count", "topic.posts_count", "topic.views")
	case "hot":
		session = session.Desc("topic.hot_score", "topic.bumped_at")
	default:
		column, ok := topicOrders[query.Order]
		if !ok {
			column = "topic.bumped_at"
		}
		if query.Ascending {
			session = session.Asc(column)
		} else {
			session = session.Desc(column)
		}
	}
	session = session.Desc("topic.id")

	topics := []*Topic{}
	err = session.Limit(pageSize+1, query.Page*pageSize).Find(&topics)
	if err != nil {
		return nil, err
	}

	more := len(topics) > pageSize
	if more {
		topics = topics[:pageSize]
	}
	if len(pinned) > 0 {
		topics = append(pinned, topics...)
	}

	err = FillTopicUserState(user, topics)
	if err != nil {
		return nil, err
	}
	users, err := GetTopicListUsers(topics)
	if err != nil {
		return nil, err
	}
	return &TopicList{Topics: topics, Users: users, More: more, Page: query.Page}, nil
}

func GetTopicListUsers(topics []*Topic) (map[int64]*User, error) {
	ids := []int64{}
	for _, topic := range topics {
		ids = append(ids, topic.UserId, topic.LastPostUserId)
		ids = append(ids, topic.PosterIds...)
	}
	return GetUserMap(ids)
}

// FillTopicUserState sets what the user has read of each topic, and whether it is new to them.
func FillTopicUserState(user *User, topics []*Topic) error {
	if user == nil || len(topics) == 0 {
		return nil
	}
	ids := []int64{}
	for _, topic := range topics {
		ids = append(ids, topic.Id)
	}
	states := []*TopicUser{}
	err := adapter.Engine.Where("user_id = ?", user.Id).In("topic_id", ids).Find(&states)
	if err != nil {
		return err
	}
	stateMap := map[int64]*TopicUser{}
	for _, state := range states {
		stateMap[state.TopicId] = state
	}

	since := newTopicSince(user)
	for _, topic := range topics {
		state := stateMap[topic.Id]
		if state == nil {
			topic.Unseen = topic.CreatedAt.After(since)
			topic.NotificationLevel = NotificationLevelRegular
			continue
		}
		topic.LastReadPostNumber = state.LastReadPostNumber
		topic.NotificationLevel = state.NotificationLevel
		if state.NotificationLevel >= NotificationLevelTracking && topic.HighestPostNumber > state.LastReadPostNumber {
			topic.UnreadPosts = topic.HighestPostNumber - state.LastReadPostNumber
		}
	}
	return nil
}

// CountNewAndUnread is for the "New (3)" and "Unread (2)" navigation items.
func CountNewAndUnread(user *User) (int64, int64, error) {
	if user == nil {
		return 0, 0, nil
	}
	newSession, err := newTopicListSession(user, &TopicListQuery{Filter: "new"})
	if err != nil {
		return 0, 0, err
	}
	newCount, err := newSession.Count()
	if err != nil {
		return 0, 0, err
	}
	unreadSession, err := newTopicListSession(user, &TopicListQuery{Filter: "unread"})
	if err != nil {
		return 0, 0, err
	}
	unreadCount, err := unreadSession.Count()
	return newCount, unreadCount, err
}

type NewTopic struct {
	Title      string   `json:"title"`
	Raw        string   `json:"raw"`
	CategoryId int64    `json:"categoryId"`
	Tags       []string `json:"tags"`
	Archetype  string   `json:"archetype"`
	Recipients []string `json:"recipients"`
}

func validateTitle(title string) (string, error) {
	title = strings.Join(strings.Fields(title), " ")
	length := utf8.RuneCountInString(title)
	if min := GetSiteSettingInt("min_topic_title_length"); length < min {
		return "", fmt.Errorf("Title must be at least %d characters", min)
	}
	if max := GetSiteSettingInt("max_topic_title_length"); max > 0 && length > max {
		return "", fmt.Errorf("Title can't be more than %d characters", max)
	}
	return title, nil
}

func validateRaw(raw string, minSetting string) error {
	length := utf8.RuneCountInString(strings.TrimSpace(raw))
	if min := GetSiteSettingInt(minSetting); length < min {
		return fmt.Errorf("Body must be at least %d characters", min)
	}
	if max := GetSiteSettingInt("max_post_length"); max > 0 && length > max {
		return fmt.Errorf("Body can't be more than %d characters", max)
	}
	return nil
}

func checkCanPost(user *User) error {
	if user == nil {
		return fmt.Errorf("Please sign in first")
	}
	if user.IsSuspended() {
		return fmt.Errorf("Your account is suspended")
	}
	if user.IsSilenced() {
		return fmt.Errorf("Your account is silenced and can't post")
	}
	return nil
}

func CreateTopic(user *User, input *NewTopic) (*Topic, error) {
	err := checkCanPost(user)
	if err != nil {
		return nil, err
	}
	title, err := validateTitle(input.Title)
	if err != nil {
		return nil, err
	}

	topic := &Topic{
		Title:     title,
		Slug:      Slugify(title),
		UserId:    user.Id,
		Archetype: ArchetypeRegular,
		Visible:   true,
	}

	recipients := []*User{}
	recipientGroups := []*Group{}
	if input.Archetype == ArchetypePrivateMessage {
		if !GetSiteSettingBool("enable_personal_messages") && !user.IsStaff() {
			return nil, fmt.Errorf("Personal messages are turned off")
		}
		err = validateRaw(input.Raw, "min_personal_message_post_length")
		if err != nil {
			return nil, err
		}
		recipients, err = GetUsersByUsernames(input.Recipients)
		if err != nil {
			return nil, err
		}
		// the names that are not users are groups
		groupNames := []string{}
		for _, name := range input.Recipients {
			found := false
			for _, recipient := range recipients {
				found = found || strings.EqualFold(recipient.Username, name)
			}
			if !found {
				groupNames = append(groupNames, name)
			}
		}
		recipientGroups, err = GetGroupsByNames(groupNames)
		if err != nil {
			return nil, err
		}
		if len(recipientGroups) < len(groupNames) {
			return nil, fmt.Errorf("Some of the recipients don't exist")
		}
		for _, group := range recipientGroups {
			if !CanMessageGroup(user, group) {
				return nil, fmt.Errorf("You can't send messages to the group %s", group.Name)
			}
		}
		if len(recipients) == 0 && len(recipientGroups) == 0 {
			return nil, fmt.Errorf("Add at least one recipient")
		}
		topic.Archetype = ArchetypePrivateMessage
	} else {
		err = validateRaw(input.Raw, "min_first_post_length")
		if err != nil {
			return nil, err
		}
		categoryId := input.CategoryId
		if categoryId == 0 {
			if !GetSiteSettingBool("allow_uncategorized_topics") && !user.IsStaff() {
				return nil, fmt.Errorf("Choose a category")
			}
			categoryId = GetUncategorizedId()
		}
		category, err := GetCategory(categoryId)
		if err != nil {
			return nil, err
		}
		if category == nil {
			return nil, fmt.Errorf("The category does not exist")
		}
		if !category.CanCreateTopic(user) {
			return nil, fmt.Errorf("You can't create topics in %s", category.Name)
		}
		topic.CategoryId = category.Id
		topic.Tags, err = normalizeTags(user, input.Tags)
		if err != nil {
			return nil, err
		}
	}

	postLock.Lock()
	defer postLock.Unlock()

	now := time.Now()
	topic.BumpedAt = now
	topic.LastPostedAt = now
	topic.LastPostUserId = user.Id
	topic.PostsCount = 1
	topic.HighestPostNumber = 1
	topic.ParticipantCount = 1
	topic.PosterIds = []int64{user.Id}
	topic.WordCount = countWords(input.Raw)
	cooked := CookMarkdown(input.Raw)
	err = validatePolls(cooked.Polls)
	if err != nil {
		return nil, err
	}
	topic.Excerpt = cooked.Excerpt
	topic.ImageUrl = cooked.ImageUrl
	_, err = adapter.Engine.Insert(topic)
	if err != nil {
		return nil, err
	}

	post := &Post{
		TopicId:    topic.Id,
		PostNumber: 1,
		UserId:     user.Id,
		Raw:        input.Raw,
		Cooked:     cooked.Html,
		PostType:   PostTypeRegular,
		Version:    1,
	}
	_, err = adapter.Engine.Insert(post)
	if err != nil {
		return nil, err
	}

	if topic.IsPrivateMessage() {
		allowed := []int64{user.Id}
		for _, recipient := range recipients {
			allowed = append(allowed, recipient.Id)
		}
		for _, id := range uniqueIds(allowed) {
			_, err = adapter.Engine.Insert(&TopicAllowedUser{TopicId: topic.Id, UserId: id})
			if err != nil {
				return nil, err
			}
		}
		for _, group := range recipientGroups {
			_, err = adapter.Engine.Insert(&TopicAllowedGroup{TopicId: topic.Id, GroupId: group.Id})
			if err != nil {
				return nil, err
			}
		}
	} else {
		err = setTopicTags(topic.Id, topic.Tags)
		if err != nil {
			return nil, err
		}
		incrCategory(topic.CategoryId, "topic_count", 1)
		incrCategory(topic.CategoryId, "post_count", 1)
	}

	incrUser(user.Id, "topic_count", 1)
	incrUser(user.Id, "post_count", 1)
	_, _ = adapter.Engine.ID(user.Id).Cols("last_posted_at").Update(&User{LastPostedAt: now})
	err = SetTopicUser(user.Id, topic.Id, func(tu *TopicUser) {
		tu.Posted = true
		tu.LastReadPostNumber = 1
		tu.NotificationLevel = NotificationLevelWatching
	})
	if err != nil {
		return nil, err
	}

	err = SyncPolls(post.Id, cooked.Polls)
	if err != nil {
		return nil, err
	}
	RebakePostAfterOneboxes(post.Id, cooked.OneboxUrls)
	notifyNewPost(user, topic, post, cooked)
	PublishLatest(topic, "created")
	TriggerBadgeGrant()
	return topic, nil
}

type TopicUpdate struct {
	Id         int64    `json:"id"`
	Title      string   `json:"title"`
	CategoryId int64    `json:"categoryId"`
	Tags       []string `json:"tags"`
}

func CanEditTopic(user *User, topic *Topic) bool {
	if user == nil || topic == nil {
		return false
	}
	if user.IsStaff() {
		return true
	}
	return topic.UserId == user.Id && !topic.Closed && !topic.Archived && !user.IsSilenced() && !user.IsSuspended()
}

func UpdateTopic(user *User, input *TopicUpdate) (*Topic, error) {
	topic, err := GetTopic(input.Id)
	if err != nil {
		return nil, err
	}
	if topic == nil || !CanEditTopic(user, topic) {
		return nil, fmt.Errorf("You can't edit this topic")
	}

	title, err := validateTitle(input.Title)
	if err != nil {
		return nil, err
	}
	topic.Title = title
	topic.Slug = Slugify(title)

	oldCategoryId := topic.CategoryId
	if !topic.IsPrivateMessage() {
		if input.CategoryId != 0 && input.CategoryId != topic.CategoryId {
			category, err := GetCategory(input.CategoryId)
			if err != nil {
				return nil, err
			}
			if category == nil || !category.CanCreateTopic(user) {
				return nil, fmt.Errorf("You can't move the topic there")
			}
			topic.CategoryId = category.Id
		}
		topic.Tags, err = normalizeTags(user, input.Tags)
		if err != nil {
			return nil, err
		}
		err = setTopicTags(topic.Id, topic.Tags)
		if err != nil {
			return nil, err
		}
	}

	_, err = adapter.Engine.ID(topic.Id).Cols("title", "slug", "category_id", "tags").Update(topic)
	if err != nil {
		return nil, err
	}
	if oldCategoryId != topic.CategoryId {
		posts := topic.PostsCount
		incrCategory(oldCategoryId, "topic_count", -1)
		incrCategory(oldCategoryId, "post_count", -posts)
		incrCategory(topic.CategoryId, "topic_count", 1)
		incrCategory(topic.CategoryId, "post_count", posts)
	}
	return topic, nil
}

var topicStatuses = map[string]string{
	"closed":          "closed",
	"archived":        "archived",
	"visible":         "visible",
	"pinned":          "pinned_at",
	"pinned_globally": "pinned_at",
}

// SetTopicStatus closes, archives, unlists or pins a topic, and adds the small "closed this topic" post.
func SetTopicStatus(user *User, topicId int64, status string, enabled bool) error {
	if _, ok := topicStatuses[status]; !ok {
		return fmt.Errorf("Unknown status %s", status)
	}
	topic, err := GetTopic(topicId)
	if err != nil {
		return err
	}
	if topic == nil {
		return fmt.Errorf("The topic does not exist")
	}
	// the author may close their own topic; everything else is for staff
	if !user.IsStaff() && !(status == "closed" && topic.UserId == user.Id) {
		return fmt.Errorf("Only staff can do this")
	}

	switch status {
	case "closed":
		topic.Closed = enabled
		_, err = adapter.Engine.ID(topic.Id).Cols("closed").Update(topic)
	case "archived":
		topic.Archived = enabled
		_, err = adapter.Engine.ID(topic.Id).Cols("archived").Update(topic)
	case "visible":
		topic.Visible = enabled
		_, err = adapter.Engine.ID(topic.Id).Cols("visible").Update(topic)
	case "pinned", "pinned_globally":
		if enabled {
			topic.PinnedAt = time.Now()
			topic.PinnedGlobally = status == "pinned_globally"
			_, err = adapter.Engine.ID(topic.Id).Cols("pinned_at", "pinned_globally").Update(topic)
		} else {
			_, err = adapter.Engine.Exec("UPDATE topic SET pinned_at = NULL, pinned_globally = ? WHERE id = ?", false, topic.Id)
		}
	}
	if err != nil {
		return err
	}

	actionCode := fmt.Sprintf("%s.%s", status, map[bool]string{true: "enabled", false: "disabled"}[enabled])
	return addSmallAction(user, topic, actionCode, "")
}

func DeleteTopic(user *User, topicId int64) error {
	topic, err := GetTopic(topicId)
	if err != nil {
		return err
	}
	if topic == nil || topic.Deleted {
		return fmt.Errorf("The topic does not exist")
	}
	if !user.IsStaff() && !(topic.UserId == user.Id && topic.PostsCount <= 1) {
		return fmt.Errorf("You can't delete this topic")
	}
	return setTopicDeleted(user, topic, true)
}

func RecoverTopic(user *User, topicId int64) error {
	topic, err := GetTopic(topicId)
	if err != nil {
		return err
	}
	if topic == nil || !topic.Deleted {
		return fmt.Errorf("The topic is not deleted")
	}
	if !user.IsStaff() {
		return fmt.Errorf("Only staff can do this")
	}
	return setTopicDeleted(user, topic, false)
}

func setTopicDeleted(user *User, topic *Topic, deleted bool) error {
	topic.Deleted = deleted
	topic.DeletedById = 0
	if deleted {
		topic.DeletedById = user.Id
	}
	_, err := adapter.Engine.ID(topic.Id).Cols("deleted", "deleted_by_id").Update(topic)
	if err != nil {
		return err
	}

	PublishToTopic(topic, map[string]interface{}{"type": "status"})
	delta := 1
	if deleted {
		delta = -1
	}
	if !topic.IsPrivateMessage() {
		incrCategory(topic.CategoryId, "topic_count", delta)
		incrCategory(topic.CategoryId, "post_count", delta*topic.PostsCount)
		for _, tag := range topic.Tags {
			incrTag(tag, delta)
		}
	}
	incrUser(topic.UserId, "topic_count", delta)
	return nil
}

// RecordTopicView counts a view once per visitor and 8 hours, and remembers the topic was opened.
func RecordTopicView(user *User, topic *Topic, visitorKey string) {
	if shouldCountView(topic.Id, visitorKey) {
		_, _ = adapter.Engine.ID(topic.Id).Incr("views").Update(&Topic{})
		topic.Views++
	}
	if user == nil {
		return
	}
	_ = SetTopicUser(user.Id, topic.Id, func(tu *TopicUser) {
		if tu.FirstVisitedAt.IsZero() || tu.FirstVisitedAt.Year() <= 1 {
			tu.FirstVisitedAt = time.Now()
			incrUser(user.Id, "topics_entered", 1)
		}
		tu.LastVisitedAt = time.Now()
	})
}

// GetSuggestedTopics are shown under a topic: the user's new and unread topics first, then the latest of the category.
func GetSuggestedTopics(user *User, topic *Topic) ([]*Topic, map[int64]*User, error) {
	const limit = 5
	res := []*Topic{}
	seen := map[int64]bool{topic.Id: true}
	add := func(list *TopicList) {
		for _, t := range list.Topics {
			if len(res) < limit && !seen[t.Id] {
				seen[t.Id] = true
				res = append(res, t)
			}
		}
	}

	if topic.IsPrivateMessage() {
		list, err := GetTopicList(user, &TopicListQuery{Filter: "private_messages", PageSize: limit + 1})
		if err != nil {
			return nil, nil, err
		}
		add(list)
	} else {
		if user != nil {
			for _, filter := range []string{"unread", "new"} {
				list, err := GetTopicList(user, &TopicListQuery{Filter: filter, PageSize: limit + 1})
				if err != nil {
					return nil, nil, err
				}
				add(list)
			}
		}
		for _, categoryId := range []int64{topic.CategoryId, 0} {
			if len(res) >= limit {
				break
			}
			list, err := GetTopicList(user, &TopicListQuery{Filter: "latest", CategoryId: categoryId, Order: "activity", PageSize: limit + 1})
			if err != nil {
				return nil, nil, err
			}
			add(list)
		}
	}

	users, err := GetTopicListUsers(res)
	return res, users, err
}

// refreshTopicPosters recounts who posted in the topic: the original poster, the most frequent ones and the last one,
// shown as avatars in topic lists.
func refreshTopicPosters(topic *Topic) error {
	type row struct {
		UserId int64
		Count  int
	}
	rows := []row{}
	err := adapter.Engine.Table("post").Select("user_id, COUNT(*) AS count").
		Where("topic_id = ? AND deleted = ? AND post_type = ?", topic.Id, false, PostTypeRegular).
		GroupBy("user_id").OrderBy("count DESC").Find(&rows)
	if err != nil {
		return err
	}

	posters := []int64{topic.UserId}
	for _, r := range rows {
		if len(posters) >= 4 {
			break
		}
		if r.UserId != topic.UserId && r.UserId != topic.LastPostUserId {
			posters = append(posters, r.UserId)
		}
	}
	if topic.LastPostUserId != 0 && topic.LastPostUserId != topic.UserId {
		posters = append(posters, topic.LastPostUserId)
	}
	topic.PosterIds = posters
	topic.ParticipantCount = len(rows)
	_, err = adapter.Engine.ID(topic.Id).Cols("poster_ids", "participant_count").Update(topic)
	return err
}

func CountTopics(since time.Time) (int64, error) {
	session := adapter.Engine.Where("deleted = ? AND archetype = ?", false, ArchetypeRegular)
	if !since.IsZero() {
		session = session.And("created_at > ?", dbTime(since))
	}
	return session.Count(&Topic{})
}

// GetTopicsForSitemap lists the public topics, newest activity first, for sitemap.xml.
func GetTopicsForSitemap(page int, pageSize int) ([]*Topic, error) {
	topics := []*Topic{}
	session := adapter.Engine.Cols("id", "slug", "bumped_at", "posts_count").
		Where("deleted = ? AND visible = ? AND archetype = ?", false, true, ArchetypeRegular)
	hidden := GetHiddenCategoryIds(nil)
	if len(hidden) > 0 {
		session = session.NotIn("category_id", hidden)
	}
	err := session.Desc("bumped_at").Limit(pageSize, page*pageSize).Find(&topics)
	return topics, err
}

func CountPublicTopics() (int64, error) {
	session := adapter.Engine.Where("deleted = ? AND visible = ? AND archetype = ?", false, true, ArchetypeRegular)
	hidden := GetHiddenCategoryIds(nil)
	if len(hidden) > 0 {
		session = session.NotIn("category_id", hidden)
	}
	return session.Count(&Topic{})
}

type TopicParticipant struct {
	UserId int64
	Count  int
}

// GetTopicParticipants are the users who posted the most in the topic, for its summary.
func GetTopicParticipants(topicId int64, limit int) ([]*TopicParticipant, error) {
	rows := []*TopicParticipant{}
	err := adapter.Engine.Table("post").Select("user_id, COUNT(*) AS count").
		Where("topic_id = ? AND deleted = ? AND post_type = ?", topicId, false, PostTypeRegular).
		GroupBy("user_id").OrderBy("count DESC").Limit(limit).Find(&rows)
	return rows, err
}

// countWords counts the words of a text for the reading time; two CJK characters count as a word.
func countWords(raw string) int {
	cjk := 0
	for _, r := range raw {
		if unicode.Is(unicode.Han, r) || unicode.Is(unicode.Hiragana, r) || unicode.Is(unicode.Katakana, r) || unicode.Is(unicode.Hangul, r) {
			cjk++
		}
	}
	return len(strings.Fields(raw)) + cjk/2
}
