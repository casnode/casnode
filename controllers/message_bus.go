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
	"strings"
	"time"

	"github.com/casnode/casnode/object"
)

const pollTimeout = 25 * time.Second

// PollMessageBus
// @Title PollMessageBus
// @Description waits for the messages after since in the channels (comma-separated, "/chat/" takes every chat
// channel); since=-1 only returns the last id
// @router /message-bus-poll [get]
func (c *ApiController) PollMessageBus() {
	// a page left open polls all the time, which is not the user being around
	c.skipTouch = true
	user := c.GetSessionUser()
	// beego writes the session back as it was read when the request ends; a poll ends long after it started, and
	// writing then would undo a sign-in or sign-out done meanwhile
	c.Ctx.Input.CruSession = nil
	channels := []string{}
	for _, channel := range strings.Split(c.GetString("channels"), ",") {
		channel = strings.TrimSpace(channel)
		if channel != "" && len(channels) < 50 {
			channels = append(channels, channel)
		}
	}
	since, err := c.GetInt64("since")
	if err != nil {
		since = -1
	}
	c.Ctx.Output.Header("Cache-Control", "no-store")
	c.ResponseOk(object.PollMessageBus(c.Ctx.Request.Context(), user, since, channels, pollTimeout))
}
