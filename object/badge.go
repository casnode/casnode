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

	"github.com/beego/beego/v2/core/logs"
)

// Badge types, as in Discourse.
const (
	BadgeGold   = 1
	BadgeSilver = 2
	BadgeBronze = 3
)

type Badge struct {
	Id              int64     `xorm:"pk autoincr" json:"id"`
	Name            string    `xorm:"varchar(100) notnull unique" json:"name"`
	Code            string    `xorm:"varchar(50) index" json:"code"`
	Description     string    `xorm:"varchar(500)" json:"description"`
	LongDescription string    `xorm:"mediumtext" json:"longDescription"`
	Icon            string    `xorm:"varchar(50)" json:"icon"`
	ImageUrl        string    `xorm:"varchar(500)" json:"imageUrl"`
	BadgeType       int       `json:"badgeType"`
	BadgeGrouping   string    `xorm:"varchar(30)" json:"grouping"`
	MultipleGrant   bool      `json:"multipleGrant"`
	AllowTitle      bool      `json:"allowTitle"`
	Enabled         bool      `json:"enabled"`
	IsSystem        bool      `json:"system"`
	Backfilled      bool      `json:"-"`
	GrantCount      int       `json:"grantCount"`
	Position        int       `json:"position"`
	CreatedAt       time.Time `xorm:"created" json:"createdAt"`

	Granted bool `xorm:"-" json:"granted"`
}

type UserBadge struct {
	Id          int64     `xorm:"pk autoincr" json:"id"`
	BadgeId     int64     `xorm:"index unique(badge_user_post)" json:"badgeId"`
	UserId      int64     `xorm:"index unique(badge_user_post)" json:"userId"`
	PostId      int64     `xorm:"unique(badge_user_post)" json:"postId"`
	GrantedById int64     `json:"grantedById"`
	Reason      string    `xorm:"varchar(500)" json:"reason,omitempty"`
	GrantedAt   time.Time `xorm:"created index" json:"grantedAt"`
}

type badgeDef struct {
	badge *Badge
	// who earned it, and for which post when it is granted per post
	query func() ([]badgeGrant, error)
}

type badgeGrant struct {
	UserId int64
	PostId int64
}

func idsToGrants(ids []int64) []badgeGrant {
	res := []badgeGrant{}
	for _, id := range uniqueIds(ids) {
		res = append(res, badgeGrant{UserId: id})
	}
	return res
}

func likeActionUsers(actionTypes ...int) func() ([]badgeGrant, error) {
	return func() ([]badgeGrant, error) {
		ids := []int64{}
		err := adapter.Engine.Table("post_action").Distinct("user_id").In("action_type", actionTypes).And("deleted = ?", false).Find(&ids)
		return idsToGrants(ids), err
	}
}

func notificationActors(notificationType int) func() ([]badgeGrant, error) {
	return func() ([]badgeGrant, error) {
		ids := []int64{}
		err := adapter.Engine.Table("notification").Distinct("acting_user_id").Where("notification_type = ?", notificationType).Find(&ids)
		return idsToGrants(ids), err
	}
}

func userColumnAtLeast(column string, min int) func() ([]badgeGrant, error) {
	return func() ([]badgeGrant, error) {
		ids := []int64{}
		err := adapter.Engine.Table(&User{}).Cols("id").Where(column+" >= ?", min).Find(&ids)
		return idsToGrants(ids), err
	}
}

func likedPosts(min int, firstPost bool) func() ([]badgeGrant, error) {
	return func() ([]badgeGrant, error) {
		posts := []*Post{}
		session := adapter.Engine.Cols("id", "user_id").Where("deleted = ? AND post_type = ? AND like_count >= ?", false, PostTypeRegular, min)
		if firstPost {
			session = session.And("post_number = ?", 1)
		} else {
			session = session.And("post_number > ?", 1)
		}
		err := session.Find(&posts)
		res := []badgeGrant{}
		for _, post := range posts {
			res = append(res, badgeGrant{UserId: post.UserId, PostId: post.Id})
		}
		return res, err
	}
}

