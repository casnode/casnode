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
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/beego/beego/v2/core/logs"
)

// Chat is Discourse's chat: channels tied to a category, readable by whoever may read the category, and direct
// message channels between a few users. Pages get new messages from the message bus.

const (
	ChatChannelCategory = "category"
	ChatChannelDirect   = "direct"

	ChatStatusOpen     = "open"
	ChatStatusReadOnly = "read_only"
	ChatStatusClosed   = "closed"
	ChatStatusArchived = "archived"

	ChatPageSize        = 50
	chatMaxDirectUsers  = 20
	chatMaxReactions    = 30
	chatMentionNotifies = 100
)

type ChatChannel struct {
	Id            int64     `xorm:"pk autoincr" json:"id"`
	Name          string    `xorm:"varchar(100)" json:"name"`
	Slug          string    `xorm:"varchar(100)" json:"slug"`
	Description   string    `xorm:"varchar(500)" json:"description"`
	ChannelType   string    `xorm:"varchar(20) index" json:"channelType"`
	CategoryId    int64     `xorm:"index" json:"categoryId"`
	Status        string    `xorm:"varchar(20)" json:"status"`
	DirectKey     string    `xorm:"varchar(300) index" json:"-"`
	UserCount     int       `json:"userCount"`
	MessageCount  int       `json:"messageCount"`
	LastMessageId int64     `json:"lastMessageId"`
	LastMessageAt time.Time `json:"lastMessageAt"`
	CreatedById   int64     `json:"createdById"`
	CreatedAt     time.Time `xorm:"created" json:"createdAt"`

	Membership  *ChatMembership `xorm:"-" json:"membership,omitempty"`
	UnreadCount int             `xorm:"-" json:"unreadCount"`
	DirectUsers []*User         `xorm:"-" json:"directUsers,omitempty"`
	CanSend     bool            `xorm:"-" json:"canSend"`
	CanModerate bool            `xorm:"-" json:"canModerate"`
}

type ChatMembership struct {
	Id                int64     `xorm:"pk autoincr" json:"id"`
	ChannelId         int64     `xorm:"index unique(channel_user)" json:"channelId"`
	UserId            int64     `xorm:"index unique(channel_user)" json:"userId"`
	Following         bool      `json:"following"`
	Muted             bool      `json:"muted"`
	LastReadMessageId int64     `json:"lastReadMessageId"`
	CreatedAt         time.Time `xorm:"created" json:"createdAt"`
}

type ChatMessage struct {
	Id          int64     `xorm:"pk autoincr" json:"id"`
	ChannelId   int64     `xorm:"index" json:"channelId"`
	UserId      int64     `xorm:"index" json:"userId"`
	Raw         string    `xorm:"mediumtext" json:"raw,omitempty"`
	Cooked      string    `xorm:"mediumtext" json:"cooked"`
	Excerpt     string    `xorm:"varchar(500)" json:"excerpt"`
	InReplyToId int64     `json:"inReplyToId"`
	Edited      bool      `json:"edited"`
	Deleted     bool      `xorm:"index" json:"deleted"`
	DeletedById int64     `json:"deletedById,omitempty"`
	CreatedAt   time.Time `xorm:"created index" json:"createdAt"`
	UpdatedAt   time.Time `xorm:"updated" json:"updatedAt"`

	Reactions []*ChatReactionSummary `xorm:"-" json:"reactions"`
	InReplyTo *ChatMessage           `xorm:"-" json:"inReplyTo,omitempty"`
}

type ChatReaction struct {
	Id        int64     `xorm:"pk autoincr" json:"id"`
	MessageId int64     `xorm:"index unique(message_user_emoji)" json:"messageId"`
	UserId    int64     `xorm:"unique(message_user_emoji)" json:"userId"`
	Emoji     string    `xorm:"varchar(50) unique(message_user_emoji)" json:"emoji"`
	CreatedAt time.Time `xorm:"created" json:"createdAt"`
}

type ChatReactionSummary struct {
	Emoji     string   `json:"emoji"`
	Count     int      `json:"count"`
	Reacted   bool     `json:"reacted"`
	Usernames []string `json:"usernames"`
}

var chatLock sync.Mutex

func IsChatEnabled() bool {
	return GetSiteSettingBool("chat_enabled")
}

func GetChatChannel(id int64) (*ChatChannel, error) {
	channel := &ChatChannel{}
	has, err := adapter.Engine.ID(id).Get(channel)
	if err != nil || !has {
		return nil, err
	}
	return channel, nil
}

func getChatMembership(channelId int64, userId int64) (*ChatMembership, error) {
	membership := &ChatMembership{}
	has, err := adapter.Engine.Where("channel_id = ? AND user_id = ?", channelId, userId).Get(membership)
	if err != nil || !has {
		return nil, err
	}
	return membership, nil
}

