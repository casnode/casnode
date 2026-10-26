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

// GetCategories
// @Title GetCategories
// @Description returns the categories with their activity, and the latest topics, for the categories page
// @router /get-categories [get]
func (c *ApiController) GetCategories() {
	user := c.GetSessionUser()
	categories, err := object.GetCategoriesWithStats(user)
	if c.responseErr(err) {
		return
	}
	latest, err := object.GetTopicList(user, &object.TopicListQuery{Filter: "latest", Order: "activity", PageSize: 25})
	if c.responseErr(err) {
		return
	}
	c.ResponseOk(object.SortCategoriesForDisplay(categories), latest)
}

// AddCategory
// @Title AddCategory
// @router /add-category [post]
func (c *ApiController) AddCategory() {
	if c.RequireAdmin() == nil {
		return
	}
	category := &object.Category{}
	if !c.readJsonBody(category) {
		return
	}
	category.Id = 0
	if c.responseErr(object.AddCategory(category)) {
		return
	}
	c.ResponseOk(category)
}

// UpdateCategory
// @Title UpdateCategory
// @router /update-category [post]
func (c *ApiController) UpdateCategory() {
	if c.RequireAdmin() == nil {
		return
	}
	category := &object.Category{}
	if !c.readJsonBody(category) {
		return
	}
	existing, err := object.GetCategory(category.Id)
	if c.responseErr(err) {
		return
	}
	if existing == nil {
		c.ResponseError("The category does not exist")
		return
	}
	if c.responseErr(object.UpdateCategory(category)) {
		return
	}
	c.ResponseOk(category)
}

// DeleteCategory
// @Title DeleteCategory
// @Description deletes a category, moving its topics to Uncategorized
// @router /delete-category [post]
func (c *ApiController) DeleteCategory() {
	if c.RequireAdmin() == nil {
		return
	}
	if c.responseErr(object.DeleteCategory(c.getInt64("id"))) {
		return
	}
	c.ResponseOk()
}

type reorderInput struct {
	Ids []int64 `json:"ids"`
}

// ReorderCategories
// @Title ReorderCategories
// @router /reorder-categories [post]
func (c *ApiController) ReorderCategories() {
	if c.RequireAdmin() == nil {
		return
	}
	input := &reorderInput{}
	if !c.readJsonBody(input) {
		return
	}
	if c.responseErr(object.ReorderCategories(input.Ids)) {
		return
	}
	c.ResponseOk()
}

// GetTags
// @Title GetTags
// @Description returns all tags in use, or those matching q
// @router /get-tags [get]
func (c *ApiController) GetTags() {
	q := c.GetString("q")
	var tags []*object.Tag
	var err error
	if q != "" {
		tags, err = object.SearchTags(q, 20)
	} else {
		tags, err = object.GetTags(0)
	}
	if c.responseErr(err) {
		return
	}
	c.ResponseOk(tags)
}

type tagInput struct {
	OldName     string `json:"oldName"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// UpdateTag
// @Title UpdateTag
// @router /update-tag [post]
func (c *ApiController) UpdateTag() {
	if c.RequireStaff() == nil {
		return
	}
	input := &tagInput{}
	if !c.readJsonBody(input) {
		return
	}
	if c.responseErr(object.UpdateTag(input.OldName, &object.Tag{Name: input.Name, Description: input.Description})) {
		return
	}
	c.ResponseOk()
}

// DeleteTag
// @Title DeleteTag
// @router /delete-tag [post]
func (c *ApiController) DeleteTag() {
	if c.RequireStaff() == nil {
		return
	}
	if c.responseErr(object.DeleteTag(c.GetString("name"))) {
		return
	}
	c.ResponseOk()
}
