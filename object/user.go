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
	"sync"
	"time"

	"github.com/casnode/casnode/conf"
	"github.com/xorm-io/builder"
)

type User struct {
	Id            int64     `xorm:"pk autoincr" json:"id"`
	Username      string    `xorm:"varchar(100) notnull unique" json:"username"`
	Name          string    `xorm:"varchar(100)" json:"name"`
	Avatar        string    `xorm:"varchar(500)" json:"avatar"`
	Email         string    `xorm:"varchar(200)" json:"email,omitempty"`
	CasdoorOwner  string    `xorm:"varchar(100)" json:"-"`
	Title         string    `xorm:"varchar(100)" json:"title"`
	Admin         bool      `json:"admin"`
	Moderator     bool      `json:"moderator"`
	TrustLevel    int       `json:"trustLevel"`
	BioRaw        string    `xorm:"mediumtext" json:"bioRaw,omitempty"`
	BioCooked     string    `xorm:"mediumtext" json:"bioCooked,omitempty"`
	Location      string    `xorm:"varchar(200)" json:"location,omitempty"`
	Website       string    `xorm:"varchar(500)" json:"website,omitempty"`
	Locale        string    `xorm:"varchar(20)" json:"locale,omitempty"`
	CreatedAt     time.Time `xorm:"created" json:"createdAt"`
	LastSeenAt    time.Time `json:"lastSeenAt"`
	LastPostedAt  time.Time `json:"lastPostedAt"`
	SuspendedTill time.Time `json:"suspendedTill"`
	SuspendReason string    `xorm:"varchar(500)" json:"suspendReason,omitempty"`
	SilencedTill  time.Time `json:"silencedTill"`
	// how often to email notifications, see email.go
	EmailLevel         int `json:"emailLevel,omitempty"`
	EmailMessagesLevel int `json:"emailMessagesLevel,omitempty"`

	TopicCount     int `json:"topicCount"`
	PostCount      int `json:"postCount"`
	LikesGiven     int `json:"likesGiven"`
	LikesReceived  int `json:"likesReceived"`
	DaysVisited    int `json:"daysVisited"`
	PostsReadCount int `json:"postsReadCount"`
	TopicsEntered  int `json:"topicsEntered"`
}

// UserVisit records the days a user visited, for "days visited".
type UserVisit struct {
	Id        int64  `xorm:"pk autoincr" json:"id"`
	UserId    int64  `xorm:"unique(user_day)" json:"userId"`
	VisitedAt string `xorm:"varchar(10) unique(user_day)" json:"visitedAt"`
}

func (u *User) IsStaff() bool {
	return u != nil && (u.Admin || u.Moderator)
}

func (u *User) IsSuspended() bool {
	return u != nil && u.SuspendedTill.After(time.Now())
}

func (u *User) IsSilenced() bool {
	return u != nil && u.SilencedTill.After(time.Now())
}

// Basic is what every page may show about a user.
func (u *User) Basic() *User {
	if u == nil {
		return nil
	}
	return &User{
		Id:         u.Id,
		Username:   u.Username,
		Name:       u.Name,
		Avatar:     u.Avatar,
		Title:      u.Title,
		Admin:      u.Admin,
		Moderator:  u.Moderator,
		TrustLevel: u.TrustLevel,
	}
}

// Public is the profile without what only the user and staff may see.
func (u *User) Public() *User {
	if u == nil {
		return nil
	}
	res := *u
	res.Email = ""
	res.BioRaw = ""
	res.EmailLevel = 0
	res.EmailMessagesLevel = 0
	res.SuspendReason = ""
	return &res
}

func GetUser(id int64) (*User, error) {
	user := &User{}
	has, err := adapter.Engine.ID(id).Get(user)
	if err != nil || !has {
		return nil, err
	}
	return user, nil
}

func GetUserByUsername(username string) (*User, error) {
	if username == "" {
		return nil, nil
	}
	user := &User{}
	has, err := adapter.Engine.Where(builder.Expr("LOWER(username) = ?", strings.ToLower(username))).Get(user)
	if err != nil || !has {
		return nil, err
	}
	return user, nil
}

func GetUsersByUsernames(usernames []string) ([]*User, error) {
	users := []*User{}
	if len(usernames) == 0 {
		return users, nil
	}
	lower := []string{}
	for _, username := range usernames {
		lower = append(lower, strings.ToLower(username))
	}
	err := adapter.Engine.Where(builder.In("LOWER(username)", lower)).Find(&users)
	return users, err
}

