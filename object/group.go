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
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/beego/beego/v2/core/logs"
	"github.com/xorm-io/builder"
)

// Who may see a group, as in Discourse.
const (
	GroupVisibilityPublic   = 0
	GroupVisibilityLoggedOn = 1
	GroupVisibilityMembers  = 2
	GroupVisibilityStaff    = 3
)

// Who may @mention or message a group, Discourse's alias levels.
const (
	GroupAliasNobody               = 0
	GroupAliasOnlyAdmins           = 1
	GroupAliasModsAndAdmins        = 2
	GroupAliasMembersModsAndAdmins = 3
	GroupAliasOwnersModsAndAdmins  = 4
	GroupAliasEveryone             = 99
)

// What a group may do in a restricted category.
const (
	CategoryPermissionFull       = 1
	CategoryPermissionCreatePost = 2
	CategoryPermissionReadonly   = 3
)

// The automatic groups, whose members follow the users' roles.
const (
	GroupAdmins     = "admins"
	GroupModerators = "moderators"
	GroupStaff      = "staff"
)

type Group struct {
	Id               int64     `xorm:"pk autoincr" json:"id"`
	Name             string    `xorm:"varchar(100) notnull unique" json:"name"`
	FullName         string    `xorm:"varchar(200)" json:"fullName"`
	BioRaw           string    `xorm:"mediumtext" json:"bioRaw,omitempty"`
	BioCooked        string    `xorm:"mediumtext" json:"bioCooked"`
	Automatic        bool      `json:"automatic"`
	Visibility       int       `json:"visibility"`
	MentionableLevel int       `json:"mentionableLevel"`
	MessageableLevel int       `json:"messageableLevel"`
	PublicJoin       bool      `json:"publicJoin"`
	PublicExit       bool      `json:"publicExit"`
	AllowRequests    bool      `json:"allowRequests"`
	Title            string    `xorm:"varchar(100)" json:"title"`
	UserCount        int       `json:"userCount"`
	CreatedAt        time.Time `xorm:"created" json:"createdAt"`

	IsMember     bool `xorm:"-" json:"isMember"`
	IsOwner      bool `xorm:"-" json:"isOwner"`
	Requested    bool `xorm:"-" json:"requested"`
	CanManage    bool `xorm:"-" json:"canManage"`
	CanMention   bool `xorm:"-" json:"canMention"`
	CanMessage   bool `xorm:"-" json:"canMessage"`
	RequestCount int  `xorm:"-" json:"requestCount"`
}

// "group" is a reserved word in SQL
func (Group) TableName() string {
	return "forum_group"
}

type GroupMember struct {
	Id        int64     `xorm:"pk autoincr" json:"id"`
	GroupId   int64     `xorm:"index unique(group_user)" json:"groupId"`
	UserId    int64     `xorm:"index unique(group_user)" json:"userId"`
	Owner     bool      `json:"owner"`
	CreatedAt time.Time `xorm:"created" json:"createdAt"`
}

type GroupRequest struct {
	Id        int64     `xorm:"pk autoincr" json:"id"`
	GroupId   int64     `xorm:"index unique(group_user)" json:"groupId"`
	UserId    int64     `xorm:"unique(group_user)" json:"userId"`
	Reason    string    `xorm:"varchar(500)" json:"reason"`
	CreatedAt time.Time `xorm:"created" json:"createdAt"`
}

// CategoryGroup lets a group into a restricted category.
type CategoryGroup struct {
	Id             int64 `xorm:"pk autoincr" json:"id"`
	CategoryId     int64 `xorm:"index" json:"categoryId"`
	GroupId        int64 `xorm:"index" json:"groupId"`
	PermissionType int   `json:"permissionType"`
}

// TopicAllowedGroup is a group a personal message was sent to.
type TopicAllowedGroup struct {
	Id      int64 `xorm:"pk autoincr" json:"id"`
	TopicId int64 `xorm:"index unique(topic_group)" json:"topicId"`
	GroupId int64 `xorm:"index unique(topic_group)" json:"groupId"`
}

// Every check of a category asks which groups the user is in, so the memberships are kept in memory.
var (
	membershipCache     map[int64][]int64
	membershipCacheLock sync.RWMutex
	groupWriteLock      sync.Mutex
)

