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
	"time"

	"github.com/beego/beego/v2/core/logs"
)

// The notification types follow Discourse's numbering.
const (
	NotificationMentioned               = 1
	NotificationReplied                 = 2
	NotificationQuoted                  = 3
	NotificationEdited                  = 4
	NotificationLiked                   = 5
	NotificationPrivateMessage          = 6
	NotificationInvitedToPrivateMessage = 7
	NotificationPosted                  = 9
	NotificationGrantedBadge            = 12
	NotificationMovedPost               = 14
	NotificationGroupMentioned          = 15
	NotificationAdminMessage            = 20
	NotificationMembershipAccepted      = 22
	NotificationMembershipRequest       = 23
	NotificationChatMention             = 29
	NotificationChatMessage             = 30
)

// Whether the email of a notification went out.
const (
	EmailPending = 0
	EmailSent    = 1
	EmailSkipped = 2
)

type Notification struct {
	Id               int64  `xorm:"pk autoincr" json:"id"`
	UserId           int64  `xorm:"index" json:"userId"`
	NotificationType int    `json:"notificationType"`
	TopicId          int64  `xorm:"index" json:"topicId"`
	PostNumber       int    `json:"postNumber"`
	ActingUserId     int64  `json:"actingUserId"`
	TopicTitle       string `xorm:"varchar(300)" json:"topicTitle"`
	Slug             string `xorm:"varchar(300)" json:"slug"`
	Excerpt          string `xorm:"varchar(500)" json:"excerpt,omitempty"`
	IsRead           bool   `xorm:"index" json:"read"`
	IsHighPriority   bool   `json:"highPriority"`
	// what the notification is about when it is not a post: the badge, the group, the chat message (JSON)
	Data        string    `xorm:"varchar(1000)" json:"data,omitempty"`
	EmailStatus int       `xorm:"index" json:"-"`
	CreatedAt   time.Time `xorm:"created index" json:"createdAt"`
}

func addNotification(notification *Notification, topic *Topic) error {
	if notification.UserId == 0 || notification.UserId == notification.ActingUserId {
		return nil
	}
	if topic != nil {
		notification.TopicTitle = topic.Title
		notification.Slug = topic.Slug
	}
	_, err := adapter.Engine.Insert(notification)
	if err != nil {
		return err
	}
	PublishToUsers("/notification", []int64{notification.UserId}, map[string]interface{}{"type": notification.NotificationType, "id": notification.Id})
	return nil
}

// notifyNewPost tells everyone who should know about a new post, each once: mentioned, quoted, replied to, the
// participants of a message and the topic's watchers, in that order of priority.
func notifyNewPost(author *User, topic *Topic, post *Post, cooked *CookedPost) {
	targets := map[int64]int{}
	targetData := map[int64]string{}
	order := []int64{}
	add := func(userId int64, notificationType int) {
		if userId == 0 || userId == author.Id {
			return
		}
		if _, ok := targets[userId]; ok {
			return
		}
		targets[userId] = notificationType
		order = append(order, userId)
	}

	for _, user := range getMentionedUsers(cooked.Mentions) {
		add(user.Id, NotificationMentioned)
	}
	for _, group := range getMentionedGroups(author, cooked.GroupMentions) {
		ids, err := GetGroupMemberIds(group.Id)
		if err != nil {
			logs.Error("notifyNewPost(): %v", err)
		}
		if len(ids) > maxGroupMentionNotifications {
			ids = ids[:maxGroupMentionNotifications]
		}
		for _, id := range ids {
			if _, ok := targets[id]; !ok && id != author.Id {
				targetData[id] = toJson(map[string]interface{}{"groupName": group.Name})
			}
			add(id, NotificationGroupMentioned)
		}
	}
	for _, user := range getMentionedUsers(cooked.Quotes) {
		add(user.Id, NotificationQuoted)
	}
	if post.ReplyToUserId != 0 {
		add(post.ReplyToUserId, NotificationReplied)
	}
	if topic.IsPrivateMessage() {
		ids, err := GetTopicAllowedUserIds(topic.Id)
		if err != nil {
			logs.Error("notifyNewPost(): %v", err)
		}
		ids = append(ids, getTopicGroupMemberIds(topic.Id)...)
		for _, id := range ids {
			add(id, NotificationPrivateMessage)
		}
	}
	watchers := []*TopicUser{}
	err := adapter.Engine.Where("topic_id = ? AND notification_level = ?", topic.Id, NotificationLevelWatching).Find(&watchers)
	if err != nil {
		logs.Error("notifyNewPost(): %v", err)
	}
	for _, watcher := range watchers {
		add(watcher.UserId, NotificationPosted)
	}

	muted := map[int64]bool{}
	if len(order) > 0 {
		rows := []*TopicUser{}
		_ = adapter.Engine.Where("topic_id = ? AND notification_level = ?", topic.Id, NotificationLevelMuted).In("user_id", order).Find(&rows)
		for _, row := range rows {
			muted[row.UserId] = true
		}
	}

	for _, userId := range order {
		notificationType := targets[userId]
		if muted[userId] && notificationType != NotificationPrivateMessage {
			continue
		}
		if !canUserSeeTopic(userId, topic) {
			continue
		}
		err = addNotification(&Notification{
			UserId:           userId,
			NotificationType: notificationType,
			TopicId:          topic.Id,
			PostNumber:       post.PostNumber,
			ActingUserId:     author.Id,
			Excerpt:          truncateRunes(cooked.Excerpt, 200),
			IsHighPriority:   notificationType == NotificationPrivateMessage,
			Data:             targetData[userId],
		}, topic)
		if err != nil {
			logs.Error("notifyNewPost(): %v", err)
		}
	}
}