// GetUserMap loads the users by id, keyed by id, with only what every page may show.
func GetUserMap(ids []int64) (map[int64]*User, error) {
	res := map[int64]*User{}
	ids = uniqueIds(ids)
	if len(ids) == 0 {
		return res, nil
	}
	users := []*User{}
	err := adapter.Engine.In("id", ids).Find(&users)
	if err != nil {
		return nil, err
	}
	for _, user := range users {
		res[user.Id] = user.Basic()
	}
	return res, nil
}

func IsConfigAdmin(username string) bool {
	for _, name := range conf.GetConfigList("admins") {
		if strings.EqualFold(name, username) {
			return true
		}
	}
	return false
}

// SyncCasdoorUser creates the forum user for a Casdoor account on first sign-in, and refreshes its name and avatar later.
func SyncCasdoorUser(account *CasdoorAccount) (*User, error) {
	user, err := GetUserByUsername(account.Name)
	if err != nil {
		return nil, err
	}
	// someone of the same name elsewhere is not this account: the imported user keeps its posts under another name
	if user != nil && user.CasdoorOwner == ImportedUserOwner {
		err = renameImportedUser(user)
		if err != nil {
			return nil, err
		}
		user = nil
	}

	isAdmin := account.IsAdmin || IsConfigAdmin(account.Name)
	if user == nil {
		user = &User{
			Username:     account.Name,
			Name:         account.DisplayName,
			Avatar:       account.Avatar,
			Email:        account.Email,
			CasdoorOwner: account.Owner,
			Admin:        isAdmin,
			TrustLevel:   1,
			LastSeenAt:   time.Now(),
		}
		_, err = adapter.Engine.Insert(user)
		if err == nil && isAdmin {
			err = RefreshAutomaticGroups()
		}
		return user, err
	}

	user.Name = account.DisplayName
	user.Avatar = account.Avatar
	user.Email = account.Email
	user.CasdoorOwner = account.Owner
	if isAdmin {
		user.Admin = true
	}
	_, err = adapter.Engine.ID(user.Id).Cols("name", "avatar", "email", "casdoor_owner", "admin").Update(user)
	if err == nil && isAdmin {
		err = RefreshAutomaticGroups()
	}
	return user, err
}

// EnsureLocalUser is for devSignin: the user of that name, created when missing.
func EnsureLocalUser(username string) (*User, error) {
	user, err := GetUserByUsername(username)
	if err != nil || user != nil {
		return user, err
	}
	user = &User{
		Username:   username,
		Name:       strings.ToUpper(username[:1]) + username[1:],
		Admin:      IsConfigAdmin(username),
		TrustLevel: 1,
		LastSeenAt: time.Now(),
	}
	_, err = adapter.Engine.Insert(user)
	if err == nil && user.Admin {
		err = RefreshAutomaticGroups()
	}
	return user, err
}

func UpdateUserProfile(user *User) error {
	user.BioCooked = CookMarkdown(user.BioRaw).Html
	_, err := adapter.Engine.ID(user.Id).Cols("bio_raw", "bio_cooked", "location", "website", "title", "locale", "email_level", "email_messages_level").Update(user)
	if err == nil {
		TriggerBadgeGrant()
	}
	return err
}

func UpdateUserStaffFields(user *User) error {
	_, err := adapter.Engine.ID(user.Id).Cols("admin", "moderator", "trust_level", "title", "suspended_till", "suspend_reason", "silenced_till").Update(user)
	if err != nil {
		return err
	}
	TriggerBadgeGrant()
	return RefreshAutomaticGroups()
}

var (
	visitedToday     = map[int64]string{}
	visitedTodayLock sync.Mutex
)

// TouchUser records that the user is around: last seen, and today's visit.
func TouchUser(user *User) {
	now := time.Now()
	if now.Sub(user.LastSeenAt) >= time.Minute {
		user.LastSeenAt = now
		_, _ = adapter.Engine.ID(user.Id).Cols("last_seen_at").Update(&User{LastSeenAt: now})
	}

	day := now.Format("2006-01-02")
	visitedTodayLock.Lock()
	seen := visitedToday[user.Id] == day
	visitedToday[user.Id] = day
	visitedTodayLock.Unlock()
	if seen {
		return
	}
	has, err := adapter.Engine.Where("user_id = ? AND visited_at = ?", user.Id, day).Exist(&UserVisit{})
	if err != nil || has {
		return
	}
	_, err = adapter.Engine.Insert(&UserVisit{UserId: user.Id, VisitedAt: day})
	if err == nil {
		_, _ = adapter.Engine.ID(user.Id).Incr("days_visited").Update(&User{})
	}
}