func invalidateMemberships() {
	membershipCacheLock.Lock()
	membershipCache = nil
	membershipCacheLock.Unlock()
}

func GetUserGroupIds(userId int64) []int64 {
	if userId == 0 {
		return nil
	}
	membershipCacheLock.RLock()
	cache := membershipCache
	membershipCacheLock.RUnlock()
	if cache == nil {
		members := []*GroupMember{}
		err := adapter.Engine.Cols("group_id", "user_id").Find(&members)
		if err != nil {
			logs.Error("GetUserGroupIds(): %v", err)
			return nil
		}
		cache = map[int64][]int64{}
		for _, member := range members {
			cache[member.UserId] = append(cache[member.UserId], member.GroupId)
		}
		membershipCacheLock.Lock()
		membershipCache = cache
		membershipCacheLock.Unlock()
	}
	return cache[userId]
}

func IsGroupMember(groupId int64, user *User) bool {
	return user != nil && containsId(GetUserGroupIds(user.Id), groupId)
}

var groupNameRegex = regexp.MustCompile(`^[\w][\w.\-]{0,58}[\w]$|^[\w]$`)

func GetGroup(id int64) (*Group, error) {
	group := &Group{}
	has, err := adapter.Engine.ID(id).Get(group)
	if err != nil || !has {
		return nil, err
	}
	return group, nil
}

func GetGroupByName(name string) (*Group, error) {
	if name == "" {
		return nil, nil
	}
	group := &Group{}
	has, err := adapter.Engine.Where(builder.Expr("LOWER(name) = ?", strings.ToLower(name))).Get(group)
	if err != nil || !has {
		return nil, err
	}
	return group, nil
}

func GetGroupsByNames(names []string) ([]*Group, error) {
	groups := []*Group{}
	if len(names) == 0 {
		return groups, nil
	}
	lower := []string{}
	for _, name := range names {
		lower = append(lower, strings.ToLower(name))
	}
	err := adapter.Engine.Where(builder.In("LOWER(name)", lower)).Find(&groups)
	return groups, err
}

func getGroupMember(groupId int64, userId int64) (*GroupMember, error) {
	member := &GroupMember{}
	has, err := adapter.Engine.Where("group_id = ? AND user_id = ?", groupId, userId).Get(member)
	if err != nil || !has {
		return nil, err
	}
	return member, nil
}

func (g *Group) isOwner(user *User) bool {
	if user == nil {
		return false
	}
	member, err := getGroupMember(g.Id, user.Id)
	return err == nil && member != nil && member.Owner
}

func CanSeeGroup(user *User, group *Group) bool {
	if group == nil {
		return false
	}
	switch group.Visibility {
	case GroupVisibilityPublic:
		return true
	case GroupVisibilityLoggedOn:
		return user != nil
	case GroupVisibilityMembers:
		return user.IsStaff() || IsGroupMember(group.Id, user)
	default:
		return user.IsStaff()
	}
}

// CanManageGroup: admins manage every group, owners their own; automatic groups follow the roles and can't be changed.
func CanManageGroup(user *User, group *Group) bool {
	if user == nil || group == nil || group.Automatic {
		return false
	}
	return user.Admin || group.isOwner(user)
}

func checkAliasLevel(user *User, group *Group, level int) bool {
	if user == nil {
		return false
	}
	switch level {
	case GroupAliasEveryone:
		return true
	case GroupAliasOnlyAdmins:
		return user.Admin
	case GroupAliasModsAndAdmins:
		return user.IsStaff()
	case GroupAliasMembersModsAndAdmins:
		return user.IsStaff() || IsGroupMember(group.Id, user)
	case GroupAliasOwnersModsAndAdmins:
		return user.IsStaff() || group.isOwner(user)
	default:
		return false
	}
}

func CanMentionGroup(user *User, group *Group) bool {
	return checkAliasLevel(user, group, group.MentionableLevel)
}

func CanMessageGroup(user *User, group *Group) bool {
	return checkAliasLevel(user, group, group.MessageableLevel)
}