func canSeeChannel(user *User, channel *ChatChannel) bool {
	if channel == nil {
		return false
	}
	if channel.ChannelType == ChatChannelDirect {
		if user == nil {
			return false
		}
		membership, err := getChatMembership(channel.Id, user.Id)
		return err == nil && membership != nil
	}
	category, err := GetCategory(channel.CategoryId)
	if err != nil || category == nil {
		return user.IsStaff()
	}
	return category.CanSee(user)
}

func CanSeeChatChannel(user *User, channelId int64) bool {
	if !IsChatEnabled() {
		return false
	}
	channel, err := GetChatChannel(channelId)
	if err != nil {
		return false
	}
	return canSeeChannel(user, channel)
}

func canSendInChannel(user *User, channel *ChatChannel) bool {
	if user == nil || user.IsSuspended() || user.IsSilenced() || !canSeeChannel(user, channel) {
		return false
	}
	if channel.Status != ChatStatusOpen {
		return user.IsStaff() && channel.Status == ChatStatusClosed
	}
	if channel.ChannelType == ChatChannelDirect {
		return true
	}
	category, err := GetCategory(channel.CategoryId)
	if err != nil || category == nil {
		return user.IsStaff()
	}
	return category.CanReply(user)
}

func loadSeeableChannel(user *User, channelId int64) (*ChatChannel, error) {
	if !IsChatEnabled() {
		return nil, fmt.Errorf("Chat is turned off")
	}
	channel, err := GetChatChannel(channelId)
	if err != nil {
		return nil, err
	}
	if !canSeeChannel(user, channel) {
		return nil, fmt.Errorf("The channel does not exist or is private")
	}
	return channel, nil
}

// fillChannelState sets the user's membership, unread count and the other people of direct channels.
func fillChannelState(user *User, channels []*ChatChannel) error {
	if len(channels) == 0 {
		return nil
	}
	ids := []int64{}
	directIds := []int64{}
	for _, channel := range channels {
		ids = append(ids, channel.Id)
		if channel.ChannelType == ChatChannelDirect {
			directIds = append(directIds, channel.Id)
		}
		channel.CanModerate = user.IsStaff() && channel.ChannelType == ChatChannelCategory
		channel.CanSend = canSendInChannel(user, channel)
	}

	if len(directIds) > 0 {
		members := []*ChatMembership{}
		err := adapter.Engine.In("channel_id", directIds).Asc("id").Find(&members)
		if err != nil {
			return err
		}
		userIds := []int64{}
		for _, member := range members {
			userIds = append(userIds, member.UserId)
		}
		users, err := GetUserMap(userIds)
		if err != nil {
			return err
		}
		for _, channel := range channels {
			for _, member := range members {
				if member.ChannelId == channel.Id && users[member.UserId] != nil {
					channel.DirectUsers = append(channel.DirectUsers, users[member.UserId])
				}
			}
		}
	}

	if user == nil {
		return nil
	}
	memberships := []*ChatMembership{}
	err := adapter.Engine.Where("user_id = ?", user.Id).In("channel_id", ids).Find(&memberships)
	if err != nil {
		return err
	}
	for _, channel := range channels {
		for _, membership := range memberships {
			if membership.ChannelId == channel.Id {
				channel.Membership = membership
			}
		}
		if channel.Membership != nil && channel.Membership.Following && channel.LastMessageId > channel.Membership.LastReadMessageId {
			count, err := adapter.Engine.Where("channel_id = ? AND id > ? AND user_id <> ? AND deleted = ?",
				channel.Id, channel.Membership.LastReadMessageId, user.Id, false).Count(&ChatMessage{})
			if err != nil {
				return err
			}
			channel.UnreadCount = int(count)
		}
	}
	return nil
}

type ChatChannelList struct {
	Channels       []*ChatChannel `json:"channels"`
	DirectChannels []*ChatChannel `json:"directChannels"`
}

// GetMyChatChannels are the category channels the user follows and their direct message channels.
func GetMyChatChannels(user *User) (*ChatChannelList, error) {
	res := &ChatChannelList{Channels: []*ChatChannel{}, DirectChannels: []*ChatChannel{}}
	if user == nil || !IsChatEnabled() {
		return res, nil
	}
	channelIds := []int64{}
	err := adapter.Engine.Table("chat_membership").Cols("channel_id").Where("user_id = ? AND following = ?", user.Id, true).Find(&channelIds)
	if err != nil || len(channelIds) == 0 {
		return res, err
	}
	channels := []*ChatChannel{}
	err = adapter.Engine.In("id", channelIds).Find(&channels)
	if err != nil {
		return nil, err
	}
	visible := []*ChatChannel{}
	for _, channel := range channels {
		if channel.Status != ChatStatusArchived && canSeeChannel(user, channel) {
			visible = append(visible, channel)
		}
	}
	err = fillChannelState(user, visible)
	if err != nil {
		return nil, err
	}
	for _, channel := range visible {
		if channel.ChannelType == ChatChannelDirect {
			res.DirectChannels = append(res.DirectChannels, channel)
		} else {
			res.Channels = append(res.Channels, channel)
		}
	}
	sort.SliceStable(res.Channels, func(i, j int) bool {
		return strings.ToLower(res.Channels[i].Name) < strings.ToLower(res.Channels[j].Name)
	})
	sort.SliceStable(res.DirectChannels, func(i, j int) bool {
		return res.DirectChannels[i].LastMessageAt.After(res.DirectChannels[j].LastMessageAt)
	})
	return res, nil
}