func postsWithLikes(likes int, posts int) func() ([]badgeGrant, error) {
	return func() ([]badgeGrant, error) {
		type row struct {
			UserId int64
			Count  int
		}
		rows := []*row{}
		err := adapter.Engine.Table("post").Select("user_id, COUNT(*) AS count").
			Where("deleted = ? AND post_type = ? AND like_count >= ?", false, PostTypeRegular, likes).GroupBy("user_id").Find(&rows)
		ids := []int64{}
		for _, r := range rows {
			if r.Count >= posts {
				ids = append(ids, r.UserId)
			}
		}
		return idsToGrants(ids), err
	}
}

// consecutiveVisits finds the users who came on days days in a row.
func consecutiveVisits(days int) func() ([]badgeGrant, error) {
	return func() ([]badgeGrant, error) {
		candidates := []int64{}
		err := adapter.Engine.Table(&User{}).Cols("id").Where("days_visited >= ?", days).Find(&candidates)
		if err != nil || len(candidates) == 0 {
			return nil, err
		}
		ids := []int64{}
		for _, userId := range candidates {
			visits := []string{}
			err = adapter.Engine.Table("user_visit").Cols("visited_at").Where("user_id = ?", userId).Asc("visited_at").Find(&visits)
			if err != nil {
				return nil, err
			}
			streak, best := 0, 0
			var previous time.Time
			for _, visit := range visits {
				day, err := time.Parse("2006-01-02", visit)
				if err != nil {
					continue
				}
				if !previous.IsZero() && day.Sub(previous) == 24*time.Hour {
					streak++
				} else {
					streak = 1
				}
				previous = day
				if streak > best {
					best = streak
				}
			}
			if best >= days {
				ids = append(ids, userId)
			}
		}
		return idsToGrants(ids), nil
	}
}

func anniversaryUsers() ([]badgeGrant, error) {
	yearAgo := dbTime(time.Now().AddDate(-1, 0, 0))
	ids := []int64{}
	err := adapter.Engine.Table(&User{}).Cols("id").Where("created_at < ? AND last_posted_at > ?", yearAgo, yearAgo).Find(&ids)
	return idsToGrants(ids), err
}

func editorUsers() ([]badgeGrant, error) {
	ids := []int64{}
	err := adapter.Engine.Table("post_revision").Distinct("user_id").Find(&ids)
	return idsToGrants(ids), err
}

func autobiographerUsers() ([]badgeGrant, error) {
	ids := []int64{}
	err := adapter.Engine.Table(&User{}).Cols("id").Where("bio_raw IS NOT NULL AND bio_raw <> ?", "").Find(&ids)
	return idsToGrants(ids), err
}

func chatUsers() ([]badgeGrant, error) {
	ids := []int64{}
	err := adapter.Engine.Table("chat_message").Distinct("user_id").Where("deleted = ?", false).Find(&ids)
	return idsToGrants(ids), err
}