// fillGroupState sets what the user is to the group and may do with it.
func fillGroupState(user *User, groups []*Group) error {
	if user == nil || len(groups) == 0 {
		return nil
	}
	ids := []int64{}
	for _, group := range groups {
		ids = append(ids, group.Id)
	}
	members := []*GroupMember{}
	err := adapter.Engine.Where("user_id = ?", user.Id).In("group_id", ids).Find(&members)
	if err != nil {
		return err
	}
	requests := []*GroupRequest{}
	err = adapter.Engine.Where("user_id = ?", user.Id).In("group_id", ids).Find(&requests)
	if err != nil {
		return err
	}
	for _, group := range groups {
		for _, member := range members {
			if member.GroupId == group.Id {
				group.IsMember = true
				group.IsOwner = member.Owner
			}
		}
		for _, request := range requests {
			if request.GroupId == group.Id {
				group.Requested = true
			}
		}
		group.CanManage = !group.Automatic && (user.Admin || group.IsOwner)
		group.CanMention = CanMentionGroup(user, group)
		group.CanMessage = CanMessageGroup(user, group)
	}
	return nil
}

// GetGroups lists the groups the user may see: the automatic ones last.
func GetGroups(user *User, term string) ([]*Group, error) {
	groups := []*Group{}
	session := adapter.Engine.Asc("automatic").Asc("name")
	if term != "" {
		like := "%" + escapeLike(strings.ToLower(term)) + "%"
		session = session.Where(builder.Or(builder.Expr("LOWER(name) LIKE ?", like), builder.Expr("LOWER(full_name) LIKE ?", like)))
	}
	err := session.Find(&groups)
	if err != nil {
		return nil, err
	}
	res := []*Group{}
	for _, group := range groups {
		if CanSeeGroup(user, group) {
			group.BioRaw = ""
			res = append(res, group)
		}
	}
	err = fillGroupState(user, res)
	return res, err
}

func GetGroupForUser(user *User, name string) (*Group, error) {
	group, err := GetGroupByName(name)
	if err != nil {
		return nil, err
	}
	if group == nil || !CanSeeGroup(user, group) {
		return nil, fmt.Errorf("The group does not exist")
	}
	err = fillGroupState(user, []*Group{group})
	if err != nil {
		return nil, err
	}
	if group.CanManage {
		count, err := adapter.Engine.Where("group_id = ?", group.Id).Count(&GroupRequest{})
		if err != nil {
			return nil, err
		}
		group.RequestCount = int(count)
	} else {
		group.BioRaw = ""
	}
	return group, nil
}

type GroupMemberItem struct {
	User    *User     `json:"user"`
	Owner   bool      `json:"owner"`
	AddedAt time.Time `json:"addedAt"`
}

func GetGroupMembers(group *Group, term string, page int) ([]*GroupMemberItem, bool, error) {
	const pageSize = 50
	session := adapter.Engine.Where("group_id = ?", group.Id)
	if term != "" {
		// "user" is a reserved word in PostgreSQL, so the user table is searched on its own instead of joined
		like := "%" + escapeLike(strings.ToLower(term)) + "%"
		matching := []int64{}
		err := adapter.Engine.Table(&User{}).Cols("id").
			Where(builder.Or(builder.Expr("LOWER(username) LIKE ?", like), builder.Expr("LOWER(name) LIKE ?", like))).Limit(500).Find(&matching)
		if err != nil {
			return nil, false, err
		}
		session = session.In("user_id", append(matching, -1))
	}
	members := []*GroupMember{}
	err := session.Desc("owner").Asc("id").Limit(pageSize+1, page*pageSize).Find(&members)
	if err != nil {
		return nil, false, err
	}
	more := len(members) > pageSize
	if more {
		members = members[:pageSize]
	}
	ids := []int64{}
	for _, member := range members {
		ids = append(ids, member.UserId)
	}
	users, err := GetUserMap(ids)
	if err != nil {
		return nil, false, err
	}
	res := []*GroupMemberItem{}
	for _, member := range members {
		if user := users[member.UserId]; user != nil {
			res = append(res, &GroupMemberItem{User: user, Owner: member.Owner, AddedAt: member.CreatedAt})
		}
	}
	return res, more, nil
}

