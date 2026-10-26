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
	"io"

	"github.com/casnode/casnode/object"
)

// Search
// @Title Search
// @Description full search of posts (q understands @user, #category, tags:x, in:title, status:open, order:latest|likes|views); limit 5 for the quick search under the header
// @router /search [get]
func (c *ApiController) Search() {
	user := c.GetSessionUser()
	limit := c.getInt("limit")
	if limit <= 0 || limit > 50 {
		limit = 50
	}
	res, err := object.Search(user, c.GetString("q"), c.getInt("page"), limit)
	if c.responseErr(err) {
		return
	}
	c.ResponseOk(res)
}

// Upload
// @Title Upload
// @Description uploads an image or a file for a post
// @router /upload [post]
func (c *ApiController) Upload() {
	user := c.RequireSignedInUser()
	if user == nil {
		return
	}
	if user.IsSilenced() || user.IsSuspended() {
		c.ResponseError("You can't upload files")
		return
	}
	file, header, err := c.GetFile("file")
	if c.responseErr(err) {
		return
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if c.responseErr(err) {
		return
	}
	upload, err := object.SaveUpload(user, header.Filename, data)
	if c.responseErr(err) {
		return
	}
	c.ResponseOk(upload)
}