// The system badges, a selection of Discourse's: names and descriptions are in English here and translated by the pages.
var badgeDefs = []*badgeDef{
	{badge: &Badge{Code: "basic", Name: "Basic", Icon: "user", BadgeType: BadgeBronze, BadgeGrouping: "trust_level", Description: "Granted all essential community functions"}, query: userColumnAtLeast("trust_level", 1)},
	{badge: &Badge{Code: "member", Name: "Member", Icon: "user", BadgeType: BadgeBronze, BadgeGrouping: "trust_level", Description: "Granted invitations, group messaging, more likes"}, query: userColumnAtLeast("trust_level", 2)},
	{badge: &Badge{Code: "regular", Name: "Regular", Icon: "user", BadgeType: BadgeSilver, BadgeGrouping: "trust_level", AllowTitle: true, Description: "Granted recategorize, rename, followed links, wiki, more likes"}, query: userColumnAtLeast("trust_level", 3)},
	{badge: &Badge{Code: "leader", Name: "Leader", Icon: "user", BadgeType: BadgeGold, BadgeGrouping: "trust_level", AllowTitle: true, Description: "Granted global edit, pin, close, archive, split and merge, more likes"}, query: userColumnAtLeast("trust_level", 4)},
	{badge: &Badge{Code: "autobiographer", Name: "Autobiographer", Icon: "file-lines", BadgeType: BadgeBronze, BadgeGrouping: "getting_started", Description: "Filled out profile information"}, query: autobiographerUsers},
	{badge: &Badge{Code: "editor", Name: "Editor", Icon: "pencil", BadgeType: BadgeBronze, BadgeGrouping: "getting_started", Description: "First post edit"}, query: editorUsers},
	{badge: &Badge{Code: "first_like", Name: "First Like", Icon: "heart", BadgeType: BadgeBronze, BadgeGrouping: "getting_started", Description: "Liked a post"}, query: likeActionUsers(PostActionLike)},
	{badge: &Badge{Code: "first_flag", Name: "First Flag", Icon: "flag", BadgeType: BadgeBronze, BadgeGrouping: "getting_started", Description: "Flagged a post"}, query: likeActionUsers(PostActionOffTopic, PostActionInappropriate, PostActionNotifyMods, PostActionSpam)},
	{badge: &Badge{Code: "first_mention", Name: "First Mention", Icon: "at", BadgeType: BadgeBronze, BadgeGrouping: "getting_started", Description: "Mentioned a user in a post"}, query: notificationActors(NotificationMentioned)},
	{badge: &Badge{Code: "first_quote", Name: "First Quote", Icon: "quote-left", BadgeType: BadgeBronze, BadgeGrouping: "getting_started", Description: "Quoted a post"}, query: notificationActors(NotificationQuoted)},
	{badge: &Badge{Code: "first_chat", Name: "First Chat", Icon: "comment", BadgeType: BadgeBronze, BadgeGrouping: "getting_started", Description: "Sent a chat message"}, query: chatUsers},
	{badge: &Badge{Code: "welcome", Name: "Welcome", Icon: "heart", BadgeType: BadgeBronze, BadgeGrouping: "community", Description: "Received a like"}, query: func() ([]badgeGrant, error) {
		ids := []int64{}
		err := adapter.Engine.Table("post").Cols("user_id").Where("like_count > ? AND deleted = ?", 0, false).Find(&ids)
		return idsToGrants(ids), err
	}},
	{badge: &Badge{Code: "thank_you", Name: "Thank You", Icon: "heart", BadgeType: BadgeBronze, BadgeGrouping: "community", Description: "Has given 20 likes"}, query: userColumnAtLeast("likes_given", 20)},
	{badge: &Badge{Code: "gives_back", Name: "Gives Back", Icon: "heart", BadgeType: BadgeSilver, BadgeGrouping: "community", Description: "Has given 100 likes"}, query: userColumnAtLeast("likes_given", 100)},
	{badge: &Badge{Code: "empathetic", Name: "Empathetic", Icon: "heart", BadgeType: BadgeGold, BadgeGrouping: "community", Description: "Has given 500 likes"}, query: userColumnAtLeast("likes_given", 500)},
	{badge: &Badge{Code: "appreciated", Name: "Appreciated", Icon: "heart", BadgeType: BadgeBronze, BadgeGrouping: "community", Description: "Received 1 like on 20 posts"}, query: postsWithLikes(1, 20)},
	{badge: &Badge{Code: "respected", Name: "Respected", Icon: "heart", BadgeType: BadgeSilver, BadgeGrouping: "community", Description: "Received 2 likes on 100 posts"}, query: postsWithLikes(2, 100)},
	{badge: &Badge{Code: "admired", Name: "Admired", Icon: "heart", BadgeType: BadgeGold, BadgeGrouping: "community", Description: "Received 5 likes on 300 posts"}, query: postsWithLikes(5, 300)},
	{badge: &Badge{Code: "anniversary", Name: "Anniversary", Icon: "clock", BadgeType: BadgeSilver, BadgeGrouping: "community", Description: "Active member for a year, posted at least once"}, query: anniversaryUsers},
	{badge: &Badge{Code: "enthusiast", Name: "Enthusiast", Icon: "calendar", BadgeType: BadgeBronze, BadgeGrouping: "community", Description: "Visited 10 consecutive days"}, query: consecutiveVisits(10)},
	{badge: &Badge{Code: "aficionado", Name: "Aficionado", Icon: "calendar", BadgeType: BadgeSilver, BadgeGrouping: "community", Description: "Visited 100 consecutive days"}, query: consecutiveVisits(100)},
	{badge: &Badge{Code: "devotee", Name: "Devotee", Icon: "calendar", BadgeType: BadgeGold, BadgeGrouping: "community", Description: "Visited 365 consecutive days"}, query: consecutiveVisits(365)},
	{badge: &Badge{Code: "nice_reply", Name: "Nice Reply", Icon: "reply", BadgeType: BadgeBronze, BadgeGrouping: "posting", MultipleGrant: true, Description: "Received 10 likes on a reply"}, query: likedPosts(10, false)},
	{badge: &Badge{Code: "good_reply", Name: "Good Reply", Icon: "reply", BadgeType: BadgeSilver, BadgeGrouping: "posting", MultipleGrant: true, Description: "Received 25 likes on a reply"}, query: likedPosts(25, false)},
	{badge: &Badge{Code: "great_reply", Name: "Great Reply", Icon: "reply", BadgeType: BadgeGold, BadgeGrouping: "posting", MultipleGrant: true, Description: "Received 50 likes on a reply"}, query: likedPosts(50, false)},
	{badge: &Badge{Code: "nice_topic", Name: "Nice Topic", Icon: "file-lines", BadgeType: BadgeBronze, BadgeGrouping: "posting", MultipleGrant: true, Description: "Received 10 likes on a topic"}, query: likedPosts(10, true)},
	{badge: &Badge{Code: "good_topic", Name: "Good Topic", Icon: "file-lines", BadgeType: BadgeSilver, BadgeGrouping: "posting", MultipleGrant: true, Description: "Received 25 likes on a topic"}, query: likedPosts(25, true)},
	{badge: &Badge{Code: "great_topic", Name: "Great Topic", Icon: "file-lines", BadgeType: BadgeGold, BadgeGrouping: "posting", MultipleGrant: true, Description: "Received 50 likes on a topic"}, query: likedPosts(50, true)},
}

