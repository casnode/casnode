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

package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web"
	"github.com/casnode/casnode/conf"
	"github.com/casnode/casnode/object"
	"github.com/casnode/casnode/routers"
	"github.com/casnode/casnode/util"
)

func main() {
	object.InitAdapter()
	object.InitData()
	object.InitFeatures()

	web.BConfig.WebConfig.Session.SessionOn = true
	web.BConfig.WebConfig.Session.SessionName = "casnode_session_id"
	web.BConfig.WebConfig.Session.SessionProvider = "file"
	web.BConfig.WebConfig.Session.SessionProviderConfig = "./tmp"
	web.BConfig.WebConfig.Session.SessionGCMaxLifetime = 3600 * 24 * 30
	web.BConfig.WebConfig.Session.SessionCookieLifeTime = 3600 * 24 * 30
	web.BConfig.WebConfig.Session.SessionCookieSameSite = http.SameSiteLaxMode
	web.BConfig.MaxUploadSize = 64 << 20
	web.BConfig.CopyRequestBody = true

	routers.InitAPI()
	web.InsertFilter("*", web.BeforeRouter, routers.StaticFilter)
	logs.SetLogFuncCall(false)

	port := conf.GetConfigInt("httpport", 16900)
	err := util.StopOldInstance(port)
	if err != nil {
		panic(err)
	}
	// give the system a moment to release the port
	time.Sleep(500 * time.Millisecond)

	// the Next.js server listens on the next port and renders every page
	routers.InitFrontendProxy(port + 1)
	routers.StartFrontend(port+1, port)

	go object.RunPeriodicJobs()

	web.Run(fmt.Sprintf(":%v", port))
}
