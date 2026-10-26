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

package controllers

import (
	"github.com/casnode/casnode/object"
)

type pollVoteInput struct {
	PostId  int64    `json:"postId"`
	Name    string   `json:"name"`
	Options []string `json:"options"`
}

// VotePoll
// @Title VotePoll
// @Description votes in a poll of a post: options are the ids of the options chosen
// @router /vote-poll [post]
func (c *ApiController) VotePoll() {
	user := c.RequireSignedInUser()
	if user == nil {
		return
	}
	input := &pollVoteInput{}
	if !c.readJsonBody(input) {
		return
	}
	poll, err := object.VotePoll(user, input.PostId, input.Name, input.Options)
	if c.responseErr(err) {
		return
	}
	c.ResponseOk(poll)
}

// RemovePollVote
// @Title RemovePollVote
// @router /remove-poll-vote [post]
func (c *ApiController) RemovePollVote() {
	user := c.RequireSignedInUser()
	if user == nil {
		return
	}
	poll, err := object.RemovePollVote(user, c.getInt64("postId"), c.GetString("name"))
	if c.responseErr(err) {
		return
	}
	c.ResponseOk(poll)
}

// SetPollStatus
// @Title SetPollStatus
// @Description opens or closes a poll (status open|closed)
// @router /set-poll-status [post]
func (c *ApiController) SetPollStatus() {
	user := c.RequireSignedInUser()
	if user == nil {
		return
	}
	poll, err := object.SetPollStatus(user, c.getInt64("postId"), c.GetString("name"), c.GetString("status"))
	if c.responseErr(err) {
		return
	}
	c.ResponseOk(poll)
}

// GetPollVoters
// @Title GetPollVoters
// @Description who voted for an option of a public poll
// @router /get-poll-voters [get]
func (c *ApiController) GetPollVoters() {
	user := c.RequireSignedInUser()
	if user == nil {
		return
	}
	voters, err := object.GetPollVoters(user, c.getInt64("postId"), c.GetString("name"), c.GetString("option"))
	if c.responseErr(err) {
		return
	}
	c.ResponseOk(voters)
}

// MovePosts
// @Title MovePosts
// @Description moves posts to a new topic or to the end of another; moving all of them merges the topic
// @router /move-posts [post]
func (c *ApiController) MovePosts() {
	user := c.RequireStaff()
	if user == nil {
		return
	}
	input := &object.MovePostsInput{}
	if !c.readJsonBody(input) {
		return
	}
	topic, err := object.MovePosts(user, input)
	if c.responseErr(err) {
		return
	}
	c.ResponseOk(topic)
}

// SearchTopicsForMove
// @Title SearchTopicsForMove
// @Description topics to move posts into, by title, id or link
// @router /search-topics-for-move [get]
func (c *ApiController) SearchTopicsForMove() {
	user := c.RequireStaff()
	if user == nil {
		return
	}
	archetype := c.GetString("archetype")
	if archetype != object.ArchetypePrivateMessage {
		archetype = object.ArchetypeRegular
	}
	topics, err := object.SearchTopicsForMove(user, c.GetString("term"), archetype)
	if c.responseErr(err) {
		return
	}
	c.ResponseOk(topics)
}
