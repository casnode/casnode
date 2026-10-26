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

// import_topics brings the conversations of another system (a Discord server, an old forum...) into the forum, with
// their own authors, avatars, times and files, and without notifying anyone. Run it in the forum's folder, so that it
// uses the forum's database and uploads directory:
//
//	go run ./tools/import_topics <dir>          shows what it would do
//	go run ./tools/import_topics -apply <dir>   does it
//
// <dir> holds import.json and the files it refers to in media/. Raw text refers to a file as upload://<name of the file
// in media/>. Every user and topic carries an externalId; running it again skips what was imported before.
//
//	{
//	  "categories": [{"name": "Casdoor", "slug": "casdoor", "description": "...", "color": "0088CC"}],
//	  "users": [{"externalId": "discord:1", "username": "alice", "name": "Alice", "avatar": "upload://a.png"}],
//	  "topics": [{
//	    "externalId": "discord:2", "title": "...", "category": "Casdoor", "tags": ["oauth"],
//	    "posts": [{"externalIds": ["3", "4"], "user": "alice", "raw": "...", "createdAt": "2024-05-02T02:31:00Z",
//	               "replyToExternalId": "5"}]
//	  }]
//	}
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/casnode/casnode/object"
)

type importCategory struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Color       string `json:"color"`
}

type importUser struct {
	ExternalId string `json:"externalId"`
	Username   string `json:"username"`
	Name       string `json:"name"`
	Avatar     string `json:"avatar"`
}

type importPost struct {
	ExternalIds       []string  `json:"externalIds"`
	User              string    `json:"user"`
	Raw               string    `json:"raw"`
	CreatedAt         time.Time `json:"createdAt"`
	ReplyToExternalId string    `json:"replyToExternalId"`
}

type importTopic struct {
	ExternalId string        `json:"externalId"`
	Title      string        `json:"title"`
	Category   string        `json:"category"`
	Tags       []string      `json:"tags"`
	Posts      []*importPost `json:"posts"`
}

type importData struct {
	Categories []*importCategory `json:"categories"`
	Users      []*importUser     `json:"users"`
	Topics     []*importTopic    `json:"topics"`
}

var uploadLink = regexp.MustCompile(`(!?)\[([^\]]*)\]\(upload://([^)\s]+)\)`)

type importer struct {
	dir     string
	apply   bool
	files   map[string]string
	skipped []string
}

// file saves a file of media/ as an upload of the user, once, and returns its URL, or "" when the forum doesn't take it.
func (im *importer) file(user *object.User, name string) string {
	if url, ok := im.files[name]; ok {
		return url
	}
	url := ""
	data, err := os.ReadFile(filepath.Join(im.dir, "media", filepath.Base(name)))
	if err == nil && im.apply {
		upload, err := object.SaveUpload(user, name, data)
		if err == nil {
			url = upload.Url
		} else {
			im.skipped = append(im.skipped, fmt.Sprintf("%s: %s", name, err))
		}
	} else if err == nil {
		// the same checks SaveUpload makes
		ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(name)), ".")
		maxKb := object.GetSiteSettingInt("max_upload_size_kb")
		if !strings.Contains("|"+object.GetSiteSetting("authorized_extensions")+"|", "|"+ext+"|") {
			im.skipped = append(im.skipped, fmt.Sprintf("%s: files of type .%s are not allowed", name, ext))
		} else if maxKb > 0 && len(data) > maxKb*1024 {
			im.skipped = append(im.skipped, fmt.Sprintf("%s: larger than %d KB", name, maxKb))
		} else {
			url = "upload://" + name
		}
	} else {
		im.skipped = append(im.skipped, fmt.Sprintf("%s: %s", name, err))
	}
	im.files[name] = url
	return url
}

func (im *importer) cook(user *object.User, raw string) string {
	return uploadLink.ReplaceAllStringFunc(raw, func(link string) string {
		m := uploadLink.FindStringSubmatch(link)
		url := im.file(user, m[3])
		if url == "" {
			return m[2]
		}
		return fmt.Sprintf("%s[%s](%s)", m[1], m[2], url)
	})
}