// BrowseChatChannels lists the category channels the user may see, with a status filter.
func BrowseChatChannels(user *User, status string, term string) ([]*ChatChannel, error) {
	channels := []*ChatChannel{}
	session := adapter.Engine.Where("channel_type = ?", ChatChannelCategory)
	if status != "" && status != "all" {
		session = session.And("status = ?", status)
	}
	if term != "" {
		session = session.And("LOWER(name) LIKE ?", "%"+escapeLike(strings.ToLower(term))+"%")
	}
	err := session.Asc("name").Find(&channels)
	if err != nil {
		return nil, err
	}
	res := []*ChatChannel{}
	for _, channel := range channels {
		if canSeeChannel(user, channel) {
			res = append(res, channel)
		}
	}
	err = fillChannelState(user, res)
	return res, err
}

func GetChatChannelForUser(user *User, id int64) (*ChatChannel, error) {
	channel, err := loadSeeableChannel(user, id)
	if err != nil {
		return nil, err
	}
	err = fillChannelState(user, []*ChatChannel{channel})
	return channel, err
}

type ChatChannelInput struct {
	Id          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CategoryId  int64  `json:"categoryId"`
	Status      string `json:"status"`
}

func CreateChatChannel(user *User, input *ChatChannelInput) (*ChatChannel, error) {
	if !user.IsStaff() {
		return nil, fmt.Errorf("Only staff can create channels")
	}
	category, err := GetCategory(input.CategoryId)
	if err != nil {
		return nil, err
	}
	if category == nil {
		return nil, fmt.Errorf("Choose a category")
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		name = category.Name
	}
	channel := &ChatChannel{
		Name:        truncateRunes(name, 90),
		Slug:        Slugify(name),
		Description: truncateRunes(strings.TrimSpace(input.Description), 480),
		ChannelType: ChatChannelCategory,
		CategoryId:  category.Id,
		Status:      ChatStatusOpen,
		CreatedById: user.Id,
	}
	_, err = adapter.Engine.Insert(channel)
	if err != nil {
		return nil, err
	}
	err = FollowChatChannel(user, channel.Id, true)
	return channel, err
}

func UpdateChatChannel(user *User, input *ChatChannelInput) (*ChatChannel, error) {
	channel, err := loadSeeableChannel(user, input.Id)
	if err != nil {
		return nil, err
	}
	if !user.IsStaff() || channel.ChannelType != ChatChannelCategory {
		return nil, fmt.Errorf("Only staff can change channels")
	}
	if name := strings.TrimSpace(input.Name); name != "" {
		channel.Name = truncateRunes(name, 90)
		channel.Slug = Slugify(name)
	}
	channel.Description = truncateRunes(strings.TrimSpace(input.Description), 480)
	switch input.Status {
	case ChatStatusOpen, ChatStatusReadOnly, ChatStatusClosed, ChatStatusArchived:
		channel.Status = input.Status
	}
	if input.CategoryId != 0 {
		category, err := GetCategory(input.CategoryId)
		if err != nil {
			return nil, err
		}
		if category != nil {
			channel.CategoryId = category.Id
		}
	}
	_, err = adapter.Engine.ID(channel.Id).Cols("name", "slug", "description", "status", "category_id").Update(channel)
	if err != nil {
		return nil, err
	}
	PublishToChatChannel(channel.Id, map[string]interface{}{"type": "channel", "channelId": channel.Id})
	return channel, nil
}

func DeleteChatChannel(user *User, id int64) error {
	if !user.Admin {
		return fmt.Errorf("Only admins can delete channels")
	}
	channel, err := GetChatChannel(id)
	if err != nil {
		return err
	}
	if channel == nil {
		return fmt.Errorf("The channel does not exist")
	}
	messageIds := []int64{}
	err = adapter.Engine.Table("chat_message").Cols("id").Where("channel_id = ?", id).Find(&messageIds)
	if err != nil {
		return err
	}
	for start := 0; start < len(messageIds); start += 500 {
		end := start + 500
		if end > len(messageIds) {
			end = len(messageIds)
		}
		_, err = adapter.Engine.In("message_id", messageIds[start:end]).Delete(&ChatReaction{})
		if err != nil {
			return err
		}
	}
	for _, bean := range []interface{}{&ChatMessage{}, &ChatMembership{}} {
		_, err = adapter.Engine.Where("channel_id = ?", id).Delete(bean)
		if err != nil {
			return err
		}
	}
	_, err = adapter.Engine.ID(id).Delete(&ChatChannel{})
	return err
}

