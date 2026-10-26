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
	"time"

	"github.com/casnode/casnode/object"
)

// GetAbout
// @Title GetAbout
// @Description the About page: the about text, the site's numbers, and its admins and moderators
// @router /get-about [get]
func (c *ApiController) GetAbout() {
	stats := map[string]int64{}
	periods := map[string]time.Time{
		"":   {},
		"7":  time.Now().AddDate(0, 0, -7),
		"30": time.Now().AddDate(0, 0, -30),
	}
	var err error
	for suffix, since := range periods {
		stats["topics"+suffix], err = object.CountTopics(since)
		if c.responseErr(err) {
			return
		}
		stats["posts"+suffix], err = object.CountPosts(since)
		if c.responseErr(err) {
			return
		}
		stats["users"+suffix], err = object.CountUsersSince(since)
		if c.responseErr(err) {
			return
		}
	}
	staff, _, err := object.GetAdminUsers("staff", "", 0)
	if c.responseErr(err) {
		return
	}
	admins := []*object.User{}
	moderators := []*object.User{}
	for _, user := range staff {
		if user.Username == object.SystemUsername {
			continue
		}
		if user.Admin {
			admins = append(admins, user.Basic())
		} else {
			moderators = append(moderators, user.Basic())
		}
	}
	c.ResponseOk(map[string]interface{}{
		"aboutHtml":  object.CookMarkdown(object.GetSiteSetting("about_text")).Html,
		"stats":      stats,
		"admins":     admins,
		"moderators": moderators,
	})
}

var staticPages = map[string]string{
	"faq":     "faq_text",
	"tos":     "tos_text",
	"privacy": "privacy_text",
}

// GetStaticPage
// @Title GetStaticPage
// @Description the FAQ (community guidelines), terms of service or privacy policy, as HTML
// @router /get-static-page [get]
func (c *ApiController) GetStaticPage() {
	setting, ok := staticPages[c.GetString("name")]
	if !ok {
		c.ResponseError("Unknown page")
		return
	}
	c.ResponseOk(object.CookMarkdown(object.GetSiteSetting(setting)).Html)
}