// EnsureBadges adds the system badges that are missing.
func EnsureBadges() error {
	for i, def := range badgeDefs {
		has, err := adapter.Engine.Where("code = ? AND is_system = ?", def.badge.Code, true).Exist(&Badge{})
		if err != nil {
			return err
		}
		if has {
			continue
		}
		badge := *def.badge
		badge.IsSystem = true
		badge.Enabled = true
		badge.Position = i + 1
		existing, err := getBadgeByName(badge.Name)
		if err != nil {
			return err
		}
		if existing != nil {
			badge.Name = badge.Name + " (system)"
		}
		_, err = adapter.Engine.Insert(&badge)
		if err != nil {
			return err
		}
	}
	return nil
}

func getBadgeByName(name string) (*Badge, error) {
	badge := &Badge{}
	has, err := adapter.Engine.Where("name = ?", name).Get(badge)
	if err != nil || !has {
		return nil, err
	}
	return badge, nil
}

func GetBadge(id int64) (*Badge, error) {
	badge := &Badge{}
	has, err := adapter.Engine.ID(id).Get(badge)
	if err != nil || !has {
		return nil, err
	}
	return badge, nil
}

var (
	badgeTrigger   = make(chan struct{}, 1)
	badgeGrantLock sync.Mutex
)

// TriggerBadgeGrant asks for the badges to be granted soon, after something that may have earned one.
func TriggerBadgeGrant() {
	select {
	case badgeTrigger <- struct{}{}:
	default:
	}
}

func runBadgeWorker() {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()
	for {
		err := GrantBadges()
		if err != nil {
			logs.Error("GrantBadges(): %v", err)
		}
		select {
		case <-badgeTrigger:
			// a burst of likes or posts makes one run
			time.Sleep(15 * time.Second)
		case <-ticker.C:
		}
	}
}

