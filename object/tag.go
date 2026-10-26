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
	"strings"
	"time"

	"github.com/xorm-io/builder"
)

type Tag struct {
	Id          int64     `xorm:"pk autoincr" json:"id"`
	Name        string    `xorm:"varchar(100) notnull unique" json:"name"`
	Description string    `xorm:"varchar(1000)" json:"description,omitempty"`
	TopicCount  int       `json:"topicCount"`
	CreatedAt   time.Time `xorm:"created" json:"createdAt"`
}

type TopicTag struct {
	Id      int64 `xorm:"pk autoincr" json:"id"`
	TopicId int64 `xorm:"index unique(topic_tag)" json:"topicId"`
	TagId   int64 `xorm:"index unique(topic_tag)" json:"tagId"`
}

var tagInvalid = regexp.MustCompile(`[\s/\\?#&%<>"'` + "`" + `]+`)

func CleanTagName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = tagInvalid.ReplaceAllString(name, "-")
	name = strings.Trim(name, "-")
	return truncateRunes(name, 30)
}

func GetTag(name string) (*Tag, error) {
	tag := &Tag{}
	has, err := adapter.Engine.Where("name = ?", CleanTagName(name)).Get(tag)
	if err != nil || !has {
		return nil, err
	}
	return tag, nil
}

func GetTags(limit int) ([]*Tag, error) {
	tags := []*Tag{}
	session := adapter.Engine.Where("topic_count > ?", 0).Desc("topic_count").Asc("name")
	if limit > 0 {
		session = session.Limit(limit)
	}
	err := session.Find(&tags)
	return tags, err
}

func SearchTags(term string, limit int) ([]*Tag, error) {
	tags := []*Tag{}
	session := adapter.Engine.Desc("topic_count").Limit(limit)
	term = CleanTagName(term)
	if term != "" {
		session = session.Where(builder.Like{"name", escapeLike(term)})
	}
	err := session.Find(&tags)
	return tags, err
}

// normalizeTags cleans the tag names of a topic and checks the user may create the ones that do not exist yet.
func normalizeTags(user *User, names []string) ([]string, error) {
	res := []string{}
	if !GetSiteSettingBool("tagging_enabled") {
		return res, nil
	}
	seen := map[string]bool{}
	for _, name := range names {
		name = CleanTagName(name)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		res = append(res, name)
	}
	if max := GetSiteSettingInt("max_tags_per_topic"); len(res) > max {
		return nil, fmt.Errorf("A topic can have at most %d tags", max)
	}
	if GetSiteSettingBool("staff_only_tag_creation") && !user.IsStaff() {
		for _, name := range res {
			tag, err := GetTag(name)
			if err != nil {
				return nil, err
			}
			if tag == nil {
				return nil, fmt.Errorf("Only staff can create the tag %s", name)
			}
		}
	}
	return res, nil
}

func setTopicTags(topicId int64, names []string) error {
	old := []*TopicTag{}
	err := adapter.Engine.Where("topic_id = ?", topicId).Find(&old)
	if err != nil {
		return err
	}

	wanted := map[int64]bool{}
	for _, name := range names {
		tag, err := GetTag(name)
		if err != nil {
			return err
		}
		if tag == nil {
			tag = &Tag{Name: name}
			_, err = adapter.Engine.Insert(tag)
			if err != nil {
				return err
			}
		}
		wanted[tag.Id] = true
	}

	existing := map[int64]bool{}
	for _, row := range old {
		existing[row.TagId] = true
		if !wanted[row.TagId] {
			_, err = adapter.Engine.ID(row.Id).Delete(&TopicTag{})
			if err != nil {
				return err
			}
			_, _ = adapter.Engine.ID(row.TagId).Decr("topic_count").Update(&Tag{})
		}
	}
	for tagId := range wanted {
		if existing[tagId] {
			continue
		}
		_, err = adapter.Engine.Insert(&TopicTag{TopicId: topicId, TagId: tagId})
		if err != nil {
			return err
		}
		_, _ = adapter.Engine.ID(tagId).Incr("topic_count").Update(&Tag{})
	}
	return nil
}

func incrTag(name string, delta int) {
	_, _ = adapter.Engine.Where("name = ?", name).Incr("topic_count", delta).Update(&Tag{})
}

func UpdateTag(oldName string, tag *Tag) error {
	existing, err := GetTag(oldName)
	if err != nil {
		return err
	}
	if existing == nil {
		return fmt.Errorf("The tag %s does not exist", oldName)
	}
	tag.Name = CleanTagName(tag.Name)
	if tag.Name == "" {
		return fmt.Errorf("Name can't be empty")
	}
	if tag.Name != existing.Name {
		other, err := GetTag(tag.Name)
		if err != nil {
			return err
		}
		if other != nil {
			return fmt.Errorf("The tag %s already exists", tag.Name)
		}
	}
	_, err = adapter.Engine.ID(existing.Id).Cols("name", "description").Update(tag)
	if err != nil {
		return err
	}
	return refreshTopicTagNames(existing.Id)
}

func DeleteTag(name string) error {
	tag, err := GetTag(name)
	if err != nil || tag == nil {
		return err
	}
	rows := []*TopicTag{}
	err = adapter.Engine.Where("tag_id = ?", tag.Id).Find(&rows)
	if err != nil {
		return err
	}
	_, err = adapter.Engine.Where("tag_id = ?", tag.Id).Delete(&TopicTag{})
	if err != nil {
		return err
	}
	_, err = adapter.Engine.ID(tag.Id).Delete(&Tag{})
	if err != nil {
		return err
	}
	for _, row := range rows {
		err = refreshTopicTags(row.TopicId)
		if err != nil {
			return err
		}
	}
	return nil
}

func refreshTopicTagNames(tagId int64) error {
	rows := []*TopicTag{}
	err := adapter.Engine.Where("tag_id = ?", tagId).Find(&rows)
	if err != nil {
		return err
	}
	for _, row := range rows {
		err = refreshTopicTags(row.TopicId)
		if err != nil {
			return err
		}
	}
	return nil
}

// refreshTopicTags rewrites the tag names stored on the topic from its topic_tag rows.
func refreshTopicTags(topicId int64) error {
	names := []string{}
	err := adapter.Engine.Table("tag").Cols("tag.name").Join("INNER", "topic_tag", "topic_tag.tag_id = tag.id").
		Where("topic_tag.topic_id = ?", topicId).Asc("topic_tag.id").Find(&names)
	if err != nil {
		return err
	}
	_, err = adapter.Engine.ID(topicId).Cols("tags").Update(&Topic{Tags: names})
	return err
}