func refreshChannelUserCount(channelId int64) {
	count, err := adapter.Engine.Where("channel_id = ? AND following = ?", channelId, true).Count(&ChatMembership{})
	if err == nil {
		_, _ = adapter.Engine.ID(channelId).Cols("user_count").Update(&ChatChannel{UserCount: int(count)})
	}
}

// FollowChatChannel joins or leaves a channel; leaving a direct channel only hides it until the next message.
func FollowChatChannel(user *User, channelId int64, follow bool) error {
	channel, err := loadSeeableChannel(user, channelId)
	if err != nil {
		return err
	}
	if user == nil {
		return fmt.Errorf("Please sign in first")
	}
	chatLock.Lock()
	membership, err := getChatMembership(channel.Id, user.Id)
	if err == nil {
		if membership == nil {
			_, err = adapter.Engine.Insert(&ChatMembership{ChannelId: channel.Id, UserId: user.Id, Following: follow, LastReadMessageId: channel.LastMessageId})
		} else if membership.Following != follow {
			_, err = adapter.Engine.ID(membership.Id).Cols("following").Update(&ChatMembership{Following: follow})
		}
	}
	chatLock.Unlock()
	if err != nil {
		return err
	}
	refreshChannelUserCount(channel.Id)
	PublishToUsers("/chat-user", []int64{user.Id}, map[string]interface{}{"type": "channels"})
	return nil
}

// GetOrCreateDirectChannel is the direct message channel between the user and the others named, made on first use.
func GetOrCreateDirectChannel(user *User, usernames []string) (*ChatChannel, error) {
	if !IsChatEnabled() {
		return nil, fmt.Errorf("Chat is turned off")
	}
	if user == nil || user.IsSuspended() || user.IsSilenced() {
		return nil, fmt.Errorf("You can't send messages")
	}
	others, err := GetUsersByUsernames(usernames)
	if err != nil {
		return nil, err
	}
	ids := []int64{user.Id}
	for _, other := range others {
		if other.Username != SystemUsername {
			ids = append(ids, other.Id)
		}
	}
	ids = uniqueIds(ids)
	if len(ids) > chatMaxDirectUsers {
		return nil, fmt.Errorf("A direct message can have at most %d people", chatMaxDirectUsers)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	parts := []string{}
	for _, id := range ids {
		parts = append(parts, itoa(id))
	}
	key := strings.Join(parts, ",")

	chatLock.Lock()
	defer chatLock.Unlock()
	channel := &ChatChannel{}
	has, err := adapter.Engine.Where("channel_type = ? AND direct_key = ?", ChatChannelDirect, key).Get(channel)
	if err != nil {
		return nil, err
	}
	if !has {
		channel = &ChatChannel{ChannelType: ChatChannelDirect, DirectKey: key, Status: ChatStatusOpen, CreatedById: user.Id, UserCount: len(ids)}
		_, err = adapter.Engine.Insert(channel)
		if err != nil {
			return nil, err
		}
		for _, id := range ids {
			_, err = adapter.Engine.Insert(&ChatMembership{ChannelId: channel.Id, UserId: id, Following: id == user.Id})
			if err != nil {
				return nil, err
			}
		}
	} else {
		_, err = adapter.Engine.Where("channel_id = ? AND user_id = ?", channel.Id, user.Id).Cols("following").Update(&ChatMembership{Following: true})
		if err != nil {
			return nil, err
		}
	}
	return channel, nil
}

type ChatMessageList struct {
	Messages   []*ChatMessage  `json:"messages"`
	Users      map[int64]*User `json:"users"`
	MorePast   bool            `json:"morePast"`
	MoreFuture bool            `json:"moreFuture"`
}

// GetChatMessages pages through a channel: before an id, after an id, around an id, or the latest messages.
func GetChatMessages(user *User, channelId int64, before int64, after int64, around int64) (*ChatMessageList, error) {
	channel, err := loadSeeableChannel(user, channelId)
	if err != nil {
		return nil, err
	}
	base := func() *ChatMessageList { return &ChatMessageList{Messages: []*ChatMessage{}} }
	query := func(op string, id int64, asc bool, limit int) ([]*ChatMessage, error) {
		messages := []*ChatMessage{}
		session := adapter.Engine.Where("channel_id = ?", channel.Id).Omit("raw")
		if !user.IsStaff() {
			session = session.And("deleted = ?", false)
		}
		if op != "" {
			session = session.And("id "+op+" ?", id)
		}
		if asc {
			session = session.Asc("id")
		} else {
			session = session.Desc("id")
		}
		err := session.Limit(limit).Find(&messages)
		if !asc {
			for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
				messages[i], messages[j] = messages[j], messages[i]
			}
		}
		return messages, err
	}

	res := base()
	switch {
	case after > 0:
		res.Messages, err = query(">", after, true, ChatPageSize+1)
		if err == nil && len(res.Messages) > ChatPageSize {
			res.Messages = res.Messages[:ChatPageSize]
			res.MoreFuture = true
		}
		res.MorePast = true
	case before > 0:
		res.Messages, err = query("<", before, false, ChatPageSize+1)
		if err == nil && len(res.Messages) > ChatPageSize {
			res.Messages = res.Messages[1:]
			res.MorePast = true
		}
		res.MoreFuture = true
	case around > 0:
		past, err1 := query("<", around, false, ChatPageSize/2+1)
		future, err2 := query(">=", around, true, ChatPageSize/2+1)
		if err1 != nil {
			return nil, err1
		}
		if err2 != nil {
			return nil, err2
		}
		if len(past) > ChatPageSize/2 {
			past = past[1:]
			res.MorePast = true
		}
		if len(future) > ChatPageSize/2 {
			future = future[:ChatPageSize/2]
			res.MoreFuture = true
		}
		res.Messages = append(past, future...)
	default:
		res.Messages, err = query("", 0, false, ChatPageSize+1)
		if err == nil && len(res.Messages) > ChatPageSize {
			res.Messages = res.Messages[1:]
			res.MorePast = true
		}
	}
	if err != nil {
		return nil, err
	}
	res.Users, err = fillChatMessages(user, res.Messages)
	return res, err
}

