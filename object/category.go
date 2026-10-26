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
	"sort"
	"strings"
	"sync"
	"time"
)

type Category struct {
	Id               int64     `xorm:"pk autoincr" json:"id"`
	Name             string    `xorm:"varchar(100) notnull" json:"name"`
	Slug             string    `xorm:"varchar(100) notnull unique" json:"slug"`
	Description      string    `xorm:"mediumtext" json:"description"`
	Color            string    `xorm:"varchar(10)" json:"color"`
	TextColor        string    `xorm:"varchar(10)" json:"textColor"`
	ParentId         int64     `xorm:"index" json:"parentId"`
	Position         int       `json:"position"`
	TopicCount       int       `json:"topicCount"`
	PostCount        int       `json:"postCount"`
	ReadRestricted   bool      `json:"readRestricted"`
	CreateRestricted bool      `json:"createRestricted"`
	TopicTemplate    string    `xorm:"mediumtext" json:"topicTemplate"`
	CreatedAt        time.Time `xorm:"created" json:"createdAt"`

	TopicsWeek  int     `xorm:"-" json:"topicsWeek"`
	TopicsMonth int     `xorm:"-" json:"topicsMonth"`
	SubIds      []int64 `xorm:"-" json:"subcategoryIds"`
	// the groups let into a restricted category
	Permissions []*CategoryPermission `xorm:"-" json:"permissions"`
}

type CategoryPermission struct {
	GroupId        int64 `json:"groupId"`
	PermissionType int   `json:"permissionType"`
}

// The categories rarely change and every page shows them, so they are kept in memory.
var (
	categoryCache     []*Category
	categoryCacheLock sync.RWMutex
)

func loadCategories() ([]*Category, error) {
	categoryCacheLock.RLock()
	cached := categoryCache
	categoryCacheLock.RUnlock()
	if cached != nil {
		return cached, nil
	}

	categories := []*Category{}
	err := adapter.Engine.Asc("position").Asc("id").Find(&categories)
	if err != nil {
		return nil, err
	}
	permissions := []*CategoryGroup{}
	err = adapter.Engine.Asc("id").Find(&permissions)
	if err != nil {
		return nil, err
	}
	for _, category := range categories {
		for _, sub := range categories {
			if sub.ParentId == category.Id {
				category.SubIds = append(category.SubIds, sub.Id)
			}
		}
		category.Permissions = []*CategoryPermission{}
		for _, permission := range permissions {
			if permission.CategoryId == category.Id {
				category.Permissions = append(category.Permissions, &CategoryPermission{GroupId: permission.GroupId, PermissionType: permission.PermissionType})
			}
		}
	}

	categoryCacheLock.Lock()
	categoryCache = categories
	categoryCacheLock.Unlock()
	return categories, nil
}

func invalidateCategories() {
	categoryCacheLock.Lock()
	categoryCache = nil
	categoryCacheLock.Unlock()
}

// groupPermission is the best permission the user's groups have in the category: 0 when none of them is let in.
func (c *Category) groupPermission(user *User) int {
	if user == nil || len(c.Permissions) == 0 {
		return 0
	}
	groupIds := GetUserGroupIds(user.Id)
	best := 0
	for _, permission := range c.Permissions {
		if containsId(groupIds, permission.GroupId) && (best == 0 || permission.PermissionType < best) {
			best = permission.PermissionType
		}
	}
	return best
}

// A read-restricted category is for staff and the groups let in; a create-restricted one takes new topics only from
// staff and groups with full permission.
func (c *Category) CanSee(user *User) bool {
	return !c.ReadRestricted || user.IsStaff() || c.groupPermission(user) != 0
}

func (c *Category) CanCreateTopic(user *User) bool {
	if user == nil {
		return false
	}
	permission := c.groupPermission(user)
	return user.IsStaff() || permission == CategoryPermissionFull || (!c.ReadRestricted && !c.CreateRestricted)
}

func (c *Category) CanReply(user *User) bool {
	if user == nil {
		return false
	}
	permission := c.groupPermission(user)
	return user.IsStaff() || !c.ReadRestricted || permission == CategoryPermissionFull || permission == CategoryPermissionCreatePost
}

