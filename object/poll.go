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
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	xhtml "golang.org/x/net/html"
)

// Polls are written like Discourse's:
//
//	[poll type=multiple min=1 max=2 public=true results=on_vote close=2026-12-31T00:00:00Z]
//	* first option
//	* second option
//	[/poll]
//
// The cooked post keeps the options as a plain list, which the pages turn into the poll; the votes are in the tables.

const (
	PollTypeRegular  = "regular"
	PollTypeMultiple = "multiple"
	PollOpen         = "open"
	PollClosed       = "closed"
)

type Poll struct {
	Id        int64     `xorm:"pk autoincr" json:"id"`
	PostId    int64     `xorm:"index unique(post_name)" json:"postId"`
	Name      string    `xorm:"varchar(100) unique(post_name)" json:"name"`
	Type      string    `xorm:"varchar(20)" json:"type"`
	Status    string    `xorm:"varchar(20)" json:"status"`
	Results   string    `xorm:"varchar(20)" json:"results"`
	IsPublic  bool      `json:"public"`
	Min       int       `json:"min"`
	Max       int       `json:"max"`
	ChartType string    `xorm:"varchar(20)" json:"chartType"`
	CloseAt   time.Time `json:"closeAt"`
	CreatedAt time.Time `xorm:"created" json:"createdAt"`

	Options       []*PollOption `xorm:"-" json:"options"`
	Voters        int           `xorm:"-" json:"voters"`
	UserVotes     []string      `xorm:"-" json:"userVotes"`
	ResultsHidden bool          `xorm:"-" json:"resultsHidden"`
	CanClose      bool          `xorm:"-" json:"canClose"`
}

type PollOption struct {
	Id       int64  `xorm:"pk autoincr" json:"-"`
	PollId   int64  `xorm:"index" json:"-"`
	Digest   string `xorm:"varchar(32)" json:"id"`
	Html     string `xorm:"mediumtext" json:"html"`
	Position int    `json:"-"`
	Votes    int    `xorm:"-" json:"votes"`
}

type PollVote struct {
	Id        int64     `xorm:"pk autoincr" json:"id"`
	PollId    int64     `xorm:"index unique(poll_option_user)" json:"pollId"`
	OptionId  int64     `xorm:"unique(poll_option_user)" json:"optionId"`
	UserId    int64     `xorm:"index unique(poll_option_user)" json:"userId"`
	CreatedAt time.Time `xorm:"created" json:"createdAt"`
}

type ParsedPoll struct {
	Name      string
	Type      string
	Results   string
	Public    bool
	Min       int
	Max       int
	ChartType string
	CloseAt   time.Time
	Status    string
	Options   []*PollOption
}

var (
	pollOpenRegex  = regexp.MustCompile(`(?m)^\[poll((?:\s+[\w-]+=(?:"[^"\n]*"|[^\s\]]+))*)\s*\]\s*$`)
	pollCloseRegex = regexp.MustCompile(`(?m)^\[/poll\]\s*$`)
	pollAttrRegex  = regexp.MustCompile(`([\w-]+)=(?:"([^"\n]*)"|([^\s\]]+))`)
	pollNameRegex  = regexp.MustCompile(`^[\w.\-]{1,100}$`)
	pollVoteLock   sync.Mutex
)