// fillChatMessages adds the reactions and the messages replied to, and returns the users they mention.
func fillChatMessages(user *User, messages []*ChatMessage) (map[int64]*User, error) {
	ids := []int64{}
	replyIds := []int64{}
	userIds := []int64{}
	for _, message := range messages {
		ids = append(ids, message.Id)
		userIds = append(userIds, message.UserId)
		if message.InReplyToId != 0 {
			replyIds = append(replyIds, message.InReplyToId)
		}
		message.Reactions = []*ChatReactionSummary{}
		if message.Deleted {
			message.Cooked = ""
		}
	}
	if len(ids) > 0 {
		reactions := []*ChatReaction{}
		err := adapter.Engine.In("message_id", ids).Asc("id").Find(&reactions)
		if err != nil {
			return nil, err
		}
		for _, reaction := range reactions {
			userIds = append(userIds, reaction.UserId)
		}
		reactionUsers, err := GetUserMap(userIds)
		if err != nil {
			return nil, err
		}
		for _, message := range messages {
			message.Reactions = summarizeReactions(user, message.Id, reactions, reactionUsers)
		}
	}
	if len(replyIds) > 0 {
		parents := []*ChatMessage{}
		err := adapter.Engine.In("id", uniqueIds(replyIds)).Cols("id", "user_id", "excerpt", "deleted", "channel_id").Find(&parents)
		if err != nil {
			return nil, err
		}
		for _, message := range messages {
			for _, parent := range parents {
				if parent.Id == message.InReplyToId && parent.ChannelId == message.ChannelId {
					if parent.Deleted {
						parent.Excerpt = ""
					}
					message.InReplyTo = parent
					userIds = append(userIds, parent.UserId)
				}
			}
		}
	}
	return GetUserMap(userIds)
}

func summarizeReactions(user *User, messageId int64, reactions []*ChatReaction, users map[int64]*User) []*ChatReactionSummary {
	res := []*ChatReactionSummary{}
	byEmoji := map[string]*ChatReactionSummary{}
	for _, reaction := range reactions {
		if reaction.MessageId != messageId {
			continue
		}
		summary := byEmoji[reaction.Emoji]
		if summary == nil {
			summary = &ChatReactionSummary{Emoji: reaction.Emoji, Usernames: []string{}}
			byEmoji[reaction.Emoji] = summary
			res = append(res, summary)
		}
		summary.Count++
		if user != nil && reaction.UserId == user.Id {
			summary.Reacted = true
		}
		if u := users[reaction.UserId]; u != nil && len(summary.Usernames) < 10 {
			summary.Usernames = append(summary.Usernames, u.Username)
		}
	}
	return res
}

func getChatMessage(id int64) (*ChatMessage, error) {
	message := &ChatMessage{}
	has, err := adapter.Engine.ID(id).Get(message)
	if err != nil || !has {
		return nil, err
	}
	return message, nil
}