func saveCategoryPermissions(category *Category) error {
	_, err := adapter.Engine.Where("category_id = ?", category.Id).Delete(&CategoryGroup{})
	if err != nil {
		return err
	}
	seen := map[int64]bool{}
	for _, permission := range category.Permissions {
		if permission.GroupId == 0 || seen[permission.GroupId] {
			continue
		}
		seen[permission.GroupId] = true
		if permission.PermissionType < CategoryPermissionFull || permission.PermissionType > CategoryPermissionReadonly {
			permission.PermissionType = CategoryPermissionFull
		}
		_, err = adapter.Engine.Insert(&CategoryGroup{CategoryId: category.Id, GroupId: permission.GroupId, PermissionType: permission.PermissionType})
		if err != nil {
			return err
		}
	}
	return nil
}

// GetCategories returns the categories the user may see, in their order.
func GetCategories(user *User) ([]*Category, error) {
	categories, err := loadCategories()
	if err != nil {
		return nil, err
	}
	res := []*Category{}
	for _, category := range categories {
		if category.CanSee(user) {
			res = append(res, category)
		}
	}
	return res, nil
}

func GetCategory(id int64) (*Category, error) {
	categories, err := loadCategories()
	if err != nil {
		return nil, err
	}
	for _, category := range categories {
		if category.Id == id {
			return category, nil
		}
	}
	return nil, nil
}

func GetCategoryBySlug(slug string) (*Category, error) {
	categories, err := loadCategories()
	if err != nil {
		return nil, err
	}
	for _, category := range categories {
		if strings.EqualFold(category.Slug, slug) {
			return category, nil
		}
	}
	return nil, nil
}

// GetHiddenCategoryIds are the categories the user may not see, to leave out of topic lists.
func GetHiddenCategoryIds(user *User) []int64 {
	categories, err := loadCategories()
	if err != nil {
		return nil
	}
	res := []int64{}
	for _, category := range categories {
		if !category.CanSee(user) {
			res = append(res, category.Id)
		}
	}
	return res
}

// GetCategoriesWithStats adds the topics of the last week and month, for the categories page.
func GetCategoriesWithStats(user *User) ([]*Category, error) {
	categories, err := GetCategories(user)
	if err != nil {
		return nil, err
	}

	type row struct {
		CategoryId int64
		Count      int
	}
	count := func(since time.Time) (map[int64]int, error) {
		rows := []row{}
		err := adapter.Engine.Table("topic").Select("category_id, COUNT(*) AS count").
			Where("archetype = ? AND deleted = ? AND visible = ? AND created_at > ?", ArchetypeRegular, false, true, since).
			GroupBy("category_id").Find(&rows)
		res := map[int64]int{}
		for _, r := range rows {
			res[r.CategoryId] = r.Count
		}
		return res, err
	}
	week, err := count(time.Now().AddDate(0, 0, -7))
	if err != nil {
		return nil, err
	}
	month, err := count(time.Now().AddDate(0, -1, 0))
	if err != nil {
		return nil, err
	}

	res := []*Category{}
	for _, category := range categories {
		c := *category
		c.TopicsWeek = week[c.Id]
		c.TopicsMonth = month[c.Id]
		res = append(res, &c)
	}
	return res, nil
}

var categoryColors = []string{"0088CC", "12A89D", "F7941D", "BF1E2E", "3AB54A", "652D90", "ED207B", "25AAE2", "8C6238", "231F20", "9EB83B", "92278F"}

func normalizeCategory(category *Category) error {
	category.Name = strings.TrimSpace(category.Name)
	if category.Name == "" {
		return fmt.Errorf("Name can't be empty")
	}
	if category.Slug == "" {
		category.Slug = Slugify(category.Name)
	} else {
		category.Slug = Slugify(category.Slug)
	}
	category.Color = strings.TrimPrefix(strings.ToUpper(category.Color), "#")
	if category.Color == "" {
		category.Color = categoryColors[time.Now().Nanosecond()%len(categoryColors)]
	}
	category.TextColor = strings.TrimPrefix(strings.ToUpper(category.TextColor), "#")
	if category.TextColor == "" {
		category.TextColor = "FFFFFF"
	}

	existing, err := GetCategoryBySlug(category.Slug)
	if err != nil {
		return err
	}
	if existing != nil && existing.Id != category.Id {
		return fmt.Errorf("The slug %s is taken", category.Slug)
	}
	if category.ParentId == category.Id && category.Id != 0 {
		return fmt.Errorf("A category can't be its own parent")
	}
	if category.ParentId != 0 {
		parent, err := GetCategory(category.ParentId)
		if err != nil {
			return err
		}
		if parent == nil || parent.ParentId != 0 {
			return fmt.Errorf("The parent category must be a top-level category")
		}
	}
	return nil
}