func main() {
	apply := flag.Bool("apply", false, "import, instead of only showing what would be imported")
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Println("usage: go run ./tools/import_topics [-apply] <dir with import.json and media/>")
		os.Exit(2)
	}
	im := &importer{dir: flag.Arg(0), apply: *apply, files: map[string]string{}}
	data := &importData{}
	text, err := os.ReadFile(filepath.Join(im.dir, "import.json"))
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(text, data)
	if err != nil {
		panic(err)
	}

	object.InitAdapter()
	err = object.SyncImportTables()
	if err != nil {
		panic(err)
	}
	if !im.apply {
		fmt.Println("plan: nothing is written; run with -apply to import")
	}

	// categories
	categoryIds := map[string]int64{}
	for _, c := range data.Categories {
		slug := c.Slug
		if slug == "" {
			slug = object.Slugify(c.Name)
		}
		existing, err := object.GetCategoryBySlug(slug)
		if err != nil {
			panic(err)
		}
		if existing != nil {
			categoryIds[c.Name] = existing.Id
			fmt.Printf("category %s: exists (%d)\n", c.Name, existing.Id)
			continue
		}
		fmt.Printf("category %s: new\n", c.Name)
		if im.apply {
			category := &object.Category{Name: c.Name, Slug: slug, Description: c.Description, Color: c.Color}
			err = object.AddCategory(category)
			if err != nil {
				panic(err)
			}
			categoryIds[c.Name] = category.Id
		}
	}

	// users
	firstPost := map[string]time.Time{}
	for _, t := range data.Topics {
		for _, p := range t.Posts {
			if at, ok := firstPost[p.User]; !ok || p.CreatedAt.Before(at) {
				firstPost[p.User] = p.CreatedAt
			}
		}
	}
	users := map[string]*object.User{}
	newUsers, linkedUsers, knownUsers := 0, []string{}, 0
	for _, u := range data.Users {
		record, err := object.GetImportRecord(u.ExternalId)
		if err != nil {
			panic(err)
		}
		if record != nil {
			knownUsers++
		} else if existing, err := object.GetUserByUsername(u.Username); err != nil {
			panic(err)
		} else if existing != nil {
			linkedUsers = append(linkedUsers, u.Username)
		} else {
			newUsers++
		}
		if !im.apply {
			users[u.Username] = &object.User{Username: u.Username}
			continue
		}
		user, created, err := object.ImportUser(u.ExternalId, u.Username, u.Name, firstPost[u.Username])
		if err != nil {
			panic(err)
		}
		if created && strings.HasPrefix(u.Avatar, "upload://") {
			if url := im.file(user, strings.TrimPrefix(u.Avatar, "upload://")); url != "" {
				err = object.SetUserAvatar(user.Id, url)
				if err != nil {
					panic(err)
				}
			}
		}
		users[u.Username] = user
	}
	fmt.Printf("users: %d new, %d imported before, %d taken as the existing user of that name %v\n", newUsers, knownUsers, len(linkedUsers), linkedUsers)

	// topics
	newTopics, knownTopics, posts := 0, 0, 0
	for i, t := range data.Topics {
		record, err := object.GetImportRecord(t.ExternalId)
		if err != nil {
			panic(err)
		}
		if record != nil {
			knownTopics++
			continue
		}
		newTopics++
		posts += len(t.Posts)
		input := &object.ImportedTopic{ExternalId: t.ExternalId, Title: t.Title, CategoryId: categoryIds[t.Category], Tags: t.Tags}
		numbers := map[string]int{}
		for j, p := range t.Posts {
			for _, id := range p.ExternalIds {
				numbers[id] = j + 1
			}
		}
		for _, p := range t.Posts {
			user := users[p.User]
			if user == nil {
				panic(fmt.Sprintf("topic %s: the user %s is not in users", t.ExternalId, p.User))
			}
			replyTo := numbers[p.ReplyToExternalId]
			if replyTo == 1 {
				replyTo = 0
			}
			input.Posts = append(input.Posts, &object.ImportedPost{UserId: user.Id, Raw: im.cook(user, p.Raw), CreatedAt: p.CreatedAt, ReplyToPostNumber: replyTo})
		}
		if !im.apply {
			if newTopics <= 15 {
				fmt.Printf("  [%s] %s (%d posts, %s)\n", t.Category, t.Title, len(t.Posts), t.Posts[0].CreatedAt.Format("2006-01-02"))
			}
			continue
		}
		topic, err := object.ImportTopic(input)
		if err != nil {
			panic(err)
		}
		if (i+1)%50 == 0 || i+1 == len(data.Topics) {
			fmt.Printf("  %d/%d topics, last: /t/%s/%d\n", i+1, len(data.Topics), topic.Slug, topic.Id)
		}
	}
	fmt.Printf("topics: %d new with %d posts, %d imported before\n", newTopics, posts, knownTopics)
	fmt.Printf("files: %d, not taken: %d\n", len(im.files), len(im.skipped))
	for _, s := range im.skipped {
		fmt.Println("  " + s)
	}
	if im.apply {
		err = object.RefreshCategoryCounts()
		if err != nil {
			panic(err)
		}
	}
}