func GetGroupMemberIds(groupId int64) ([]int64, error) {
	ids := []int64{}
	err := adapter.Engine.Table("group_member").Cols("user_id").Where("group_id = ?", groupId).Find(&ids)
	return ids, err
}

func validateGroup(group *Group) error {
	group.Name = strings.TrimSpace(group.Name)
	if !groupNameRegex.MatchString(group.Name) {
		return fmt.Errorf("A group name has letters, digits, \"_\", \"-\" and \".\" only, and starts and ends with a letter or digit")
	}
	user, err := GetUserByUsername(group.Name)
	if err != nil {
		return err
	}
	if user != nil {
		return fmt.Errorf("%s is the name of a user", group.Name)
	}
	existing, err := GetGroupByName(group.Name)
	if err != nil {
		return err
	}
	if existing != nil && existing.Id != group.Id {
		return fmt.Errorf("The group %s already exists", group.Name)
	}
	if group.Visibility < GroupVisibilityPublic || group.Visibility > GroupVisibilityStaff {
		group.Visibility = GroupVisibilityPublic
	}
	group.BioCooked = CookMarkdown(group.BioRaw).Html
	return nil
}

func AddGroup(group *Group) error {
	group.Id = 0
	group.Automatic = false
	group.UserCount = 0
	err := validateGroup(group)
	if err != nil {
		return err
	}
	_, err = adapter.Engine.Insert(group)
	return err
}

func UpdateGroup(user *User, group *Group) error {
	existing, err := GetGroup(group.Id)
	if err != nil {
		return err
	}
	if existing == nil {
		return fmt.Errorf("The group does not exist")
	}
	if !user.Admin && !CanManageGroup(user, existing) {
		return fmt.Errorf("You can't change this group")
	}
	if existing.Automatic {
		// only how it is shown and who may mention or message it
		existing.FullName = group.FullName
		existing.BioRaw = group.BioRaw
		existing.BioCooked = CookMarkdown(group.BioRaw).Html
		existing.MentionableLevel = group.MentionableLevel
		existing.MessageableLevel = group.MessageableLevel
		_, err = adapter.Engine.ID(existing.Id).Cols("full_name", "bio_raw", "bio_cooked", "mentionable_level", "messageable_level").Update(existing)
		return err
	}
	if !user.Admin {
		// owners can't rename the group or change who may see it
		group.Name = existing.Name
		group.Visibility = existing.Visibility
	}
	err = validateGroup(group)
	if err != nil {
		return err
	}
	_, err = adapter.Engine.ID(group.Id).Cols("name", "full_name", "bio_raw", "bio_cooked", "visibility", "mentionable_level",
		"messageable_level", "public_join", "public_exit", "allow_requests", "title").Update(group)
	return err
}

func DeleteGroup(id int64) error {
	group, err := GetGroup(id)
	if err != nil {
		return err
	}
	if group == nil {
		return fmt.Errorf("The group does not exist")
	}
	if group.Automatic {
		return fmt.Errorf("Automatic groups can't be deleted")
	}
	for _, bean := range []interface{}{&GroupMember{}, &GroupRequest{}, &CategoryGroup{}, &TopicAllowedGroup{}} {
		_, err = adapter.Engine.Where("group_id = ?", id).Delete(bean)
		if err != nil {
			return err
		}
	}
	_, err = adapter.Engine.ID(id).Delete(&Group{})
	invalidateMemberships()
	invalidateCategories()
	return err
}

func refreshGroupUserCount(groupId int64) {
	count, err := adapter.Engine.Where("group_id = ?", groupId).Count(&GroupMember{})
	if err == nil {
		_, _ = adapter.Engine.ID(groupId).Cols("user_count").Update(&Group{UserCount: int(count)})
	}
}

func addGroupMember(group *Group, userId int64, owner bool) error {
	groupWriteLock.Lock()
	defer groupWriteLock.Unlock()
	member, err := getGroupMember(group.Id, userId)
	if err != nil {
		return err
	}
	if member == nil {
		_, err = adapter.Engine.Insert(&GroupMember{GroupId: group.Id, UserId: userId, Owner: owner})
	} else if owner && !member.Owner {
		_, err = adapter.Engine.ID(member.Id).Cols("owner").Update(&GroupMember{Owner: true})
	}
	if err != nil {
		return err
	}
	_, _ = adapter.Engine.Where("group_id = ? AND user_id = ?", group.Id, userId).Delete(&GroupRequest{})
	refreshGroupUserCount(group.Id)
	invalidateMemberships()
	return nil
}