// GrantBadges grants the system badges to whoever earned them. The first run of a badge grants it quietly, so a new
// badge on an old forum doesn't send everyone a notification.
func GrantBadges() error {
	badgeGrantLock.Lock()
	defer badgeGrantLock.Unlock()

	system, err := GetUserByUsername(SystemUsername)
	if err != nil {
		return err
	}
	for _, def := range badgeDefs {
		badge := &Badge{}
		has, err := adapter.Engine.Where("code = ? AND is_system = ?", def.badge.Code, true).Get(badge)
		if err != nil {
			return err
		}
		if !has || !badge.Enabled {
			continue
		}
		grants, err := def.query()
		if err != nil {
			return fmt.Errorf("badge %s: %v", badge.Code, err)
		}
		existing := []*UserBadge{}
		err = adapter.Engine.Cols("user_id", "post_id").Where("badge_id = ?", badge.Id).Find(&existing)
		if err != nil {
			return err
		}
		have := map[string]bool{}
		for _, ub := range existing {
			have[fmt.Sprintf("%d/%d", ub.UserId, ub.PostId)] = true
		}
		added := 0
		for _, grant := range grants {
			if grant.UserId == 0 || (system != nil && grant.UserId == system.Id) {
				continue
			}
			if !badge.MultipleGrant {
				grant.PostId = 0
			}
			key := fmt.Sprintf("%d/%d", grant.UserId, grant.PostId)
			if have[key] {
				continue
			}
			have[key] = true
			err = grantBadge(badge, grant.UserId, grant.PostId, 0, "", badge.Backfilled)
			if err != nil {
				return err
			}
			added++
		}
		if !badge.Backfilled || added > 0 {
			badge.Backfilled = true
			_, err = adapter.Engine.ID(badge.Id).Cols("backfilled").Update(badge)
			if err != nil {
				return err
			}
			refreshBadgeGrantCount(badge.Id)
		}
	}
	return nil
}

func refreshBadgeGrantCount(badgeId int64) {
	count, err := adapter.Engine.Where("badge_id = ?", badgeId).Count(&UserBadge{})
	if err == nil {
		_, _ = adapter.Engine.ID(badgeId).Cols("grant_count").Update(&Badge{GrantCount: int(count)})
	}
}

func grantBadge(badge *Badge, userId int64, postId int64, grantedById int64, reason string, notify bool) error {
	userBadge := &UserBadge{BadgeId: badge.Id, UserId: userId, PostId: postId, GrantedById: grantedById, Reason: reason}
	_, err := adapter.Engine.Insert(userBadge)
	if err != nil {
		return err
	}
	if !notify {
		return nil
	}
	notification := &Notification{
		UserId:           userId,
		NotificationType: NotificationGrantedBadge,
		TopicTitle:       badge.Name,
		Data:             toJson(map[string]interface{}{"badgeId": badge.Id, "badgeName": badge.Name, "badgeCode": badge.Code, "badgeIcon": badge.Icon, "badgeType": badge.BadgeType}),
	}
	if postId != 0 {
		if post, err := GetPost(postId); err == nil && post != nil {
			notification.TopicId = post.TopicId
			notification.PostNumber = post.PostNumber
		}
	}
	return addNotification(notification, nil)
}

// GetBadges are the enabled badges, or all of them for the admin page, marking the ones the user has.
func GetBadges(user *User, all bool) ([]*Badge, error) {
	badges := []*Badge{}
	session := adapter.Engine.Asc("badge_type").Asc("position").Asc("id")
	if !all {
		session = session.Where("enabled = ?", true)
	}
	err := session.Find(&badges)
	if err != nil || user == nil {
		return badges, err
	}
	granted := []int64{}
	err = adapter.Engine.Table("user_badge").Distinct("badge_id").Where("user_id = ?", user.Id).Find(&granted)
	for _, badge := range badges {
		badge.Granted = containsId(granted, badge.Id)
	}
	return badges, err
}

type BadgeGrantItem struct {
	Id        int64     `json:"id"`
	User      *User     `json:"user"`
	GrantedAt time.Time `json:"grantedAt"`
	Topic     *Topic    `json:"topic,omitempty"`
	Post      *Post     `json:"post,omitempty"`
	Reason    string    `json:"reason,omitempty"`
}