func validateChatRaw(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("The message is empty")
	}
	max := GetSiteSettingInt("chat_max_message_length")
	if max > 0 && utf8.RuneCountInString(raw) > max {
		return "", fmt.Errorf("The message can't be more than %d characters", max)
	}
	return raw, nil
}

func publishChatMessage(kind string, message *ChatMessage) {
	users, err := fillChatMessages(nil, []*ChatMessage{message})
	if err != nil {
		logs.Error("publishChatMessage(): %v", err)
	}
	copied := *message
	copied.Raw = ""
	PublishToChatChannel(message.ChannelId, map[string]interface{}{"type": kind, "message": &copied, "users": users})
}

// SendChatMessage posts a message, joining the channel on the way, and tells the people mentioned.
func SendChatMessage(user *User, channelId int64, raw string, inReplyToId int64) (*ChatMessage, error) {
	channel, err := loadSeeableChannel(user, channelId)
	if err != nil {
		return nil, err
	}
	if !canSendInChannel(user, channel) {
		return nil, fmt.Errorf("You can't send messages in this channel")
	}
	raw, err = validateChatRaw(raw)
	if err != nil {
		return nil, err
	}
	if inReplyToId != 0 {
		parent, err := getChatMessage(inReplyToId)
		if err != nil {
			return nil, err
		}
		if parent == nil || parent.ChannelId != channel.Id {
			inReplyToId = 0
		}
	}

	cooked := CookMarkdown(raw)
	message := &ChatMessage{
		ChannelId:   channel.Id,
		UserId:      user.Id,
		Raw:         raw,
		Cooked:      cooked.Html,
		Excerpt:     truncateRunes(cooked.Excerpt, 200),
		InReplyToId: inReplyToId,
	}
	_, err = adapter.Engine.Insert(message)
	if err != nil {
		return nil, err
	}
	channel.LastMessageId = message.Id
	channel.LastMessageAt = message.CreatedAt
	_, err = adapter.Engine.ID(channel.Id).Incr("message_count").Cols("last_message_id", "last_message_at").Update(channel)
	if err != nil {
		return nil, err
	}

	chatLock.Lock()
	membership, err := getChatMembership(channel.Id, user.Id)
	if err == nil {
		if membership == nil {
			_, err = adapter.Engine.Insert(&ChatMembership{ChannelId: channel.Id, UserId: user.Id, Following: true, LastReadMessageId: message.Id})
		} else {
			_, err = adapter.Engine.ID(membership.Id).Cols("following", "last_read_message_id").Update(&ChatMembership{Following: true, LastReadMessageId: message.Id})
		}
	}
	chatLock.Unlock()
	if err != nil {
		return nil, err
	}
	if membership == nil || !membership.Following {
		refreshChannelUserCount(channel.Id)
		PublishToUsers("/chat-user", []int64{user.Id}, map[string]interface{}{"type": "channels"})
	}

	if channel.ChannelType == ChatChannelDirect {
		// a new message shows the conversation again to those who closed it
		others := []*ChatMembership{}
		err = adapter.Engine.Where("channel_id = ? AND user_id <> ?", channel.Id, user.Id).Find(&others)
		if err == nil {
			hidden := []int64{}
			for _, other := range others {
				if !other.Following {
					hidden = append(hidden, other.UserId)
				}
			}
			if len(hidden) > 0 {
				_, _ = adapter.Engine.Where("channel_id = ?", channel.Id).In("user_id", hidden).Cols("following").Update(&ChatMembership{Following: true})
				PublishToUsers("/chat-user", hidden, map[string]interface{}{"type": "channels"})
			}
		}
	}

	publishChatMessage("sent", message)
	notifyChatMessage(user, channel, message, cooked)
	TriggerBadgeGrant()
	message.Raw = ""
	return message, nil
}