// AddGroupMembers adds users by name, as members or owners.
func AddGroupMembers(actor *User, group *Group, usernames []string, owner bool) error {
	if !CanManageGroup(actor, group) {
		return fmt.Errorf("You can't change the members of this group")
	}
	users, err := GetUsersByUsernames(usernames)
	if err != nil {
		return err
	}
	if len(users) == 0 {
		return fmt.Errorf("No such users")
	}
	for _, user := range users {
		err = addGroupMember(group, user.Id, owner)
		if err != nil {
			return err
		}
	}
	return nil
}

func RemoveGroupMember(actor *User, group *Group, userId int64) error {
	self := actor != nil && actor.Id == userId
	if !CanManageGroup(actor, group) && !(self && group.PublicExit && !group.Automatic) {
		return fmt.Errorf("You can't remove members of this group")
	}
	_, err := adapter.Engine.Where("group_id = ? AND user_id = ?", group.Id, userId).Delete(&GroupMember{})
	if err != nil {
		return err
	}
	refreshGroupUserCount(group.Id)
	invalidateMemberships()
	return nil
}

func SetGroupOwner(actor *User, group *Group, userId int64, owner bool) error {
	if !CanManageGroup(actor, group) {
		return fmt.Errorf("You can't change the owners of this group")
	}
	_, err := adapter.Engine.Where("group_id = ? AND user_id = ?", group.Id, userId).Cols("owner").Update(&GroupMember{Owner: owner})
	return err
}

func JoinGroup(user *User, group *Group) error {
	if user == nil {
		return fmt.Errorf("Please sign in first")
	}
	if !group.PublicJoin || group.Automatic || !CanSeeGroup(user, group) {
		return fmt.Errorf("This group can't be joined freely")
	}
	return addGroupMember(group, user.Id, false)
}

// RequestGroupMembership asks the owners of the group to let the user in.
func RequestGroupMembership(user *User, group *Group, reason string) error {
	if user == nil {
		return fmt.Errorf("Please sign in first")
	}
	if !group.AllowRequests || group.Automatic || !CanSeeGroup(user, group) {
		return fmt.Errorf("This group doesn't take membership requests")
	}
	if IsGroupMember(group.Id, user) {
		return fmt.Errorf("You are already a member")
	}
	has, err := adapter.Engine.Where("group_id = ? AND user_id = ?", group.Id, user.Id).Exist(&GroupRequest{})
	if err != nil || has {
		return err
	}
	_, err = adapter.Engine.Insert(&GroupRequest{GroupId: group.Id, UserId: user.Id, Reason: truncateRunes(reason, 480)})
	if err != nil {
		return err
	}

	owners := []int64{}
	err = adapter.Engine.Table("group_member").Cols("user_id").Where("group_id = ? AND owner = ?", group.Id, true).Find(&owners)
	if err != nil {
		return err
	}
	if len(owners) == 0 {
		err = adapter.Engine.Table(&User{}).Cols("id").Where("admin = ?", true).Find(&owners)
		if err != nil {
			return err
		}
	}
	for _, owner := range owners {
		_ = addNotification(&Notification{
			UserId:           owner,
			NotificationType: NotificationMembershipRequest,
			ActingUserId:     user.Id,
			TopicTitle:       group.Name,
			Excerpt:          truncateRunes(reason, 200),
			Data:             toJson(map[string]interface{}{"groupName": group.Name}),
		}, nil)
	}
	return nil
}

type GroupRequestItem struct {
	User      *User     `json:"user"`
	Reason    string    `json:"reason"`
	CreatedAt time.Time `json:"createdAt"`
}

