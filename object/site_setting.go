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
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

// SiteSetting is one of the settings admins change on /admin/site_settings.
type SiteSetting struct {
	Name      string    `xorm:"varchar(100) pk" json:"name"`
	Value     string    `xorm:"mediumtext" json:"value"`
	UpdatedAt time.Time `xorm:"updated" json:"updatedAt"`
}

type SettingDef struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Default  string `json:"default"`
	Category string `json:"category"`
	Public   bool   `json:"-"`
	Choices  string `json:"choices,omitempty"`
}

// SettingDefs are all the settings, with their defaults. Public ones are sent to every page.
var SettingDefs = []*SettingDef{
	{Name: "title", Type: "string", Default: "Casnode", Category: "required", Public: true},
	{Name: "site_description", Type: "string", Default: "A forum for discussion", Category: "required", Public: true},
	{Name: "short_site_description", Type: "string", Default: "", Category: "required", Public: true},
	{Name: "contact_email", Type: "string", Default: "", Category: "required", Public: true},
	{Name: "site_url", Type: "string", Default: "", Category: "required", Public: true},
	{Name: "logo_url", Type: "string", Default: "", Category: "branding", Public: true},
	{Name: "logo_small_url", Type: "string", Default: "", Category: "branding", Public: true},
	{Name: "logo_dark_url", Type: "string", Default: "", Category: "branding", Public: true},
	{Name: "favicon_url", Type: "string", Default: "", Category: "branding", Public: true},
	{Name: "brand_color", Type: "color", Default: "", Category: "branding", Public: true},
	{Name: "brand_color_dark", Type: "color", Default: "", Category: "branding", Public: true},
	{Name: "accent_color", Type: "color", Default: "", Category: "branding", Public: true},
	{Name: "accent_color_dark", Type: "color", Default: "", Category: "branding", Public: true},
	{Name: "header_background_color", Type: "color", Default: "", Category: "branding", Public: true},
	{Name: "header_background_color_dark", Type: "color", Default: "", Category: "branding", Public: true},
	{Name: "header_text_color", Type: "color", Default: "", Category: "branding", Public: true},
	{Name: "header_text_color_dark", Type: "color", Default: "", Category: "branding", Public: true},
	{Name: "base_font", Type: "enum", Default: "inter", Category: "branding", Public: true, Choices: fontChoices},
	{Name: "heading_font", Type: "enum", Default: "inter", Category: "branding", Public: true, Choices: fontChoices},
	{Name: "header_links", Type: "list", Default: "", Category: "branding", Public: true},
	{Name: "homepage_banner_title", Type: "string", Default: "", Category: "branding", Public: true},
	{Name: "homepage_banner_text", Type: "string", Default: "", Category: "branding", Public: true},
	{Name: "homepage_banner_image_url", Type: "string", Default: "", Category: "branding", Public: true},
	{Name: "homepage_banner_search", Type: "bool", Default: "true", Category: "branding", Public: true},
	{Name: "custom_css", Type: "css", Default: "", Category: "branding", Public: true},
	{Name: "default_locale", Type: "enum", Default: "en", Category: "basic", Public: true, Choices: "en|zh"},
	{Name: "top_menu", Type: "list", Default: "latest|new|unread|hot|top|categories", Category: "basic", Public: true},
	{Name: "desktop_category_page_style", Type: "enum", Default: "categories_and_latest_topics", Category: "basic", Public: true,
		Choices: "categories_and_latest_topics|categories_only|categories_boxes"},
	{Name: "about_text", Type: "text", Default: "", Category: "basic", Public: true},
	{Name: "faq_text", Type: "text", Default: defaultFaq, Category: "basic", Public: true},
	{Name: "tos_text", Type: "text", Default: "", Category: "basic", Public: true},
	{Name: "privacy_text", Type: "text", Default: "", Category: "basic", Public: true},
	{Name: "header_html", Type: "html", Default: "", Category: "basic", Public: true},
	{Name: "footer_html", Type: "html", Default: "", Category: "basic", Public: true},
	{Name: "allow_uncategorized_topics", Type: "bool", Default: "true", Category: "posting", Public: true},
	{Name: "min_topic_title_length", Type: "integer", Default: "15", Category: "posting", Public: true},
	{Name: "max_topic_title_length", Type: "integer", Default: "255", Category: "posting", Public: true},
	{Name: "min_post_length", Type: "integer", Default: "20", Category: "posting", Public: true},
	{Name: "min_first_post_length", Type: "integer", Default: "20", Category: "posting", Public: true},
	{Name: "min_personal_message_post_length", Type: "integer", Default: "10", Category: "posting", Public: true},
	{Name: "max_post_length", Type: "integer", Default: "32000", Category: "posting", Public: true},
	{Name: "post_edit_time_limit_minutes", Type: "integer", Default: "0", Category: "posting", Public: true},
	{Name: "max_tags_per_topic", Type: "integer", Default: "5", Category: "tags", Public: true},
	{Name: "tagging_enabled", Type: "bool", Default: "true", Category: "tags", Public: true},
	{Name: "staff_only_tag_creation", Type: "bool", Default: "false", Category: "tags", Public: true},
	{Name: "new_topic_duration_days", Type: "integer", Default: "2", Category: "basic", Public: true},
	{Name: "enable_personal_messages", Type: "bool", Default: "true", Category: "basic", Public: true},
	{Name: "login_required", Type: "bool", Default: "false", Category: "login", Public: true},
	{Name: "max_upload_size_kb", Type: "integer", Default: "4096", Category: "files", Public: true},
	{Name: "authorized_extensions", Type: "list", Default: "jpg|jpeg|png|gif|webp|svg|pdf|zip|txt", Category: "files", Public: true},
	{Name: "poll_enabled", Type: "bool", Default: "true", Category: "posting", Public: true},
	{Name: "chat_enabled", Type: "bool", Default: "true", Category: "chat", Public: true},
	{Name: "chat_max_message_length", Type: "integer", Default: "6000", Category: "chat", Public: true},
	{Name: "enable_badges", Type: "bool", Default: "true", Category: "badges", Public: true},
	{Name: "email_enabled", Type: "bool", Default: "false", Category: "email", Public: true},
	{Name: "notification_email", Type: "string", Default: "", Category: "email", Public: false},
	{Name: "smtp_host", Type: "string", Default: "", Category: "email", Public: false},
	{Name: "smtp_port", Type: "integer", Default: "587", Category: "email", Public: false},
	{Name: "smtp_security", Type: "enum", Default: "starttls", Category: "email", Public: false, Choices: "starttls|ssl|none"},
	{Name: "smtp_username", Type: "string", Default: "", Category: "email", Public: false},
	{Name: "smtp_password", Type: "secret", Default: "", Category: "email", Public: false},
	{Name: "email_delay_minutes", Type: "integer", Default: "5", Category: "email", Public: false},
	{Name: "google_analytics_id", Type: "string", Default: "", Category: "analytics", Public: true},
	{Name: "baidu_analytics_id", Type: "string", Default: "", Category: "analytics", Public: true},
	{Name: "uncategorized_category_id", Type: "integer", Default: "0", Category: "hidden", Public: true},
	{Name: "staff_category_id", Type: "integer", Default: "0", Category: "hidden", Public: false},
}