func AddCategory(category *Category) error {
	err := normalizeCategory(category)
	if err != nil {
		return err
	}
	if category.Position == 0 {
		categories, err := loadCategories()
		if err != nil {
			return err
		}
		category.Position = len(categories) + 1
	}
	_, err = adapter.Engine.Insert(category)
	if err == nil {
		err = saveCategoryPermissions(category)
	}
	invalidateCategories()
	return err
}

func UpdateCategory(category *Category) error {
	err := normalizeCategory(category)
	if err != nil {
		return err
	}
	_, err = adapter.Engine.ID(category.Id).Cols("name", "slug", "description", "color", "text_color", "parent_id", "position",
		"read_restricted", "create_restricted", "topic_template").Update(category)
	if err == nil {
		err = saveCategoryPermissions(category)
	}
	invalidateCategories()
	return err
}

// DeleteCategory moves the category's topics to Uncategorized first.
func DeleteCategory(id int64) error {
	uncategorizedId := GetUncategorizedId()
	if id == uncategorizedId {
		return fmt.Errorf("Uncategorized can't be deleted")
	}
	category, err := GetCategory(id)
	if err != nil || category == nil {
		return err
	}
	if len(category.SubIds) > 0 {
		return fmt.Errorf("Delete or move its subcategories first")
	}

	_, err = adapter.Engine.Table("topic").Where("category_id = ?", id).Update(map[string]interface{}{"category_id": uncategorizedId})
	if err != nil {
		return err
	}
	_, err = adapter.Engine.ID(id).Delete(&Category{})
	if err != nil {
		return err
	}
	_, err = adapter.Engine.Where("category_id = ?", id).Delete(&CategoryGroup{})
	if err != nil {
		return err
	}
	invalidateCategories()
	return RefreshCategoryCounts()
}

// RefreshCategoryCounts recounts the topics and posts of every category.
func RefreshCategoryCounts() error {
	categories := []*Category{}
	err := adapter.Engine.Find(&categories)
	if err != nil {
		return err
	}
	for _, category := range categories {
		topicCount, err := adapter.Engine.Where("category_id = ? AND archetype = ? AND deleted = ?", category.Id, ArchetypeRegular, false).Count(&Topic{})
		if err != nil {
			return err
		}
		postCount, err := adapter.Engine.Table("post").Join("INNER", "topic", "topic.id = post.topic_id").
			Where("topic.category_id = ? AND topic.archetype = ? AND topic.deleted = ? AND post.deleted = ? AND post.post_type = ?",
				category.Id, ArchetypeRegular, false, false, PostTypeRegular).Count()
		if err != nil {
			return err
		}
		_, err = adapter.Engine.ID(category.Id).Cols("topic_count", "post_count").Update(&Category{TopicCount: int(topicCount), PostCount: int(postCount)})
		if err != nil {
			return err
		}
	}
	invalidateCategories()
	return nil
}

func incrCategory(categoryId int64, column string, delta int) {
	if categoryId == 0 {
		return
	}
	_, _ = adapter.Engine.ID(categoryId).Incr(column, delta).Update(&Category{})
	invalidateCategories()
}

func ReorderCategories(ids []int64) error {
	for i, id := range ids {
		_, err := adapter.Engine.ID(id).Cols("position").Update(&Category{Position: i + 1})
		if err != nil {
			return err
		}
	}
	invalidateCategories()
	return nil
}

func GetUncategorizedId() int64 {
	id := GetSiteSettingInt("uncategorized_category_id")
	return int64(id)
}

// SortCategoriesForDisplay puts each subcategory right after its parent.
func SortCategoriesForDisplay(categories []*Category) []*Category {
	res := []*Category{}
	parents := []*Category{}
	for _, category := range categories {
		if category.ParentId == 0 {
			parents = append(parents, category)
		}
	}
	sort.SliceStable(parents, func(i, j int) bool { return parents[i].Position < parents[j].Position })
	for _, parent := range parents {
		res = append(res, parent)
		for _, category := range categories {
			if category.ParentId == parent.Id {
				res = append(res, category)
			}
		}
	}
	return res
}