func GetGroupRequests(actor *User, group *Group) ([]*GroupRequestItem, error) {
	if !CanManageGroup(actor, group) {
		return nil, fmt.Errorf("You can't see the requests of this group")
	}
	requests := []*GroupRequest{}
	err := adapter.Engine.Where("group_id = ?", group.Id).Desc("id").Find(&requests)
	if err != nil {
		return nil, err
	}
	ids := []int64{}
	for _, request := range requests {
		ids = append(ids, request.UserId)
	}
	users, err := GetUserMap(ids)
	if err != nil {
		return nil, err
	}
	res := []*GroupRequestItem{}
	for _, request := range requests {
		if user := users[request.UserId]; user != nil {
			res = append(res, &GroupRequestItem{User: user, Reason: request.Reason, CreatedAt: request.CreatedAt})
		}
	}
	return res, nil
}

func HandleGroupRequest(actor *User, group *Group, userId int64, accept bool) error {
	if !CanManageGroup(actor, group) {
		return fmt.Errorf("You can't handle the requests of this group")
	}
	has, err := adapter.Engine.Where("group_id = ? AND user_id = ?", group.Id, userId).Exist(&GroupRequest{})
	if err != nil {
		return err
	}
	if !has {
		return fmt.Errorf("The request does not exist")
	}
	if !accept {
		_, err = adapter.Engine.Where("group_id = ? AND user_id = ?", group.Id, userId).Delete(&GroupRequest{})
		return err
	}
	err = addGroupMember(group, userId, false)
	if err != nil {
		return err
	}
	return addNotification(&Notification{
		UserId:           userId,
		NotificationType: NotificationMembershipAccepted,
		ActingUserId:     actor.Id,
		TopicTitle:       group.Name,
		Data:             toJson(map[string]interface{}{"groupName": group.Name}),
	}, nil)
}

// GetGroupPosts are the latest posts of the group's members, for its Activity tab.
func GetGroupPosts(viewer *User, group *Group, page int) ([]*UserPost, bool, error) {
	const pageSize = 30
	session := adapter.Engine.Table("post").Select("post.*").Join("INNER", "topic", "topic.id = post.topic_id").
		Join("INNER", "group_member", "group_member.user_id = post.user_id AND group_member.group_id = ?", group.Id).
		Where("post.deleted = ? AND post.post_type = ? AND topic.deleted = ? AND topic.archetype = ? AND topic.visible = ?",
			false, PostTypeRegular, false, ArchetypeRegular, true)
	hidden := GetHiddenCategoryIds(viewer)
	if len(hidden) > 0 {
		session = session.NotIn("topic.category_id", hidden)
	}
	posts := []*Post{}
	err := session.Desc("post.created_at").Limit(pageSize+1, page*pageSize).Find(&posts)
	if err != nil {
		return nil, false, err
	}
	more := len(posts) > pageSize
	if more {
		posts = posts[:pageSize]
	}
	res, err := attachTopics(posts)
	return res, more, err
}

// GetUserGroups are the groups shown on a profile: the ones the viewer may see.
func GetUserGroups(viewer *User, userId int64) ([]*Group, error) {
	ids := GetUserGroupIds(userId)
	groups := []*Group{}
	if len(ids) == 0 {
		return groups, nil
	}
	err := adapter.Engine.In("id", ids).Asc("name").Find(&groups)
	if err != nil {
		return nil, err
	}
	res := []*Group{}
	for _, group := range groups {
		if CanSeeGroup(viewer, group) {
			res = append(res, &Group{Id: group.Id, Name: group.Name, FullName: group.FullName, Automatic: group.Automatic, UserCount: group.UserCount})
		}
	}
	return res, nil
}

// GetTopicAllowedGroups are the groups a message was sent to.
func GetTopicAllowedGroups(topicId int64) ([]*Group, error) {
	ids := []int64{}
	err := adapter.Engine.Table("topic_allowed_group").Cols("group_id").Where("topic_id = ?", topicId).Find(&ids)
	if err != nil {
		return nil, err
	}
	groups := []*Group{}
	if len(ids) == 0 {
		return groups, nil
	}
	err = adapter.Engine.In("id", ids).Cols("id", "name", "full_name", "user_count").Find(&groups)
	return groups, err
}

func isTopicAllowedGroupMember(topicId int64, user *User) bool {
	groupIds := GetUserGroupIds(user.Id)
	if len(groupIds) == 0 {
		return false
	}
	has, err := adapter.Engine.Where("topic_id = ?", topicId).In("group_id", groupIds).Exist(&TopicAllowedGroup{})
	return err == nil && has
}

