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
	"strconv"
	"time"
)

const SystemUsername = "system"

// InitData fills an empty database with what a new forum starts with: the system user, the default categories
// and a pinned welcome topic.
func InitData() {
	count, err := adapter.Engine.Count(&Category{})
	if err != nil {
		panic(err)
	}
	if count > 0 {
		return
	}

	system, err := GetUserByUsername(SystemUsername)
	if err != nil {
		panic(err)
	}
	if system == nil {
		system = &User{Username: SystemUsername, Name: "system", Admin: true, TrustLevel: 4, LastSeenAt: time.Now()}
		_, err = adapter.Engine.Insert(system)
		if err != nil {
			panic(err)
		}
	}

	categories := []*Category{
		{Name: "Uncategorized", Slug: "uncategorized", Color: "0088CC", TextColor: "FFFFFF", Position: 1,
			Description: "Topics that don't need a category, or don't fit into any other existing category."},
		{Name: "General", Slug: "general", Color: "25AAE2", TextColor: "FFFFFF", Position: 2,
			Description: "Create topics here that don't fit into any other existing category."},
		{Name: "Site Feedback", Slug: "site-feedback", Color: "808281", TextColor: "FFFFFF", Position: 3,
			Description: "Discussion about this site, its organization, how it works, and how we can improve it."},
		{Name: "Staff", Slug: "staff", Color: "E45735", TextColor: "FFFFFF", Position: 4, ReadRestricted: true,
			Description: "Private category for staff discussions. Topics are only visible to admins and moderators."},
	}
	for _, category := range categories {
		_, err = adapter.Engine.Insert(category)
		if err != nil {
			panic(err)
		}
	}
	invalidateCategories()
	err = SetSiteSetting("uncategorized_category_id", strconv.FormatInt(categories[0].Id, 10))
	if err != nil {
		panic(err)
	}
	err = SetSiteSetting("staff_category_id", strconv.FormatInt(categories[3].Id, 10))
	if err != nil {
		panic(err)
	}

	raw := fmt.Sprintf(`We're glad you're here! This is our community, where you can connect, ask questions and share ideas.

Here are some things you can do to get started:

- **Introduce yourself** by adding your picture and information about yourself and your interests to your profile.
- **Get to know the community** by browsing discussions that are already happening here. When you find a post interesting, informative, or entertaining, use the :heart: to show your appreciation or support!
- **Contribute** by commenting, sharing your own perspective, asking questions, or offering feedback in the discussion. Before replying or starting new topics, please review the [Community Guidelines](/faq).

> If you need help or have a suggestion, feel free to ask in [Site Feedback](/c/site-feedback/%d).`, categories[2].Id)
	topic, err := CreateTopic(system, &NewTopic{Title: "Welcome to our community!", Raw: raw, CategoryId: categories[1].Id})
	if err != nil {
		panic(err)
	}
	topic.PinnedAt = time.Now()
	topic.PinnedGlobally = true
	_, err = adapter.Engine.ID(topic.Id).Cols("pinned_at", "pinned_globally").Update(topic)
	if err != nil {
		panic(err)
	}
}