// GetBadgeGrants lists who has a badge, newest first; with username, only that user's grants.
func GetBadgeGrants(viewer *User, badgeId int64, userId int64, page int) ([]*BadgeGrantItem, bool, error) {
	const pageSize = 48
	rows := []*UserBadge{}
	session := adapter.Engine.Where("badge_id = ?", badgeId)
	if userId != 0 {
		session = session.And("user_id = ?", userId)
	}
	err := session.Desc("granted_at").Desc("id").Limit(pageSize+1, page*pageSize).Find(&rows)
	if err != nil {
		return nil, false, err
	}
	more := len(rows) > pageSize
	if more {
		rows = rows[:pageSize]
	}
	return makeGrantItems(viewer, rows, more)
}

func makeGrantItems(viewer *User, rows []*UserBadge, more bool) ([]*BadgeGrantItem, bool, error) {
	userIds := []int64{}
	postIds := []int64{}
	for _, row := range rows {
		userIds = append(userIds, row.UserId)
		if row.PostId != 0 {
			postIds = append(postIds, row.PostId)
		}
	}
	users, err := GetUserMap(userIds)
	if err != nil {
		return nil, false, err
	}
	posts := []*Post{}
	if len(postIds) > 0 {
		err = adapter.Engine.In("id", postIds).Omit("raw", "cooked").Find(&posts)
		if err != nil {
			return nil, false, err
		}
	}
	withTopics, err := attachTopics(posts)
	if err != nil {
		return nil, false, err
	}
	res := []*BadgeGrantItem{}
	for _, row := range rows {
		item := &BadgeGrantItem{Id: row.Id, User: users[row.UserId], GrantedAt: row.GrantedAt, Reason: row.Reason}
		for _, p := range withTopics {
			if p.Post.Id == row.PostId && p.Topic != nil {
				canSee, err := CanSeeTopic(viewer, p.Topic)
				if err == nil && canSee {
					item.Post = p.Post
					item.Topic = p.Topic
				}
			}
		}
		if item.User != nil {
			res = append(res, item)
		}
	}
	return res, more, nil
}

type UserBadgeSummary struct {
	Badge     *Badge    `json:"badge"`
	Count     int       `json:"count"`
	GrantedAt time.Time `json:"grantedAt"`
}

// GetUserBadges are the badges of a user, once each with how many times it was granted; the best first.
func GetUserBadges(userId int64) ([]*UserBadgeSummary, error) {
	rows := []*UserBadge{}
	err := adapter.Engine.Where("user_id = ?", userId).Desc("granted_at").Find(&rows)
	if err != nil {
		return nil, err
	}
	badgeIds := []int64{}
	for _, row := range rows {
		badgeIds = append(badgeIds, row.BadgeId)
	}
	badges := []*Badge{}
	if len(badgeIds) > 0 {
		err = adapter.Engine.In("id", uniqueIds(badgeIds)).And("enabled = ?", true).Find(&badges)
		if err != nil {
			return nil, err
		}
	}
	summaries := map[int64]*UserBadgeSummary{}
	res := []*UserBadgeSummary{}
	for _, row := range rows {
		summary := summaries[row.BadgeId]
		if summary == nil {
			for _, badge := range badges {
				if badge.Id == row.BadgeId {
					summary = &UserBadgeSummary{Badge: badge, GrantedAt: row.GrantedAt}
				}
			}
			if summary == nil {
				continue
			}
			summaries[row.BadgeId] = summary
			res = append(res, summary)
		}
		summary.Count++
	}
	sort.SliceStable(res, func(i, j int) bool {
		if res[i].Badge.BadgeType != res[j].Badge.BadgeType {
			return res[i].Badge.BadgeType < res[j].Badge.BadgeType
		}
		return res[i].GrantedAt.After(res[j].GrantedAt)
	})
	return res, nil
}

// GetBadgeTitles are the titles a user may pick: the names of their badges that allow it, and of their groups' titles.
func GetBadgeTitles(userId int64) ([]string, error) {
	summaries, err := GetUserBadges(userId)
	if err != nil {
		return nil, err
	}
	titles := []string{}
	for _, summary := range summaries {
		if summary.Badge.AllowTitle {
			titles = append(titles, summary.Badge.Name)
		}
	}
	groupIds := GetUserGroupIds(userId)
	if len(groupIds) > 0 {
		groupTitles := []string{}
		err = adapter.Engine.Table("forum_group").Cols("title").In("id", groupIds).And("title <> ?", "").Find(&groupTitles)
		if err != nil {
			return nil, err
		}
		titles = append(titles, groupTitles...)
	}
	return titles, nil
}

