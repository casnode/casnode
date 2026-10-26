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
	"context"
	"strings"
	"sync"
	"time"
)

// The message bus tells open pages what changed, like Discourse's MessageBus: pages long-poll with the id of the last
// message they got, and get the newer messages of the channels they listen to that their user may see. Messages are
// kept in memory for a few minutes; a page that was away longer reloads what it shows.

type BusMessage struct {
	Id      int64       `json:"id"`
	Channel string      `json:"channel"`
	Data    interface{} `json:"data"`

	at            time.Time
	userIds       []int64
	topic         *Topic
	chatChannelId int64
	staffOnly     bool
	allowed       map[int64]bool
	allowedLock   sync.Mutex
}

type messageBus struct {
	lock     sync.Mutex
	lastId   int64
	messages []*BusMessage
	waiters  map[chan struct{}]bool
}

const (
	busKeepCount    = 2000
	busKeepDuration = 10 * time.Minute
)

// ids start from the time the server started, so a page that polled the previous server gets everything kept since
var bus = &messageBus{lastId: time.Now().UnixMilli(), waiters: map[chan struct{}]bool{}}

func (b *messageBus) publish(message *BusMessage) {
	b.lock.Lock()
	b.lastId++
	message.Id = b.lastId
	message.at = time.Now()
	b.messages = append(b.messages, message)
	cut := 0
	for cut < len(b.messages) && (len(b.messages)-cut > busKeepCount || time.Since(b.messages[cut].at) > busKeepDuration) {
		cut++
	}
	b.messages = b.messages[cut:]
	waiters := b.waiters
	b.waiters = map[chan struct{}]bool{}
	b.lock.Unlock()

	for waiter := range waiters {
		close(waiter)
	}
}

// PublishToTopic tells the readers of a topic, and only those who may see it.
func PublishToTopic(topic *Topic, data map[string]interface{}) {
	if topic == nil {
		return
	}
	bus.publish(&BusMessage{Channel: "/topic/" + itoa(topic.Id), Data: data, topic: topic})
}

// PublishLatest tells the topic lists that a topic is new or has a new post.
func PublishLatest(topic *Topic, kind string) {
	data := map[string]interface{}{"type": kind, "topicId": topic.Id, "categoryId": topic.CategoryId, "archetype": topic.Archetype, "tags": topic.Tags}
	bus.publish(&BusMessage{Channel: "/latest", Data: data, topic: topic})
}

// PublishToUsers is for what only some users may know: their notification counts, their chat channels.
func PublishToUsers(channel string, userIds []int64, data interface{}) {
	if len(userIds) == 0 {
		return
	}
	bus.publish(&BusMessage{Channel: channel, Data: data, userIds: userIds})
}

func PublishToStaff(channel string, data interface{}) {
	bus.publish(&BusMessage{Channel: channel, Data: data, staffOnly: true})
}

func PublishToChatChannel(channelId int64, data interface{}) {
	bus.publish(&BusMessage{Channel: "/chat/" + itoa(channelId), Data: data, chatChannelId: channelId})
}

func (m *BusMessage) canSee(user *User) bool {
	if m.staffOnly && !user.IsStaff() {
		return false
	}
	if len(m.userIds) > 0 {
		return user != nil && containsId(m.userIds, user.Id)
	}
	if m.topic == nil && m.chatChannelId == 0 {
		return true
	}

	key := int64(0)
	if user != nil {
		key = user.Id
	}
	m.allowedLock.Lock()
	defer m.allowedLock.Unlock()
	if allowed, ok := m.allowed[key]; ok {
		return allowed
	}
	allowed := false
	if m.topic != nil {
		allowed, _ = CanSeeTopic(user, m.topic)
	} else {
		allowed = CanSeeChatChannel(user, m.chatChannelId)
	}
	if m.allowed == nil {
		m.allowed = map[int64]bool{}
	}
	m.allowed[key] = allowed
	return allowed
}

// matchesChannel: a subscription ending with "/" takes every channel under it, like "/chat/".
func matchesChannel(subscriptions []string, channel string) bool {
	for _, subscription := range subscriptions {
		if subscription == channel || (strings.HasSuffix(subscription, "/") && strings.HasPrefix(channel, subscription)) {
			return true
		}
	}
	return false
}

type BusPoll struct {
	LastId   int64         `json:"lastId"`
	Messages []*BusMessage `json:"messages"`
}

// PollMessageBus waits up to timeout for messages after since in the channels; since < 0 only asks for the last id.
func PollMessageBus(ctx context.Context, user *User, since int64, channels []string, timeout time.Duration) *BusPoll {
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	for {
		bus.lock.Lock()
		res := &BusPoll{LastId: bus.lastId, Messages: []*BusMessage{}}
		if since < 0 || since > bus.lastId {
			bus.lock.Unlock()
			return res
		}
		pending := []*BusMessage{}
		for _, message := range bus.messages {
			if message.Id > since && matchesChannel(channels, message.Channel) {
				pending = append(pending, message)
			}
		}
		var waiter chan struct{}
		if len(pending) == 0 {
			waiter = make(chan struct{})
			bus.waiters[waiter] = true
		}
		bus.lock.Unlock()

		for _, message := range pending {
			if message.canSee(user) {
				res.Messages = append(res.Messages, message)
			}
		}
		if len(res.Messages) > 0 {
			return res
		}
		// the messages were for others: wait from the newest one on
		if len(pending) > 0 {
			since = pending[len(pending)-1].Id
			continue
		}

		select {
		case <-waiter:
		case <-deadline.C:
			res.LastId = since
			return res
		case <-ctx.Done():
			res.LastId = since
			return res
		}
	}
}
