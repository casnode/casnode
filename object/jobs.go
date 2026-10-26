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
	"math"
	"sync"
	"time"

	"github.com/beego/beego/v2/core/logs"
)

var (
	viewedTopics     = map[string]time.Time{}
	viewedTopicsLock sync.Mutex
)

func shouldCountView(topicId int64, visitorKey string) bool {
	key := fmt.Sprintf("%d/%s", topicId, visitorKey)
	viewedTopicsLock.Lock()
	defer viewedTopicsLock.Unlock()
	if at, ok := viewedTopics[key]; ok && time.Since(at) < 8*time.Hour {
		return false
	}
	viewedTopics[key] = time.Now()
	return true
}

func cleanViewedTopics() {
	viewedTopicsLock.Lock()
	defer viewedTopicsLock.Unlock()
	for key, at := range viewedTopics {
		if time.Since(at) > 8*time.Hour {
			delete(viewedTopics, key)
		}
	}
}

// refreshHotScores scores the topics active in the last 30 days for "Hot": the posts and likes of the last week,
// fading with the topic's age.
func refreshHotScores() error {
	since := time.Now().AddDate(0, 0, -30)
	topics := []*Topic{}
	err := adapter.Engine.Cols("id", "created_at", "like_count").Where("bumped_at > ? AND deleted = ?", dbTime(since), false).Find(&topics)
	if err != nil {
		return err
	}

	type row struct {
		TopicId int64
		Count   int
	}
	recentPosts := []row{}
	err = adapter.Engine.Table("post").Select("topic_id, COUNT(*) AS count").
		Where("created_at > ? AND deleted = ? AND post_type = ?", dbTime(time.Now().AddDate(0, 0, -7)), false, PostTypeRegular).
		GroupBy("topic_id").Find(&recentPosts)
	if err != nil {
		return err
	}
	recent := map[int64]int{}
	for _, r := range recentPosts {
		recent[r.TopicId] = r.Count
	}

	for _, topic := range topics {
		ageHours := time.Since(topic.CreatedAt).Hours()
		score := float64(recent[topic.Id]*2+topic.LikeCount) / math.Pow(ageHours+2, 0.8)
		_, err = adapter.Engine.ID(topic.Id).Cols("hot_score").Update(&Topic{HotScore: score})
		if err != nil {
			return err
		}
	}
	return nil
}

func RunPeriodicJobs() {
	for {
		err := refreshHotScores()
		if err != nil {
			logs.Error("refreshHotScores(): %v", err)
		}
		cleanViewedTopics()
		time.Sleep(10 * time.Minute)
	}
}