const fontChoices = "inter|system|helvetica|arial|verdana|georgia|times"

var colorRegex = regexp.MustCompile(`^#([0-9a-fA-F]{3}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})$`)

var (
	settingCache     map[string]string
	settingCacheLock sync.RWMutex
)

func getSettingDef(name string) *SettingDef {
	for _, def := range SettingDefs {
		if def.Name == name {
			return def
		}
	}
	return nil
}

func loadSettings() map[string]string {
	settingCacheLock.RLock()
	cached := settingCache
	settingCacheLock.RUnlock()
	if cached != nil {
		return cached
	}

	res := map[string]string{}
	for _, def := range SettingDefs {
		res[def.Name] = def.Default
	}
	settings := []*SiteSetting{}
	err := adapter.Engine.Find(&settings)
	if err != nil {
		return res
	}
	for _, setting := range settings {
		res[setting.Name] = setting.Value
	}

	settingCacheLock.Lock()
	settingCache = res
	settingCacheLock.Unlock()
	return res
}

func GetSiteSetting(name string) string {
	return loadSettings()[name]
}

func GetSiteSettingInt(name string) int {
	value, _ := strconv.Atoi(GetSiteSetting(name))
	return value
}

func GetSiteSettingBool(name string) bool {
	return GetSiteSetting(name) == "true"
}

// GetPublicSettings are the settings every page may know.
func GetPublicSettings() map[string]string {
	settings := loadSettings()
	res := map[string]string{}
	for _, def := range SettingDefs {
		if def.Public {
			res[def.Name] = settings[def.Name]
		}
	}
	return res
}

type SettingItem struct {
	*SettingDef
	Value string `json:"value"`
}

func GetAllSettings() []*SettingItem {
	settings := loadSettings()
	res := []*SettingItem{}
	for _, def := range SettingDefs {
		if def.Category == "hidden" {
			continue
		}
		value := settings[def.Name]
		if def.Type == "secret" && value != "" {
			value = SecretPlaceholder
		}
		res = append(res, &SettingItem{SettingDef: def, Value: value})
	}
	return res
}

// SecretPlaceholder stands for a secret setting that is set, which the admin page never gets back.
const SecretPlaceholder = "********"

func SetSiteSetting(name string, value string) error {
	def := getSettingDef(name)
	if def == nil {
		return fmt.Errorf("Unknown setting %s", name)
	}
	if def.Type == "secret" && value == SecretPlaceholder {
		return nil
	}
	if def.Type == "integer" {
		if _, err := strconv.Atoi(value); err != nil {
			return fmt.Errorf("%s must be a number", name)
		}
	}
	if def.Type == "bool" && value != "true" && value != "false" {
		return fmt.Errorf("%s must be true or false", name)
	}
	if def.Type == "color" && value != "" && !colorRegex.MatchString(value) {
		return fmt.Errorf("%s must be a hex color like #0088cc", name)
	}
	if def.Type == "enum" && !slices.Contains(strings.Split(def.Choices, "|"), value) {
		return fmt.Errorf("%s must be one of %s", name, strings.ReplaceAll(def.Choices, "|", ", "))
	}

	has, err := adapter.Engine.Exist(&SiteSetting{Name: name})
	if err != nil {
		return err
	}
	if has {
		_, err = adapter.Engine.ID(name).Cols("value").Update(&SiteSetting{Value: value})
	} else {
		_, err = adapter.Engine.Insert(&SiteSetting{Name: name, Value: value})
	}

	settingCacheLock.Lock()
	settingCache = nil
	settingCacheLock.Unlock()
	return err
}

const defaultFaq = `## Community guidelines

These guidelines keep the forum a useful and friendly place. Admins can replace this text in the site settings.

## Be helpful

Post when you have something to add: an answer, a question, an experience or a useful link. Search before asking, and give enough detail for others to understand your problem.

## Be respectful

Discuss ideas, not people. Disagreement is welcome; insults, harassment, discrimination and personal attacks are not.

## Stay on topic

Start a topic in the category that fits it, keep replies relevant to the topic, and don't post the same thing in several places.

## No spam

Don't advertise, don't post links only to promote something, and don't post content you don't have the right to share.

## Report problems

If a post breaks these guidelines, flag it instead of replying to it. The staff will review it.`
