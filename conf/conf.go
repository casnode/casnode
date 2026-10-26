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

package conf

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/beego/beego/v2/core/config"
	"github.com/beego/beego/v2/server/web"
)

// localConfigPath holds a server's own settings and secrets (clientSecret, dataSourceName, ...), which override
// conf/app.conf. It is not in git, so pulling a new version never conflicts with it.
const localConfigPath = "conf/app.local.conf"

// sharedCodeDir is the code shared by the forums of a server. A thin site's folder only has conf/app.local.conf and its
// own files (uploads, sessions), and reads app.conf and the built frontend from here.
const sharedCodeDir = "../casnode"

func init() {
	loadSharedConfig()

	if _, err := os.Stat(localConfigPath); err == nil {
		local, err := config.NewConfig("ini", localConfigPath)
		if err != nil {
			panic(err)
		}
		items, err := local.GetSection("default")
		if err != nil {
			panic(err)
		}
		for key, value := range items {
			err = web.AppConfig.Set(key, value)
			if err != nil {
				panic(err)
			}
		}
	}

	presetConfigItems := []string{"httpport", "appname"}
	for _, key := range presetConfigItems {
		if value, ok := os.LookupEnv(key); ok {
			err := web.AppConfig.Set(key, value)
			if err != nil {
				panic(err)
			}
		}
	}
}

// loadSharedConfig reads the app.conf of the shared code when the site's folder has none (a thin site).
func loadSharedConfig() {
	if fileExists("conf/app.conf") {
		return
	}
	path := filepath.Join(sharedCodeDir, "conf/app.conf")
	if !fileExists(path) {
		return
	}
	err := web.LoadAppConfig("ini", path)
	if err != nil {
		panic(err)
	}
}

// GetSharedPath returns path, or the same path in the shared code when the site's folder doesn't have it.
func GetSharedPath(path string) string {
	if fileExists(path) || filepath.IsAbs(path) {
		return path
	}
	res := filepath.Join(sharedCodeDir, path)
	if fileExists(res) {
		return res
	}
	return path
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

var defaults = map[string]string{
	"driverName": "sqlite",
	"dbName":     "casnode.db",
	"uploadsDir": "files/uploads",
}

// GetConfigString reads a config item, letting an environment variable of the same name override app.conf.
func GetConfigString(key string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	res, _ := web.AppConfig.String(key)
	if res == "" {
		res = defaults[key]
	}
	return res
}

func GetConfigBool(key string) bool {
	return GetConfigString(key) == "true"
}

func GetConfigInt(key string, defaultValue int) int {
	value, err := strconv.Atoi(GetConfigString(key))
	if err != nil {
		return defaultValue
	}
	return value
}

// GetConfigList splits a comma-separated config item, dropping empty entries.
func GetConfigList(key string) []string {
	res := []string{}
	for _, item := range strings.Split(GetConfigString(key), ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			res = append(res, item)
		}
	}
	return res
}