// notifyChatMessage tells the people mentioned, and the other people of a direct channel (one unread notification
// per channel each).
func notifyChatMessage(author *User, channel *ChatChannel, message *ChatMessage, cooked *CookedPost) {
	notified := map[int64]bool{author.Id: true}
	data := toJson(map[string]interface{}{"chatChannelId": channel.Id, "chatMessageId": message.Id, "chatChannelName": channel.Name, "chatChannelSlug": channel.Slug})
	notify := func(userId int64, notificationType int) {
		if notified[userId] {
			return
		}
		notified[userId] = true
		if notificationType == NotificationChatMessage {
			existing := &Notification{}
			has, err := adapter.Engine.Where("user_id = ? AND notification_type = ? AND is_read = ? AND topic_title = ?",
				userId, NotificationChatMessage, false, chatChannelKey(channel.Id)).Get(existing)
			if err == nil && has {
				_, _ = adapter.Engine.ID(existing.Id).Cols("excerpt", "acting_user_id", "data").
					Update(&Notification{Excerpt: message.Excerpt, ActingUserId: author.Id, Data: data})
				PublishToUsers("/notification", []int64{userId}, map[string]interface{}{"type": notificationType, "id": existing.Id})
				return
			}
		}
		member, err := GetUser(userId)
		if err != nil || member == nil || !canSeeChannel(member, channel) {
			return
		}
		err = addNotification(&Notification{
			UserId:           userId,
			NotificationType: notificationType,
			ActingUserId:     author.Id,
			TopicTitle:       chatChannelKey(channel.Id),
			Excerpt:          message.Excerpt,
			IsHighPriority:   notificationType == NotificationChatMessage,
			Data:             data,
		}, nil)
		if err != nil {
			logs.Error("notifyChatMessage(): %v", err)
		}
	}

	for _, user := range getMentionedUsers(cooked.Mentions) {
		notify(user.Id, NotificationChatMention)
	}
	for _, group := range getMentionedGroups(author, cooked.GroupMentions) {
		ids, err := GetGroupMemberIds(group.Id)
		if err != nil {
			continue
		}
		if len(ids) > chatMentionNotifies {
			ids = ids[:chatMentionNotifies]
		}
		for _, id := range ids {
			notify(id, NotificationChatMention)
		}
	}
	if channel.ChannelType == ChatChannelDirect {
		members := []*ChatMembership{}
		err := adapter.Engine.Where("channel_id = ? AND muted = ?", channel.Id, false).Find(&members)
		if err != nil {
			return
		}
		for _, member := range members {
			notify(member.UserId, NotificationChatMessage)
		}
	}
}

// chatChannelKey names the channel in a notification, so the unread one of a direct channel can be found again.
func chatChannelKey(channelId int64) string {
	return "chat:" + itoa(channelId)
}

func loadEditableMessage(user *User, id int64) (*ChatMessage, *ChatChannel, error) {
	message, err := getChatMessage(id)
	if err != nil {
		return nil, nil, err
	}
	if message == nil {
		return nil, nil, fmt.Errorf("The message does not exist")
	}
	channel, err := loadSeeableChannel(user, message.ChannelId)
	if err != nil {
		return nil, nil, err
	}
	return message, channel, nil
}

func EditChatMessage(user *User, id int64, raw string) (*ChatMessage, error) {
	message, channel, err := loadEditableMessage(user, id)
	if err != nil {
		return nil, err
	}
	if message.UserId != user.Id || message.Deleted || !canSendInChannel(user, channel) {
		return nil, fmt.Errorf("You can't edit this message")
	}
	raw, err = validateChatRaw(raw)
	if err != nil {
		return nil, err
	}
	if raw == message.Raw {
		return message, nil
	}
	cooked := CookMarkdown(raw)
	message.Raw = raw
	message.Cooked = cooked.Html
	message.Excerpt = truncateRunes(cooked.Excerpt, 200)
	message.Edited = true
	_, err = adapter.Engine.ID(message.Id).Cols("raw", "cooked", "excerpt", "edited").Update(message)
	if err != nil {
		return nil, err
	}
	publishChatMessage("edit", message)
	message.Raw = ""
	return message, nil
}

// GetChatMessageRaw is a message's Markdown, for editing or quoting it.
func GetChatMessageRaw(user *User, id int64) (*ChatMessage, error) {
	message, _, err := loadEditableMessage(user, id)
	if err != nil {
		return nil, err
	}
	if message.Deleted && !user.IsStaff() {
		return nil, fmt.Errorf("The message does not exist")
	}
	return message, nil
}

func SetChatMessageDeleted(user *User, id int64, deleted bool) error {
	message, channel, err := loadEditableMessage(user, id)
	if err != nil {
		return err
	}
	staff := user.IsStaff()
	if !(staff || (message.UserId == user.Id && deleted && channel.Status == ChatStatusOpen)) {
		return fmt.Errorf("You can't delete this message")
	}
	message.Deleted = deleted
	message.DeletedById = 0
	if deleted {
		message.DeletedById = user.Id
	}
	_, err = adapter.Engine.ID(message.Id).Cols("deleted", "deleted_by_id").Update(message)
	if err != nil {
		return err
	}
	if deleted {
		PublishToChatChannel(channel.Id, map[string]interface{}{"type": "delete", "messageId": message.Id})
	} else {
		publishChatMessage("restore", message)
	}
	return nil
}

