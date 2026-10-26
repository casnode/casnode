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
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"image"
	_ "image/gif"  // image sizes
	_ "image/jpeg" // image sizes
	_ "image/png"  // image sizes
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/casnode/casnode/conf"
	_ "golang.org/x/image/webp" // image sizes
)

type Upload struct {
	Id               int64     `xorm:"pk autoincr" json:"id"`
	UserId           int64     `xorm:"index" json:"userId"`
	OriginalFilename string    `xorm:"varchar(300)" json:"originalFilename"`
	Filesize         int64     `json:"filesize"`
	Width            int       `json:"width"`
	Height           int       `json:"height"`
	Sha1             string    `xorm:"varchar(40) index" json:"sha1"`
	Extension        string    `xorm:"varchar(20)" json:"extension"`
	Url              string    `xorm:"varchar(500)" json:"url"`
	CreatedAt        time.Time `xorm:"created" json:"createdAt"`
}

func IsImageExtension(ext string) bool {
	switch ext {
	case "jpg", "jpeg", "png", "gif", "webp", "svg":
		return true
	}
	return false
}

// SaveUpload stores the file under uploadsDir by its hash, so the same file uploaded twice is kept once.
func SaveUpload(user *User, filename string, data []byte) (*Upload, error) {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(filename)), ".")
	allowed := false
	for _, item := range strings.Split(GetSiteSetting("authorized_extensions"), "|") {
		if strings.TrimSpace(strings.ToLower(item)) == ext || item == "*" {
			allowed = true
		}
	}
	if !allowed {
		return nil, fmt.Errorf("Files of type .%s are not allowed", ext)
	}
	if maxKb := GetSiteSettingInt("max_upload_size_kb"); maxKb > 0 && len(data) > maxKb*1024 {
		return nil, fmt.Errorf("The file is larger than %d KB", maxKb)
	}

	hash := sha1.Sum(data)
	sum := hex.EncodeToString(hash[:])
	upload := &Upload{
		UserId:           user.Id,
		OriginalFilename: filepath.Base(filename),
		Filesize:         int64(len(data)),
		Sha1:             sum,
		Extension:        ext,
		Url:              fmt.Sprintf("/uploads/%s/%s.%s", sum[:2], sum, ext),
	}
	if IsImageExtension(ext) && ext != "svg" {
		config, _, err := image.DecodeConfig(bytes.NewReader(data))
		if err == nil {
			upload.Width = config.Width
			upload.Height = config.Height
		}
	}

	path := filepath.Join(conf.GetConfigString("uploadsDir"), sum[:2], sum+"."+ext)
	if _, err := os.Stat(path); err != nil {
		err = os.MkdirAll(filepath.Dir(path), 0o755)
		if err != nil {
			return nil, err
		}
		err = os.WriteFile(path, data, 0o644)
		if err != nil {
			return nil, err
		}
	}

	_, err := adapter.Engine.Insert(upload)
	return upload, err
}

// GetUploadPath maps a /uploads/ URL path to the file, or "" when it is not one of ours.
func GetUploadPath(urlPath string) string {
	rel := strings.TrimPrefix(urlPath, "/uploads/")
	if rel == urlPath || !filepath.IsLocal(rel) {
		return ""
	}
	return filepath.Join(conf.GetConfigString("uploadsDir"), rel)
}
