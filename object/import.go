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
	"strings"
	"time"
)

// ImportedUserOwner marks the users an import created: nobody has signed in as them.
const ImportedUserOwner = "import"

// ImportRecord remembers what an import (tools/import_topics) created for each item of another system, so that running
// the import again skips what is already there.
type ImportRecord struct {
	Id         int64     `xorm:"pk autoincr" json:"id"`
	ExternalId string    `xorm:"varchar(150) notnull unique" json:"externalId"`
	Kind       string    `xorm:"varchar(20)" json:"kind"`
	LocalId    int64     `json:"localId"`
	CreatedAt  time.Time `xorm:"created" json:"createdAt"`
}

type ImportedPost struct {
	UserId            int64
	Raw               string
	CreatedAt         time.Time
	ReplyToPostNumber int
}

type ImportedTopic struct {
	ExternalId string
	Title      string
	CategoryId int64
	Tags       []string
	Posts      []*ImportedPost
}

func SyncImportTables() error {
	return adapter.Engine.Sync2(new(ImportRecord))
}

func GetImportRecord(externalId string) (*ImportRecord, error) {
	record := &ImportRecord{}
	has, err := adapter.Engine.Where("external_id = ?", externalId).Get(record)
	if err != nil || !has {
		return nil, err
	}
	return record, nil
}

func addImportRecord(externalId string, kind string, localId int64) error {
	_, err := adapter.Engine.Insert(&ImportRecord{ExternalId: externalId, Kind: kind, LocalId: localId})
	return err
}

// renameImportedUser moves an imported user out of the way of the Casdoor account of the same name, to <name>-<source>.
func renameImportedUser(user *User) error {
	source := "imported"
	record := &ImportRecord{}
	has, err := adapter.Engine.Where("kind = ? AND local_id = ?", "user", user.Id).Get(record)
	if err != nil {
		return err
	}
	if has && strings.Contains(record.ExternalId, ":") {
		source = strings.SplitN(record.ExternalId, ":", 2)[0]
	}
	for i := 1; ; i++ {
		name := user.Username + "-" + source
		if i > 1 {
			name = fmt.Sprintf("%s%d", name, i)
		}
		existing, err := GetUserByUsername(name)
		if err != nil {
			return err
		}
		if existing == nil {
			_, err = adapter.Engine.ID(user.Id).Cols("username").Update(&User{Username: name})
			return err
		}
	}
}

// ImportUser returns the user an external account was imported as, creating it the first time. A user of the same name
// who already exists is taken as that account, and so is a later Casdoor account of the forum's admins (conf "admins");
// anyone else of that name who signs in gets a user of their own.
func ImportUser(externalId string, username string, name string, createdAt time.Time) (*User, bool, error) {
	record, err := GetImportRecord(externalId)
	if err != nil {
		return nil, false, err
	}
	if record != nil {
		user, err := GetUser(record.LocalId)
		return user, false, err
	}

	user, err := GetUserByUsername(username)
	if err != nil {
		return nil, false, err
	}
	created := false
	if user == nil {
		user = &User{
			Username:     username,
			Name:         name,
			CasdoorOwner: ImportedUserOwner,
			TrustLevel:   1,
			CreatedAt:    createdAt,
			LastSeenAt:   createdAt,
		}
		if IsConfigAdmin(username) {
			user.CasdoorOwner = ""
		}
		_, err = adapter.Engine.NoAutoTime().Insert(user)
		if err != nil {
			return nil, false, err
		}
		created = true
	}
	return user, created, addImportRecord(externalId, "user", user.Id)
}

func SetUserAvatar(userId int64, avatar string) error {
	_, err := adapter.Engine.ID(userId).Cols("avatar").Update(&User{Avatar: avatar})
	return err
}