func validateBadge(badge *Badge) error {
	badge.Name = strings.TrimSpace(badge.Name)
	if badge.Name == "" {
		return fmt.Errorf("Name can't be empty")
	}
	if badge.BadgeType < BadgeGold || badge.BadgeType > BadgeBronze {
		badge.BadgeType = BadgeBronze
	}
	existing, err := getBadgeByName(badge.Name)
	if err != nil {
		return err
	}
	if existing != nil && existing.Id != badge.Id {
		return fmt.Errorf("The badge %s already exists", badge.Name)
	}
	return nil
}

func AddBadge(badge *Badge) error {
	badge.Id = 0
	badge.IsSystem = false
	badge.Code = ""
	badge.GrantCount = 0
	badge.Backfilled = true
	if badge.BadgeGrouping == "" {
		badge.BadgeGrouping = "other"
	}
	err := validateBadge(badge)
	if err != nil {
		return err
	}
	_, err = adapter.Engine.Insert(badge)
	return err
}

// UpdateBadge changes a badge; of a system badge only whether it is enabled and its picture.
func UpdateBadge(badge *Badge) error {
	existing, err := GetBadge(badge.Id)
	if err != nil {
		return err
	}
	if existing == nil {
		return fmt.Errorf("The badge does not exist")
	}
	if existing.IsSystem {
		_, err = adapter.Engine.ID(badge.Id).Cols("enabled", "icon", "image_url", "allow_title").Update(badge)
		return err
	}
	err = validateBadge(badge)
	if err != nil {
		return err
	}
	_, err = adapter.Engine.ID(badge.Id).Cols("name", "description", "long_description", "icon", "image_url", "badge_type",
		"badge_grouping", "multiple_grant", "allow_title", "enabled").Update(badge)
	return err
}

func DeleteBadge(id int64) error {
	badge, err := GetBadge(id)
	if err != nil {
		return err
	}
	if badge == nil {
		return fmt.Errorf("The badge does not exist")
	}
	if badge.IsSystem {
		return fmt.Errorf("System badges can't be deleted, disable them instead")
	}
	_, err = adapter.Engine.Where("badge_id = ?", id).Delete(&UserBadge{})
	if err != nil {
		return err
	}
	_, err = adapter.Engine.ID(id).Delete(&Badge{})
	return err
}

// GrantBadgeManually is an admin giving a badge to a user.
func GrantBadgeManually(admin *User, badgeId int64, username string, reason string) error {
	badge, err := GetBadge(badgeId)
	if err != nil {
		return err
	}
	if badge == nil || !badge.Enabled {
		return fmt.Errorf("The badge does not exist or is disabled")
	}
	user, err := GetUserByUsername(username)
	if err != nil {
		return err
	}
	if user == nil {
		return fmt.Errorf("The user %s does not exist", username)
	}
	has, err := adapter.Engine.Where("badge_id = ? AND user_id = ? AND post_id = ?", badge.Id, user.Id, 0).Exist(&UserBadge{})
	if err != nil {
		return err
	}
	if has {
		return fmt.Errorf("%s already has this badge", user.Username)
	}
	err = grantBadge(badge, user.Id, 0, admin.Id, truncateRunes(reason, 480), true)
	if err != nil {
		return err
	}
	refreshBadgeGrantCount(badge.Id)
	return nil
}

func RevokeBadge(userBadgeId int64) error {
	userBadge := &UserBadge{}
	has, err := adapter.Engine.ID(userBadgeId).Get(userBadge)
	if err != nil {
		return err
	}
	if !has {
		return fmt.Errorf("The grant does not exist")
	}
	_, err = adapter.Engine.ID(userBadgeId).Delete(&UserBadge{})
	if err != nil {
		return err
	}
	refreshBadgeGrantCount(userBadge.BadgeId)
	return nil
}