// notifyMentions tells the users newly mentioned in an edited post.
func notifyMentions(author *User, topic *Topic, post *Post, cooked *CookedPost, onlyNew bool) {
	for _, user := range getMentionedUsers(cooked.Mentions) {
		if user.Id == author.Id || !canUserSeeTopic(user.Id, topic) {
			continue
		}
		if onlyNew {
			has, err := adapter.Engine.Where("user_id = ? AND topic_id = ? AND post_number = ?", user.Id, topic.Id, post.PostNumber).Exist(&Notification{})
			if err != nil || has {
				continue
			}
		}
		_ = addNotification(&Notification{
			UserId:           user.Id,
			NotificationType: NotificationMentioned,
			TopicId:          topic.Id,
			PostNumber:       post.PostNumber,
			ActingUserId:     author.Id,
			Excerpt:          truncateRunes(cooked.Excerpt, 200),
		}, topic)
	}
}

func notifyLike(liker *User, topic *Topic, post *Post) {
	has, err := adapter.Engine.Where("user_id = ? AND topic_id = ? AND post_number = ? AND notification_type = ? AND acting_user_id = ?",
		post.UserId, topic.Id, post.PostNumber, NotificationLiked, liker.Id).Exist(&Notification{})
	if err != nil || has {
		return
	}
	_ = addNotification(&Notification{
		UserId:           post.UserId,
		NotificationType: NotificationLiked,
		TopicId:          topic.Id,
		PostNumber:       post.PostNumber,
		ActingUserId:     liker.Id,
	}, topic)
}

func getMentionedUsers(usernames []string) []*User {
	if len(usernames) == 0 {
		return nil
	}
	users, err := GetUsersByUsernames(usernames)
	if err != nil {
		logs.Error("getMentionedUsers(): %v", err)
		return nil
	}
	return users
}

const maxGroupMentionNotifications = 100

// getMentionedGroups are the groups mentioned that the author may mention.
func getMentionedGroups(author *User, names []string) []*Group {
	if len(names) == 0 {
		return nil
	}
	groups, err := GetGroupsByNames(names)
	if err != nil {
		logs.Error("getMentionedGroups(): %v", err)
		return nil
	}
	res := []*Group{}
	for _, group := range groups {
		if CanMentionGroup(author, group) {
			res = append(res, group)
		}
	}
	return res
}

func canUserSeeTopic(userId int64, topic *Topic) bool {
	user, err := GetUser(userId)
	if err != nil || user == nil {
		return false
	}
	canSee, err := CanSeeTopic(user, topic)
	return err == nil && canSee
}

var notificationFilters = map[string][]int{
	"replies":  {NotificationMentioned, NotificationReplied, NotificationQuoted, NotificationPosted, NotificationGroupMentioned},
	"likes":    {NotificationLiked},
	"messages": {NotificationPrivateMessage, NotificationInvitedToPrivateMessage, NotificationAdminMessage, NotificationMembershipRequest, NotificationMembershipAccepted},
	"chat":     {NotificationChatMention, NotificationChatMessage},
	"badges":   {NotificationGrantedBadge},
}

type NotificationList struct {
	Notifications []*Notification `json:"notifications"`
	Users         map[int64]*User `json:"users"`
	More          bool            `json:"more"`
}

func GetNotifications(user *User, filter string, page int, pageSize int) (*NotificationList, error) {
	notifications := []*Notification{}
	session := adapter.Engine.Where("user_id = ?", user.Id)
	if types, ok := notificationFilters[filter]; ok {
		session = session.In("notification_type", types)
	}
	if filter == "unread" {
		session = session.And("is_read = ?", false)
	}
	err := session.Desc("id").Limit(pageSize+1, page*pageSize).Find(&notifications)
	if err != nil {
		return nil, err
	}
	more := len(notifications) > pageSize
	if more {
		notifications = notifications[:pageSize]
	}

	ids := []int64{}
	for _, notification := range notifications {
		ids = append(ids, notification.ActingUserId)
	}
	users, err := GetUserMap(ids)
	if err != nil {
		return nil, err
	}
	return &NotificationList{Notifications: notifications, Users: users, More: more}, nil
}

func CountUnreadNotifications(user *User) (int64, int64, error) {
	if user == nil {
		return 0, 0, nil
	}
	all, err := adapter.Engine.Where("user_id = ? AND is_read = ?", user.Id, false).Count(&Notification{})
	if err != nil {
		return 0, 0, err
	}
	highPriority, err := adapter.Engine.Where("user_id = ? AND is_read = ? AND is_high_priority = ?", user.Id, false, true).Count(&Notification{})
	return all, highPriority, err
}

func MarkNotificationRead(user *User, id int64) error {
	session := adapter.Engine.Where("user_id = ?", user.Id)
	if id != 0 {
		session = session.And("id = ?", id)
	}
	_, err := session.Cols("is_read").Update(&Notification{IsRead: true})
	return err
}

func MarkTopicNotificationsRead(userId int64, topicId int64, upTo int) error {
	_, err := adapter.Engine.Where("user_id = ? AND topic_id = ? AND post_number <= ? AND is_read = ?", userId, topicId, upTo, false).
		Cols("is_read").Update(&Notification{IsRead: true})
	return err
}
