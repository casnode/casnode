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
	"sync"
	"time"
)

// The notification levels of a topic, as in Discourse.
const (
	NotificationLevelMuted    = 0
	NotificationLevelRegular  = 1
	NotificationLevelTracking = 2
	NotificationLevelWatching = 3
)

// TopicUser is what a user has read of a topic and how closely they follow it.
type TopicUser struct {
	Id                 int64     `xorm:"pk autoincr" json:"id"`
	UserId             int64     `xorm:"index unique(user_topic)" json:"userId"`
	TopicId            int64     `xorm:"index unique(user_topic)" json:"topicId"`
	NotificationLevel  int       `json:"notificationLevel"`
	LastReadPostNumber int       `json:"lastReadPostNumber"`
	Posted             bool      `json:"posted"`
	FirstVisitedAt     time.Time `json:"firstVisitedAt"`
	LastVisitedAt      time.Time `json:"lastVisitedAt"`
}

// TopicAllowedUser is a participant of a personal message.
type TopicAllowedUser struct {
	Id      int64 `xorm:"pk autoincr" json:"id"`
	TopicId int64 `xorm:"index unique(topic_user)" json:"topicId"`
	UserId  int64 `xorm:"index unique(topic_user)" json:"userId"`
}

var topicUserLock sync.Mutex

func GetTopicUser(userId int64, topicId int64) (*TopicUser, error) {
	tu := &TopicUser{}
	has, err := adapter.Engine.Where("user_id = ? AND topic_id = ?", userId, topicId).Get(tu)
	if err != nil || !has {
		return nil, err
	}
	return tu, nil
}

// SetTopicUser changes the user's row for the topic, creating it first when missing.
func SetTopicUser(userId int64, topicId int64, change func(tu *TopicUser)) error {
	topicUserLock.Lock()
	defer topicUserLock.Unlock()

	tu, err := GetTopicUser(userId, topicId)
	if err != nil {
		return err
	}
	if tu == nil {
		tu = &TopicUser{UserId: userId, TopicId: topicId, NotificationLevel: NotificationLevelRegular}
		change(tu)
		_, err = adapter.Engine.Insert(tu)
		return err
	}
	change(tu)
	_, err = adapter.Engine.ID(tu.Id).AllCols().Update(tu)
	return err
}

// RecordTopicTiming moves the user's read position forward; reading 3 or more posts starts tracking the topic.
func RecordTopicTiming(user *User, topic *Topic, lastRead int) error {
	if lastRead > topic.HighestPostNumber {
		lastRead = topic.HighestPostNumber
	}
	newlyRead := 0
	err := SetTopicUser(user.Id, topic.Id, func(tu *TopicUser) {
		if lastRead <= tu.LastReadPostNumber {
			return
		}
		newlyRead = lastRead - tu.LastReadPostNumber
		tu.LastReadPostNumber = lastRead
		if tu.NotificationLevel == NotificationLevelRegular && lastRead >= 3 {
			tu.NotificationLevel = NotificationLevelTracking
		}
		if tu.FirstVisitedAt.IsZero() || tu.FirstVisitedAt.Year() <= 1 {
			tu.FirstVisitedAt = time.Now()
		}
		tu.LastVisitedAt = time.Now()
	})
	if err != nil {
		return err
	}
	if newlyRead > 0 {
		incrUser(user.Id, "posts_read_count", newlyRead)
		_ = MarkTopicNotificationsRead(user.Id, topic.Id, lastRead)
	}
	return nil
}

func SetTopicNotificationLevel(user *User, topicId int64, level int) error {
	if level < NotificationLevelMuted || level > NotificationLevelWatching {
		level = NotificationLevelRegular
	}
	return SetTopicUser(user.Id, topicId, func(tu *TopicUser) {
		tu.NotificationLevel = level
	})
}

// DismissTopics marks the topics read, for the "Dismiss" button of New and Unread.
func DismissTopics(user *User, filter string, categoryId int64) error {
	list, err := GetTopicList(user, &TopicListQuery{Filter: filter, CategoryId: categoryId, PageSize: 500})
	if err != nil {
		return err
	}
	for _, topic := range list.Topics {
		highest := topic.HighestPostNumber
		err = SetTopicUser(user.Id, topic.Id, func(tu *TopicUser) {
			tu.LastReadPostNumber = highest
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func IsTopicAllowedUser(topicId int64, userId int64) (bool, error) {
	return adapter.Engine.Where("topic_id = ? AND user_id = ?", topicId, userId).Exist(&TopicAllowedUser{})
}

func GetTopicAllowedUserIds(topicId int64) ([]int64, error) {
	rows := []*TopicAllowedUser{}
	err := adapter.Engine.Where("topic_id = ?", topicId).Asc("id").Find(&rows)
	ids := []int64{}
	for _, row := range rows {
		ids = append(ids, row.UserId)
	}
	return ids, err
}

func AddTopicAllowedUser(user *User, topic *Topic, username string) error {
	target, err := GetUserByUsername(username)
	if err != nil {
		return err
	}
	if target == nil {
		return errorf("The user %s does not exist", username)
	}
	has, err := IsTopicAllowedUser(topic.Id, target.Id)
	if err != nil || has {
		return err
	}
	_, err = adapter.Engine.Insert(&TopicAllowedUser{TopicId: topic.Id, UserId: target.Id})
	if err != nil {
		return err
	}
	return addNotification(&Notification{
		UserId:           target.Id,
		NotificationType: NotificationInvitedToPrivateMessage,
		TopicId:          topic.Id,
		PostNumber:       1,
		ActingUserId:     user.Id,
		IsHighPriority:   true,
	}, topic)
}

func CountUnreadPrivateMessages(user *User) (int64, error) {
	if user == nil {
		return 0, nil
	}
	return adapter.Engine.Table("topic").
		Join("INNER", "topic_allowed_user", "topic_allowed_user.topic_id = topic.id AND topic_allowed_user.user_id = ?", user.Id).
		Join("LEFT", "topic_user", "topic_user.topic_id = topic.id AND topic_user.user_id = ?", user.Id).
		Where("topic.archetype = ? AND topic.deleted = ?", ArchetypePrivateMessage, false).
		And("(topic_user.id IS NULL OR topic.highest_post_number > topic_user.last_read_post_number)").Count()
}