// expandPolls turns each [poll] block into a div the sanitizer keeps, with its options still Markdown, and returns
// the polls' settings in order.
func expandPolls(raw string) (string, []*ParsedPoll) {
	polls := []*ParsedPoll{}
	if !strings.Contains(raw, "[poll") || !GetSiteSettingBool("poll_enabled") {
		return raw, polls
	}
	names := map[string]bool{}
	raw = pollOpenRegex.ReplaceAllStringFunc(raw, func(s string) string {
		attrs := map[string]string{}
		for _, m := range pollAttrRegex.FindAllStringSubmatch(pollOpenRegex.FindStringSubmatch(s)[1], -1) {
			value := m[2]
			if value == "" {
				value = m[3]
			}
			attrs[strings.ToLower(m[1])] = value
		}
		poll := &ParsedPoll{Name: attrs["name"], Type: PollTypeRegular, Results: "always", ChartType: "bar", Status: PollOpen}
		if !pollNameRegex.MatchString(poll.Name) {
			poll.Name = "poll"
		}
		// every poll of a post needs its own name
		base := poll.Name
		for i := 2; names[poll.Name]; i++ {
			poll.Name = fmt.Sprintf("%s-%d", base, i)
		}
		names[poll.Name] = true
		if attrs["type"] == PollTypeMultiple {
			poll.Type = PollTypeMultiple
		}
		switch attrs["results"] {
		case "on_vote", "on_close", "staff_only":
			poll.Results = attrs["results"]
		}
		poll.Public = attrs["public"] == "true"
		poll.Min, _ = strconv.Atoi(attrs["min"])
		poll.Max, _ = strconv.Atoi(attrs["max"])
		if attrs["chartType"] == "pie" || attrs["charttype"] == "pie" {
			poll.ChartType = "pie"
		}
		if attrs["status"] == PollClosed {
			poll.Status = PollClosed
		}
		if attrs["close"] != "" {
			if at, err := time.Parse(time.RFC3339, attrs["close"]); err == nil {
				poll.CloseAt = at
			} else if at, err := time.ParseInLocation("2006-01-02", attrs["close"], time.Local); err == nil {
				poll.CloseAt = at
			}
		}
		polls = append(polls, poll)
		return fmt.Sprintf("<div class=\"poll\" data-poll-name=\"%s\">\n\n", html.EscapeString(poll.Name))
	})
	raw = pollCloseRegex.ReplaceAllString(raw, "\n\n</div>\n")
	return raw, polls
}