// getTopicGroupMemberIds are the members of the groups a message was sent to.
func getTopicGroupMemberIds(topicId int64) []int64 {
	ids := []int64{}
	_ = adapter.Engine.Table("group_member").Cols("user_id").
		Where(builder.In("group_id", builder.Select("group_id").From("topic_allowed_group").Where(builder.Eq{"topic_id": topicId}))).Find(&ids)
	return uniqueIds(ids)
}

// EnsureAutomaticGroups creates the automatic groups and fills them from the users' roles.
func EnsureAutomaticGroups() error {
	defs := []*Group{
		{Name: GroupAdmins, FullName: "Admins", Automatic: true, Visibility: GroupVisibilityPublic, MentionableLevel: GroupAliasModsAndAdmins, MessageableLevel: GroupAliasEveryone},
		{Name: GroupModerators, FullName: "Moderators", Automatic: true, Visibility: GroupVisibilityPublic, MentionableLevel: GroupAliasModsAndAdmins, MessageableLevel: GroupAliasEveryone},
		{Name: GroupStaff, FullName: "Staff", Automatic: true, Visibility: GroupVisibilityPublic, MentionableLevel: GroupAliasModsAndAdmins, MessageableLevel: GroupAliasModsAndAdmins},
	}
	for _, def := range defs {
		group, err := GetGroupByName(def.Name)
		if err != nil {
			return err
		}
		if group == nil {
			_, err = adapter.Engine.Insert(def)
			if err != nil {
				return err
			}
		}
	}
	return RefreshAutomaticGroups()
}

// RefreshAutomaticGroups makes the members of admins, moderators and staff match the users' roles.
func RefreshAutomaticGroups() error {
	groupWriteLock.Lock()
	defer groupWriteLock.Unlock()

	users := []*User{}
	err := adapter.Engine.Cols("id", "username", "admin", "moderator", "login_type", "casdoor_owner").Where(builder.Or(builder.Eq{"admin": true}, builder.Eq{"moderator": true})).Find(&users)
	if err != nil {
		return err
	}
	want := map[string][]int64{}
	for _, user := range users {
		if user.Username == SystemUsername {
			continue
		}
		if user.Admin || user.IsConfigAdmin() {
			want[GroupAdmins] = append(want[GroupAdmins], user.Id)
		}
		if user.Moderator {
			want[GroupModerators] = append(want[GroupModerators], user.Id)
		}
		want[GroupStaff] = append(want[GroupStaff], user.Id)
	}

	for _, name := range []string{GroupAdmins, GroupModerators, GroupStaff} {
		group, err := GetGroupByName(name)
		if err != nil {
			return err
		}
		if group == nil {
			continue
		}
		current, err := GetGroupMemberIds(group.Id)
		if err != nil {
			return err
		}
		for _, id := range want[name] {
			if !containsId(current, id) {
				_, err = adapter.Engine.Insert(&GroupMember{GroupId: group.Id, UserId: id})
				if err != nil {
					return err
				}
			}
		}
		for _, id := range current {
			if !containsId(want[name], id) {
				_, err = adapter.Engine.Where("group_id = ? AND user_id = ?", group.Id, id).Delete(&GroupMember{})
				if err != nil {
					return err
				}
			}
		}
		refreshGroupUserCount(group.Id)
	}
	invalidateMemberships()
	return nil
}

// SearchGroupsForMention are the groups the user may @mention or message, for the composer's suggestions.
func SearchGroupsForMention(user *User, term string, messaging bool) ([]*Group, error) {
	groups, err := GetGroups(user, strings.TrimPrefix(term, "@"))
	if err != nil {
		return nil, err
	}
	res := []*Group{}
	for _, group := range groups {
		if (messaging && group.CanMessage) || (!messaging && group.CanMention) {
			res = append(res, group)
		}
	}
	sort.SliceStable(res, func(i, j int) bool {
		return strings.HasPrefix(strings.ToLower(res[i].Name), strings.ToLower(term)) && !strings.HasPrefix(strings.ToLower(res[j].Name), strings.ToLower(term))
	})
	if len(res) > 5 {
		res = res[:5]
	}
	return res, nil
}
