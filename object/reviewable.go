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

const (
	ReviewablePending  = 0
	ReviewableApproved = 1
	ReviewableRejected = 2
	ReviewableIgnored  = 3
	ReviewableDeleted  = 4
)

type ReviewableScore struct {
	UserId     int64     `json:"userId"`
	ActionType int       `json:"actionType"`
	Message    string    `json:"message,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
}

// Reviewable is a flagged post waiting for staff in the review queue.
type Reviewable struct {
	Id         int64              `xorm:"pk autoincr" json:"id"`
	PostId     int64              `xorm:"index" json:"postId"`
	TopicId    int64              `json:"topicId"`
	Status     int                `xorm:"index" json:"status"`
	Score      int                `json:"score"`
	Scores     []*ReviewableScore `xorm:"mediumtext" json:"scores"`
	ReviewedBy int64              `json:"reviewedBy,omitempty"`
	CreatedAt  time.Time          `xorm:"created" json:"createdAt"`
	UpdatedAt  time.Time          `xorm:"updated" json:"updatedAt"`

	Post  *Post  `xorm:"-" json:"post,omitempty"`
	Topic *Topic `xorm:"-" json:"topic,omitempty"`
}

func addFlagToReview(user *User, topic *Topic, post *Post, actionType int, message string) error {
	reviewable := &Reviewable{}
	has, err := adapter.Engine.Where("post_id = ? AND status = ?", post.Id, ReviewablePending).Get(reviewable)
	if err != nil {
		return err
	}
	score := &ReviewableScore{UserId: user.Id, ActionType: actionType, Message: message, CreatedAt: time.Now()}
	if !has {
		reviewable = &Reviewable{PostId: post.Id, TopicId: topic.Id, Status: ReviewablePending, Score: 1, Scores: []*ReviewableScore{score}}
		_, err = adapter.Engine.Insert(reviewable)
		return err
	}
	reviewable.Scores = append(reviewable.Scores, score)
	reviewable.Score = len(reviewable.Scores)
	_, err = adapter.Engine.ID(reviewable.Id).Cols("scores", "score").Update(reviewable)
	return err
}

type ReviewableList struct {
	Reviewables []*Reviewable   `json:"reviewables"`
	Users       map[int64]*User `json:"users"`
}

func GetReviewables(status int) (*ReviewableList, error) {
	reviewables := []*Reviewable{}
	err := adapter.Engine.Where("status = ?", status).Desc("score").Desc("id").Limit(100).Find(&reviewables)
	if err != nil {
		return nil, err
	}

	userIds := []int64{}
	for _, reviewable := range reviewables {
		reviewable.Post, err = GetPost(reviewable.PostId)
		if err != nil {
			return nil, err
		}
		reviewable.Topic, err = GetTopic(reviewable.TopicId)
		if err != nil {
			return nil, err
		}
		if reviewable.Post != nil {
			userIds = append(userIds, reviewable.Post.UserId)
		}
		for _, score := range reviewable.Scores {
			userIds = append(userIds, score.UserId)
		}
	}
	users, err := GetUserMap(userIds)
	if err != nil {
		return nil, err
	}
	return &ReviewableList{Reviewables: reviewables, Users: users}, nil
}

func CountPendingReviewables() (int64, error) {
	return adapter.Engine.Where("status = ?", ReviewablePending).Count(&Reviewable{})
}

// PerformReviewable: "agree" hides the post, "disagree" clears the flags, "delete" deletes the post, "ignore" just
// takes it off the queue.
func PerformReviewable(user *User, id int64, action string) error {
	reviewable := &Reviewable{}
	has, err := adapter.Engine.ID(id).Get(reviewable)
	if err != nil {
		return err
	}
	if !has || reviewable.Status != ReviewablePending {
		return fmt.Errorf("The item has already been reviewed")
	}
	post, err := GetPost(reviewable.PostId)
	if err != nil {
		return err
	}

	switch action {
	case "agree":
		reviewable.Status = ReviewableApproved
		if post != nil {
			_, err = adapter.Engine.ID(post.Id).Cols("hidden").Update(&Post{Hidden: true})
		}
	case "disagree":
		reviewable.Status = ReviewableRejected
		if post != nil {
			_, err = adapter.Engine.ID(post.Id).Cols("hidden").Update(&Post{Hidden: false})
		}
	case "delete":
		reviewable.Status = ReviewableDeleted
		if post != nil && !post.Deleted {
			err = DeletePost(user, post.Id)
		}
	case "ignore":
		reviewable.Status = ReviewableIgnored
	default:
		return fmt.Errorf("Unknown action %s", action)
	}
	if err != nil {
		return err
	}
	reviewable.ReviewedBy = user.Id
	_, err = adapter.Engine.ID(reviewable.Id).Cols("status", "reviewed_by").Update(reviewable)
	return err
}