// collectPollOptions reads each poll's options from its list in the cooked HTML.
func collectPollOptions(cooked string, polls []*ParsedPoll) []*ParsedPoll {
	if len(polls) == 0 {
		return polls
	}
	doc, err := xhtml.Parse(strings.NewReader(cooked))
	if err != nil {
		return nil
	}
	byName := map[string]*ParsedPoll{}
	for _, poll := range polls {
		byName[poll.Name] = poll
	}

	var walk func(n *xhtml.Node)
	walk = func(n *xhtml.Node) {
		if n.Type == xhtml.ElementNode && n.Data == "div" {
			for _, attr := range n.Attr {
				if attr.Key != "data-poll-name" {
					continue
				}
				poll := byName[attr.Val]
				if poll == nil {
					continue
				}
				list := findElement(n, "ul")
				if ordered := findElement(n, "ol"); list == nil {
					list = ordered
				}
				if list == nil {
					continue
				}
				seen := map[string]bool{}
				for li := list.FirstChild; li != nil; li = li.NextSibling {
					if li.Type != xhtml.ElementNode || li.Data != "li" {
						continue
					}
					var buf bytes.Buffer
					for c := li.FirstChild; c != nil; c = c.NextSibling {
						_ = xhtml.Render(&buf, c)
					}
					optionHtml := strings.TrimSpace(buf.String())
					sum := md5.Sum([]byte(optionHtml))
					digest := hex.EncodeToString(sum[:])
					if optionHtml == "" || seen[digest] {
						continue
					}
					seen[digest] = true
					poll.Options = append(poll.Options, &PollOption{Digest: digest, Html: optionHtml, Position: len(poll.Options)})
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	res := []*ParsedPoll{}
	for _, poll := range polls {
		if len(poll.Options) > 0 {
			res = append(res, poll)
		}
	}
	return res
}

func validatePolls(polls []*ParsedPoll) error {
	for _, poll := range polls {
		if len(poll.Options) < 2 {
			return fmt.Errorf("A poll needs at least 2 options")
		}
		if len(poll.Options) > 50 {
			return fmt.Errorf("A poll can have at most 50 options")
		}
	}
	return nil
}

// SyncPolls makes the polls of a post match its text: the votes for options that are still there are kept.
func SyncPolls(postId int64, parsed []*ParsedPoll) error {
	pollVoteLock.Lock()
	defer pollVoteLock.Unlock()

	existing := []*Poll{}
	err := adapter.Engine.Where("post_id = ?", postId).Find(&existing)
	if err != nil {
		return err
	}
	keep := map[int64]bool{}
	for _, p := range parsed {
		var poll *Poll
		for _, e := range existing {
			if e.Name == p.Name {
				poll = e
			}
		}
		if poll == nil {
			poll = &Poll{PostId: postId, Name: p.Name}
		}
		poll.Type, poll.Results, poll.IsPublic, poll.Min, poll.Max, poll.ChartType, poll.CloseAt = p.Type, p.Results, p.Public, p.Min, p.Max, p.ChartType, p.CloseAt
		if poll.Id == 0 {
			poll.Status = p.Status
			_, err = adapter.Engine.Insert(poll)
		} else {
			_, err = adapter.Engine.ID(poll.Id).Cols("type", "results", "is_public", "min", "max", "chart_type", "close_at").Update(poll)
		}
		if err != nil {
			return err
		}
		keep[poll.Id] = true

		options := []*PollOption{}
		err = adapter.Engine.Where("poll_id = ?", poll.Id).Find(&options)
		if err != nil {
			return err
		}
		wanted := map[string]bool{}
		for _, option := range p.Options {
			wanted[option.Digest] = true
			var found *PollOption
			for _, o := range options {
				if o.Digest == option.Digest {
					found = o
				}
			}
			if found == nil {
				_, err = adapter.Engine.Insert(&PollOption{PollId: poll.Id, Digest: option.Digest, Html: option.Html, Position: option.Position})
			} else {
				_, err = adapter.Engine.ID(found.Id).Cols("html", "position").Update(&PollOption{Html: option.Html, Position: option.Position})
			}
			if err != nil {
				return err
			}
		}
		for _, o := range options {
			if !wanted[o.Digest] {
				_, err = adapter.Engine.Where("option_id = ?", o.Id).Delete(&PollVote{})
				if err != nil {
					return err
				}
				_, err = adapter.Engine.ID(o.Id).Delete(&PollOption{})
				if err != nil {
					return err
				}
			}
		}
	}
	for _, e := range existing {
		if !keep[e.Id] {
			err = deletePoll(e.Id)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func deletePoll(pollId int64) error {
	_, err := adapter.Engine.Where("poll_id = ?", pollId).Delete(&PollVote{})
	if err != nil {
		return err
	}
	_, err = adapter.Engine.Where("poll_id = ?", pollId).Delete(&PollOption{})
	if err != nil {
		return err
	}
	_, err = adapter.Engine.ID(pollId).Delete(&Poll{})
	return err
}

func (p *Poll) isClosed() bool {
	return p.Status == PollClosed || (!p.CloseAt.IsZero() && p.CloseAt.Year() > 1 && p.CloseAt.Before(time.Now()))
}

// GetPostPolls loads the polls of the posts with their counts, and what the user voted.
func GetPostPolls(user *User, posts []*Post) (map[int64][]*Poll, error) {
	res := map[int64][]*Poll{}
	postIds := []int64{}
	authors := map[int64]int64{}
	for _, post := range posts {
		if strings.Contains(post.Cooked, "data-poll-name") {
			postIds = append(postIds, post.Id)
			authors[post.Id] = post.UserId
		}
	}
	if len(postIds) == 0 {
		return res, nil
	}

	polls := []*Poll{}
	err := adapter.Engine.In("post_id", postIds).Asc("id").Find(&polls)
	if err != nil || len(polls) == 0 {
		return res, err
	}
	pollIds := []int64{}
	for _, poll := range polls {
		pollIds = append(pollIds, poll.Id)
	}
	options := []*PollOption{}
	err = adapter.Engine.In("poll_id", pollIds).Asc("position").Find(&options)
	if err != nil {
		return nil, err
	}

	type count struct {
		OptionId int64
		Count    int
	}
	counts := []*count{}
	err = adapter.Engine.Table("poll_vote").Select("option_id, COUNT(*) AS count").In("poll_id", pollIds).GroupBy("option_id").Find(&counts)
	if err != nil {
		return nil, err
	}
	type voters struct {
		PollId int64
		Count  int
	}
	voterCounts := []*voters{}
	err = adapter.Engine.Table("poll_vote").Select("poll_id, COUNT(DISTINCT user_id) AS count").In("poll_id", pollIds).GroupBy("poll_id").Find(&voterCounts)
	if err != nil {
		return nil, err
	}
	mine := []*PollVote{}
	if user != nil {
		err = adapter.Engine.Where("user_id = ?", user.Id).In("poll_id", pollIds).Find(&mine)
		if err != nil {
			return nil, err
		}
	}

	for _, poll := range polls {
		if poll.isClosed() {
			poll.Status = PollClosed
		}
		poll.Options = []*PollOption{}
		poll.UserVotes = []string{}
		for _, option := range options {
			if option.PollId != poll.Id {
				continue
			}
			for _, c := range counts {
				if c.OptionId == option.Id {
					option.Votes = c.Count
				}
			}
			for _, vote := range mine {
				if vote.OptionId == option.Id {
					poll.UserVotes = append(poll.UserVotes, option.Digest)
				}
			}
			poll.Options = append(poll.Options, option)
		}
		for _, v := range voterCounts {
			if v.PollId == poll.Id {
				poll.Voters = v.Count
			}
		}
		closed := poll.Status == PollClosed
		switch poll.Results {
		case "on_vote":
			poll.ResultsHidden = len(poll.UserVotes) == 0 && !closed
		case "on_close":
			poll.ResultsHidden = !closed
		case "staff_only":
			poll.ResultsHidden = !user.IsStaff()
		}
		if poll.ResultsHidden {
			for _, option := range poll.Options {
				option.Votes = 0
			}
		}
		poll.CanClose = user.IsStaff() || (user != nil && authors[poll.PostId] == user.Id)
		res[poll.PostId] = append(res[poll.PostId], poll)
	}
	return res, nil
}

func loadVotablePoll(user *User, postId int64, name string) (*Post, *Topic, *Poll, error) {
	post, topic, err := getActionablePost(user, postId)
	if err != nil {
		return nil, nil, nil, err
	}
	poll := &Poll{}
	has, err := adapter.Engine.Where("post_id = ? AND name = ?", post.Id, name).Get(poll)
	if err != nil {
		return nil, nil, nil, err
	}
	if !has {
		return nil, nil, nil, fmt.Errorf("The poll does not exist")
	}
	return post, topic, poll, nil
}

func getFreshPoll(user *User, post *Post, name string) (*Poll, error) {
	polls, err := GetPostPolls(user, []*Post{post})
	if err != nil {
		return nil, err
	}
	for _, poll := range polls[post.Id] {
		if poll.Name == name {
			return poll, nil
		}
	}
	return nil, fmt.Errorf("The poll does not exist")
}

// VotePoll replaces the user's votes in a poll with the options chosen (their digests).
func VotePoll(user *User, postId int64, name string, digests []string) (*Poll, error) {
	post, topic, poll, err := loadVotablePoll(user, postId, name)
	if err != nil {
		return nil, err
	}
	if user.IsSuspended() || user.IsSilenced() {
		return nil, fmt.Errorf("Your account can't vote")
	}
	if poll.isClosed() || topic.Archived {
		return nil, fmt.Errorf("The poll is closed")
	}
	options := []*PollOption{}
	err = adapter.Engine.Where("poll_id = ?", poll.Id).Find(&options)
	if err != nil {
		return nil, err
	}
	chosen := []*PollOption{}
	for _, digest := range digests {
		for _, option := range options {
			if option.Digest == digest && !containsOption(chosen, option) {
				chosen = append(chosen, option)
			}
		}
	}
	if len(chosen) == 0 {
		return nil, fmt.Errorf("Choose an option")
	}
	if poll.Type == PollTypeRegular && len(chosen) > 1 {
		return nil, fmt.Errorf("Choose one option")
	}
	if poll.Type == PollTypeMultiple {
		min, max := poll.Min, poll.Max
		if min < 1 {
			min = 1
		}
		if max < 1 || max > len(options) {
			max = len(options)
		}
		if len(chosen) < min || len(chosen) > max {
			return nil, fmt.Errorf("Choose between %d and %d options", min, max)
		}
	}

	pollVoteLock.Lock()
	_, err = adapter.Engine.Where("poll_id = ? AND user_id = ?", poll.Id, user.Id).Delete(&PollVote{})
	for _, option := range chosen {
		if err != nil {
			break
		}
		_, err = adapter.Engine.Insert(&PollVote{PollId: poll.Id, OptionId: option.Id, UserId: user.Id})
	}
	pollVoteLock.Unlock()
	if err != nil {
		return nil, err
	}

	PublishToTopic(topic, map[string]interface{}{"type": "poll", "postId": post.Id, "postNumber": post.PostNumber})
	return getFreshPoll(user, post, name)
}

func containsOption(options []*PollOption, option *PollOption) bool {
	for _, o := range options {
		if o.Id == option.Id {
			return true
		}
	}
	return false
}

func RemovePollVote(user *User, postId int64, name string) (*Poll, error) {
	post, topic, poll, err := loadVotablePoll(user, postId, name)
	if err != nil {
		return nil, err
	}
	if poll.isClosed() {
		return nil, fmt.Errorf("The poll is closed")
	}
	_, err = adapter.Engine.Where("poll_id = ? AND user_id = ?", poll.Id, user.Id).Delete(&PollVote{})
	if err != nil {
		return nil, err
	}
	PublishToTopic(topic, map[string]interface{}{"type": "poll", "postId": post.Id, "postNumber": post.PostNumber})
	return getFreshPoll(user, post, name)
}

// SetPollStatus opens or closes a poll, for its author and staff.
func SetPollStatus(user *User, postId int64, name string, status string) (*Poll, error) {
	post, topic, poll, err := loadVotablePoll(user, postId, name)
	if err != nil {
		return nil, err
	}
	if post.UserId != user.Id && !user.IsStaff() {
		return nil, fmt.Errorf("Only the author and staff can close the poll")
	}
	if status != PollOpen && status != PollClosed {
		return nil, fmt.Errorf("Unknown status %s", status)
	}
	poll.Status = status
	cols := []string{"status"}
	if status == PollOpen && poll.isClosed() {
		// reopening a poll whose closing date passed drops the date
		poll.CloseAt = time.Time{}
		cols = append(cols, "close_at")
	}
	_, err = adapter.Engine.ID(poll.Id).Cols(cols...).Update(poll)
	if err != nil {
		return nil, err
	}
	PublishToTopic(topic, map[string]interface{}{"type": "poll", "postId": post.Id, "postNumber": post.PostNumber})
	return getFreshPoll(user, post, name)
}

// GetPollVoters are who voted for an option of a public poll.
func GetPollVoters(user *User, postId int64, name string, digest string) ([]*User, error) {
	post, _, poll, err := loadVotablePoll(user, postId, name)
	if err != nil {
		return nil, err
	}
	fresh, err := getFreshPoll(user, post, name)
	if err != nil {
		return nil, err
	}
	if !poll.IsPublic || fresh.ResultsHidden {
		return nil, fmt.Errorf("The voters of this poll are secret")
	}
	option := &PollOption{}
	has, err := adapter.Engine.Where("poll_id = ? AND digest = ?", poll.Id, digest).Get(option)
	if err != nil || !has {
		return nil, err
	}
	userIds := []int64{}
	err = adapter.Engine.Table("poll_vote").Cols("user_id").Where("option_id = ?", option.Id).Asc("id").Limit(200).Find(&userIds)
	if err != nil {
		return nil, err
	}
	users, err := GetUserMap(userIds)
	if err != nil {
		return nil, err
	}
	res := []*User{}
	for _, id := range userIds {
		if u := users[id]; u != nil {
			res = append(res, u)
		}
	}
	return res, nil
}