func ReactChatMessage(user *User, id int64, emoji string, add bool) ([]*ChatReactionSummary, error) {
	message, channel, err := loadEditableMessage(user, id)
	if err != nil {
		return nil, err
	}
	if message.Deleted || !canSendInChannel(user, channel) {
		return nil, fmt.Errorf("You can't react to this message")
	}
	emoji = strings.TrimSpace(emoji)
	if emoji == "" || utf8.RuneCountInString(emoji) > 20 {
		return nil, fmt.Errorf("Invalid emoji")
	}
	if add {
		count, err := adapter.Engine.Where("message_id = ?", message.Id).Count(&ChatReaction{})
		if err != nil {
			return nil, err
		}
		has, err := adapter.Engine.Where("message_id = ? AND user_id = ? AND emoji = ?", message.Id, user.Id, emoji).Exist(&ChatReaction{})
		if err != nil {
			return nil, err
		}
		if !has {
			if count >= chatMaxReactions*5 {
				return nil, fmt.Errorf("Too many reactions")
			}
			_, err = adapter.Engine.Insert(&ChatReaction{MessageId: message.Id, UserId: user.Id, Emoji: emoji})
			if err != nil {
				return nil, err
			}
		}
	} else {
		_, err = adapter.Engine.Where("message_id = ? AND user_id = ? AND emoji = ?", message.Id, user.Id, emoji).Delete(&ChatReaction{})
		if err != nil {
			return nil, err
		}
	}
	reactions := []*ChatReaction{}
	err = adapter.Engine.Where("message_id = ?", message.Id).Asc("id").Find(&reactions)
	if err != nil {
		return nil, err
	}
	userIds := []int64{}
	for _, reaction := range reactions {
		userIds = append(userIds, reaction.UserId)
	}
	users, err := GetUserMap(userIds)
	if err != nil {
		return nil, err
	}
	// the reacted flags are the user's; the others work them out from the usernames
	PublishToChatChannel(channel.Id, map[string]interface{}{"type": "reaction", "messageId": message.Id, "reactions": summarizeReactions(nil, message.Id, reactions, users)})
	return summarizeReactions(user, message.Id, reactions, users), nil
}

// MarkChatRead moves the user's read position in a channel forward.
func MarkChatRead(user *User, channelId int64, messageId int64) error {
	channel, err := loadSeeableChannel(user, channelId)
	if err != nil {
		return err
	}
	if messageId > channel.LastMessageId || messageId <= 0 {
		messageId = channel.LastMessageId
	}
	membership, err := getChatMembership(channel.Id, user.Id)
	if err != nil || membership == nil || membership.LastReadMessageId >= messageId {
		return err
	}
	_, err = adapter.Engine.ID(membership.Id).Cols("last_read_message_id").Update(&ChatMembership{LastReadMessageId: messageId})
	if err != nil {
		return err
	}
	_, err = adapter.Engine.Where("user_id = ? AND is_read = ? AND topic_title = ?", user.Id, false, chatChannelKey(channel.Id)).
		Cols("is_read").Update(&Notification{IsRead: true})
	if err != nil {
		return err
	}
	PublishToUsers("/chat-user", []int64{user.Id}, map[string]interface{}{"type": "read", "channelId": channel.Id, "messageId": messageId})
	PublishToUsers("/notification", []int64{user.Id}, map[string]interface{}{"type": "read"})
	return nil
}

func CountUnreadChat(user *User) (int64, int64, error) {
	list, err := GetMyChatChannels(user)
	if err != nil {
		return 0, 0, err
	}
	channels, direct := int64(0), int64(0)
	for _, channel := range list.Channels {
		if channel.UnreadCount > 0 && (channel.Membership == nil || !channel.Membership.Muted) {
			channels++
		}
	}
	for _, channel := range list.DirectChannels {
		direct += int64(channel.UnreadCount)
	}
	return channels, direct, nil
}

// EnsureDefaultChatChannel gives a forum with no channels a General channel, like Discourse.
func EnsureDefaultChatChannel() error {
	count, err := adapter.Engine.Count(&ChatChannel{})
	if err != nil || count > 0 {
		return err
	}
	category, err := GetCategoryBySlug("general")
	if err != nil {
		return err
	}
	if category == nil {
		category, err = GetCategory(GetUncategorizedId())
		if err != nil || category == nil {
			return err
		}
	}
	_, err = adapter.Engine.Insert(&ChatChannel{Name: "General", Slug: "general", ChannelType: ChatChannelCategory, CategoryId: category.Id, Status: ChatStatusOpen})
	return err
}

// GetChatChannelsOfCategory are the channels of a category, for its page.
func GetChatChannelsOfCategory(user *User, categoryId int64) ([]*ChatChannel, error) {
	channels := []*ChatChannel{}
	err := adapter.Engine.Where("channel_type = ? AND category_id = ? AND status <> ?", ChatChannelCategory, categoryId, ChatStatusArchived).Find(&channels)
	if err != nil {
		return nil, err
	}
	res := []*ChatChannel{}
	for _, channel := range channels {
		if canSeeChannel(user, channel) {
			res = append(res, channel)
		}
	}
	return res, nil
}
