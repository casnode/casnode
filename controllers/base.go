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
	"encoding/json"
	"fmt"
	"net"
	"strings"

	"github.com/beego/beego/v2/server/web"
	"github.com/casnode/casnode/conf"
	"github.com/casnode/casnode/object"
)

type ApiController struct {
	web.Controller
	user       *object.User
	userLoaded bool
}

type Response struct {
	Status string      `json:"status"`
	Msg    string      `json:"msg"`
	Data   interface{} `json:"data"`
	Data2  interface{} `json:"data2"`
}

func (c *ApiController) ResponseOk(data ...interface{}) {
	res := &Response{Status: "ok"}
	if len(data) > 0 {
		res.Data = data[0]
	}
	if len(data) > 1 {
		res.Data2 = data[1]
	}
	c.Data["json"] = res
	c.ServeJSON()
}

func (c *ApiController) ResponseError(error string) {
	c.Data["json"] = &Response{Status: "error", Msg: error}
	c.ServeJSON()
}

// responseErr answers with err and tells whether there was one.
func (c *ApiController) responseErr(err error) bool {
	if err == nil {
		return false
	}
	c.ResponseError(err.Error())
	return true
}

func (c *ApiController) getClientIp() string {
	if conf.GetConfigBool("trustProxyHeader") {
		forwardedFor := c.Ctx.Request.Header.Get("X-Forwarded-For")
		if forwardedFor != "" {
			return strings.TrimSpace(strings.Split(forwardedFor, ",")[0])
		}
		realIp := c.Ctx.Request.Header.Get("X-Real-IP")
		if realIp != "" {
			return realIp
		}
	}

	host, _, err := net.SplitHostPort(c.Ctx.Request.RemoteAddr)
	if err != nil {
		return c.Ctx.Request.RemoteAddr
	}
	return host
}

func (c *ApiController) readJsonBody(v interface{}) bool {
	err := json.Unmarshal(c.Ctx.Input.RequestBody, v)
	if err != nil {
		c.ResponseError(fmt.Sprintf("invalid request body: %v", err))
		return false
	}
	return true
}

// GetSessionUser returns the signed-in user, read fresh from the database so that a suspension or a new role holds
// at once; nil when nobody is signed in.
func (c *ApiController) GetSessionUser() *object.User {
	if c.userLoaded {
		return c.user
	}
	c.userLoaded = true

	userId, ok := c.GetSession("userId").(int64)
	if !ok || userId == 0 {
		return nil
	}
	user, err := object.GetUser(userId)
	if err != nil || user == nil {
		return nil
	}
	if object.IsConfigAdmin(user.Username) {
		user.Admin = true
	}
	object.TouchUser(user)
	c.user = user
	return user
}

func (c *ApiController) SetSessionUser(user *object.User) error {
	c.user = user
	c.userLoaded = true
	if user == nil {
		return c.DelSession("userId")
	}
	return c.SetSession("userId", user.Id)
}

// RequireSignedInUser responds with an error and returns nil when nobody is signed in.
func (c *ApiController) RequireSignedInUser() *object.User {
	user := c.GetSessionUser()
	if user == nil {
		c.ResponseError("Please sign in first")
		return nil
	}
	return user
}

func (c *ApiController) RequireStaff() *object.User {
	user := c.RequireSignedInUser()
	if user == nil {
		return nil
	}
	if !user.IsStaff() {
		c.ResponseError("Only staff can do this")
		return nil
	}
	return user
}

func (c *ApiController) RequireAdmin() *object.User {
	user := c.RequireSignedInUser()
	if user == nil {
		return nil
	}
	if !user.Admin {
		c.ResponseError("Only administrators can do this")
		return nil
	}
	return user
}

// visitorKey identifies a reader for counting topic views once.
func (c *ApiController) visitorKey(user *object.User) string {
	if user != nil {
		return fmt.Sprintf("u%d", user.Id)
	}
	forwarded := c.Ctx.Request.Header.Get("X-Visitor-Ip")
	if forwarded != "" && isFromLocalhost(c.Ctx.Request.RemoteAddr) {
		// the Next.js server renders the page and passes on who asked
		return forwarded
	}
	return c.getClientIp()
}

func isFromLocalhost(remoteAddr string) bool {
	host, _, _ := net.SplitHostPort(remoteAddr)
	return host == "127.0.0.1" || host == "::1"
}

func (c *ApiController) getInt64(key string) int64 {
	value, _ := c.GetInt64(key)
	return value
}

func (c *ApiController) getInt(key string) int {
	value, _ := c.GetInt(key)
	return value
}