// ImportTopic creates a topic with its posts as they were written elsewhere: their own authors and times, and no
// notifications.
func ImportTopic(input *ImportedTopic) (*Topic, error) {
	if len(input.Posts) == 0 {
		return nil, fmt.Errorf("the topic %s has no posts", input.ExternalId)
	}
	first := input.Posts[0]
	last := input.Posts[len(input.Posts)-1]
	tags := []string{}
	for _, name := range input.Tags {
		if name = CleanTagName(name); name != "" {
			tags = append(tags, name)
		}
	}

	cooked := make([]*CookedPost, len(input.Posts))
	words := 0
	replyCounts := map[int]int{}
	for i, p := range input.Posts {
		cooked[i] = CookMarkdown(p.Raw)
		words += countWords(p.Raw)
		if p.ReplyToPostNumber > 0 {
			replyCounts[p.ReplyToPostNumber]++
		}
	}

	title := truncateRunes(input.Title, 255)
	topic := &Topic{
		Title:             title,
		Slug:              Slugify(title),
		CategoryId:        input.CategoryId,
		UserId:            first.UserId,
		Archetype:         ArchetypeRegular,
		Excerpt:           cooked[0].Excerpt,
		ImageUrl:          cooked[0].ImageUrl,
		Visible:           true,
		PostsCount:        len(input.Posts),
		ReplyCount:        len(input.Posts) - 1,
		HighestPostNumber: len(input.Posts),
		WordCount:         words,
		LastPostUserId:    last.UserId,
		LastPostedAt:      last.CreatedAt,
		BumpedAt:          last.CreatedAt,
		CreatedAt:         first.CreatedAt,
		UpdatedAt:         last.CreatedAt,
		PosterIds:         []int64{first.UserId},
		Tags:              tags,
	}

	postLock.Lock()
	defer postLock.Unlock()

	session := adapter.Engine.NewSession()
	defer session.Close()
	err := session.Begin()
	if err != nil {
		return nil, err
	}
	_, err = session.NoAutoTime().Insert(topic)
	if err != nil {
		_ = session.Rollback()
		return nil, err
	}
	for i, p := range input.Posts {
		post := &Post{
			TopicId:           topic.Id,
			PostNumber:        i + 1,
			UserId:            p.UserId,
			Raw:               p.Raw,
			Cooked:            cooked[i].Html,
			PostType:          PostTypeRegular,
			ReplyToPostNumber: p.ReplyToPostNumber,
			ReplyCount:        replyCounts[i+1],
			Version:           1,
			CreatedAt:         p.CreatedAt,
			UpdatedAt:         p.CreatedAt,
		}
		if p.ReplyToPostNumber > 0 && p.ReplyToPostNumber <= len(input.Posts) {
			post.ReplyToUserId = input.Posts[p.ReplyToPostNumber-1].UserId
		}
		_, err = session.NoAutoTime().Insert(post)
		if err != nil {
			_ = session.Rollback()
			return nil, err
		}
	}
	_, err = session.Insert(&ImportRecord{ExternalId: input.ExternalId, Kind: "topic", LocalId: topic.Id})
	if err != nil {
		_ = session.Rollback()
		return nil, err
	}
	err = session.Commit()
	if err != nil {
		return nil, err
	}

	err = setTopicTags(topic.Id, topic.Tags)
	if err != nil {
		return nil, err
	}
	incrCategory(topic.CategoryId, "topic_count", 1)
	incrCategory(topic.CategoryId, "post_count", len(input.Posts))
	incrUser(first.UserId, "topic_count", 1)
	lastPosted := map[int64]time.Time{}
	for _, p := range input.Posts {
		incrUser(p.UserId, "post_count", 1)
		lastPosted[p.UserId] = p.CreatedAt
	}
	for userId, at := range lastPosted {
		_, _ = adapter.Engine.Where("id = ? AND last_posted_at < ?", userId, dbTime(at)).Cols("last_posted_at", "last_seen_at").
			NoAutoTime().Update(&User{LastPostedAt: at, LastSeenAt: at})
	}
	return topic, refreshTopicPosters(topic)
}