func incrUser(userId int64, column string, delta int) {
	if userId == 0 {
		return
	}
	_, _ = adapter.Engine.ID(userId).Incr(column, delta).Update(&User{})
}

func SearchUsers(term string, limit int) ([]*User, error) {
	users := []*User{}
	term = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(term, "@")))
	session := adapter.Engine.Desc("last_seen_at").Limit(limit)
	if term != "" {
		like := "%" + escapeLike(term) + "%"
		session = session.Where(builder.Or(builder.Expr("LOWER(username) LIKE ?", like), builder.Expr("LOWER(name) LIKE ?", like)))
	}
	err := session.Find(&users)
	res := []*User{}
	for _, user := range users {
		res = append(res, user.Basic())
	}
	return res, err
}

type DirectoryItem struct {
	User          *User `json:"user"`
	LikesReceived int   `json:"likesReceived"`
	LikesGiven    int   `json:"likesGiven"`
	TopicCount    int   `json:"topicCount"`
	PostCount     int   `json:"postCount"`
	TopicsEntered int   `json:"topicsEntered"`
	PostsRead     int   `json:"postsRead"`
	DaysVisited   int   `json:"daysVisited"`
}

var directoryOrders = map[string]string{
	"likes_received": "likes_received",
	"likes_given":    "likes_given",
	"topic_count":    "topic_count",
	"post_count":     "post_count",
	"topics_entered": "topics_entered",
	"posts_read":     "posts_read_count",
	"days_visited":   "days_visited",
}

// GetDirectory lists the users for the Users page, sorted by one of its columns.
func GetDirectory(order string, asc bool, name string, page int) ([]*DirectoryItem, bool, error) {
	column, ok := directoryOrders[order]
	if !ok {
		column = "likes_received"
	}
	const pageSize = 50
	session := adapter.Engine.Limit(pageSize+1, page*pageSize)
	if asc {
		session = session.Asc(column)
	} else {
		session = session.Desc(column)
	}
	session = session.Asc("id")
	if name != "" {
		like := "%" + escapeLike(strings.ToLower(name)) + "%"
		session = session.Where(builder.Or(builder.Expr("LOWER(username) LIKE ?", like), builder.Expr("LOWER(name) LIKE ?", like)))
	}

	users := []*User{}
	err := session.Find(&users)
	if err != nil {
		return nil, false, err
	}
	more := len(users) > pageSize
	if more {
		users = users[:pageSize]
	}

	items := []*DirectoryItem{}
	for _, user := range users {
		items = append(items, &DirectoryItem{
			User:          user.Basic(),
			LikesReceived: user.LikesReceived,
			LikesGiven:    user.LikesGiven,
			TopicCount:    user.TopicCount,
			PostCount:     user.PostCount,
			TopicsEntered: user.TopicsEntered,
			PostsRead:     user.PostsReadCount,
			DaysVisited:   user.DaysVisited,
		})
	}
	return items, more, nil
}

func GetAdminUsers(filter string, name string, page int) ([]*User, bool, error) {
	const pageSize = 50
	users := []*User{}
	session := adapter.Engine.Limit(pageSize+1, page*pageSize)
	switch filter {
	case "staff":
		session = session.Where(builder.Or(builder.Eq{"admin": true}, builder.Eq{"moderator": true})).Asc("username")
	case "suspended":
		session = session.Where("suspended_till > ?", time.Now()).Desc("suspended_till")
	case "silenced":
		session = session.Where("silenced_till > ?", time.Now()).Desc("silenced_till")
	case "new":
		session = session.Desc("created_at")
	default:
		session = session.Desc("last_seen_at")
	}
	if name != "" {
		like := "%" + escapeLike(strings.ToLower(name)) + "%"
		session = session.Where(builder.Or(builder.Expr("LOWER(username) LIKE ?", like), builder.Expr("LOWER(name) LIKE ?", like), builder.Expr("LOWER(email) LIKE ?", like)))
	}
	err := session.Find(&users)
	if err != nil {
		return nil, false, err
	}
	more := len(users) > pageSize
	if more {
		users = users[:pageSize]
	}
	return users, more, nil
}

func CountUsers() (int64, error) {
	return adapter.Engine.Count(&User{})
}

func CountUsersSince(since time.Time) (int64, error) {
	session := adapter.Engine.Where("username <> ?", SystemUsername)
	if !since.IsZero() {
		session = session.And("created_at > ?", dbTime(since))
	}
	return session.Count(&User{})
}

func GetUsernameLink(user *User) string {
	if user == nil {
		return ""
	}
	return fmt.Sprintf("/u/%s", user.Username)
}
