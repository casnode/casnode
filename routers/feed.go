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

package routers

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/server/web/context"
	"github.com/casnode/casnode/object"
)

const sitemapPageSize = 10000

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	Creator     string `xml:"dc:creator"`
	PubDate     string `xml:"pubDate"`
	Guid        string `xml:"guid"`
	Category    string `xml:"category,omitempty"`
}

type rssChannel struct {
	Title       string     `xml:"title"`
	Link        string     `xml:"link"`
	Description string     `xml:"description"`
	Language    string     `xml:"language,omitempty"`
	Items       []*rssItem `xml:"item"`
}

type rssDoc struct {
	XMLName xml.Name    `xml:"rss"`
	Version string      `xml:"version,attr"`
	DC      string      `xml:"xmlns:dc,attr"`
	Channel *rssChannel `xml:"channel"`
}

func getSiteUrl(ctx *context.Context) string {
	siteUrl := strings.TrimSuffix(object.GetSiteSetting("site_url"), "/")
	if siteUrl != "" {
		return siteUrl
	}
	scheme := "http"
	if ctx.Request.TLS != nil || ctx.Request.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return scheme + "://" + ctx.Request.Host
}

func topicUrl(siteUrl string, topic *object.Topic) string {
	return fmt.Sprintf("%s/t/%s/%d", siteUrl, url.PathEscape(topic.Slug), topic.Id)
}

var (
	categoryFeedRegex = regexp.MustCompile(`^/c/(?:[^/]+/)*(\d+)\.rss$`)
	topicFeedRegex    = regexp.MustCompile(`^/t/[^/]+/(\d+)\.rss$`)
	tagFeedRegex      = regexp.MustCompile(`^/tag/([^/]+)\.rss$`)
	sitemapPageRegex  = regexp.MustCompile(`^/sitemap_(\d+)\.xml$`)
)

// serveFeed answers the RSS feeds, sitemaps and robots.txt, and tells whether the path was one of them.
func serveFeed(ctx *context.Context) bool {
	path := ctx.Request.URL.Path
	switch {
	case path == "/robots.txt":
		serveRobots(ctx)
	case path == "/sitemap.xml":
		serveSitemapIndex(ctx)
	case sitemapPageRegex.MatchString(path):
		page, _ := strconv.Atoi(sitemapPageRegex.FindStringSubmatch(path)[1])
		serveSitemapPage(ctx, page)
	case path == "/latest.rss":
		serveTopicListFeed(ctx, &object.TopicListQuery{Filter: "latest", Order: "created"}, "")
	case path == "/top.rss":
		serveTopicListFeed(ctx, &object.TopicListQuery{Filter: "top", Period: "monthly"}, "Top")
	case categoryFeedRegex.MatchString(path):
		id, _ := strconv.ParseInt(categoryFeedRegex.FindStringSubmatch(path)[1], 10, 64)
		category, err := object.GetCategory(id)
		if err != nil || category == nil || !category.CanSee(nil) {
			http.NotFound(ctx.ResponseWriter, ctx.Request)
			return true
		}
		serveTopicListFeed(ctx, &object.TopicListQuery{Filter: "latest", Order: "created", CategoryId: id}, category.Name)
	case tagFeedRegex.MatchString(path):
		tag, _ := url.PathUnescape(tagFeedRegex.FindStringSubmatch(path)[1])
		serveTopicListFeed(ctx, &object.TopicListQuery{Filter: "latest", Order: "created", Tag: tag}, "#"+tag)
	case topicFeedRegex.MatchString(path):
		id, _ := strconv.ParseInt(topicFeedRegex.FindStringSubmatch(path)[1], 10, 64)
		serveTopicFeed(ctx, id)
	default:
		return false
	}
	return true
}

func writeXml(ctx *context.Context, contentType string, v interface{}) {
	data, err := xml.MarshalIndent(v, "", "  ")
	if err != nil {
		http.Error(ctx.ResponseWriter, err.Error(), http.StatusInternalServerError)
		return
	}
	ctx.ResponseWriter.Header().Set("Content-Type", contentType)
	ctx.ResponseWriter.Header().Set("Cache-Control", "public, max-age=300")
	_, _ = ctx.ResponseWriter.Write([]byte(xml.Header))
	_, _ = ctx.ResponseWriter.Write(data)
}

func serveTopicListFeed(ctx *context.Context, query *object.TopicListQuery, title string) {
	if object.GetSiteSettingBool("login_required") {
		http.NotFound(ctx.ResponseWriter, ctx.Request)
		return
	}
	query.PageSize = 25
	list, err := object.GetTopicList(nil, query)
	if err != nil {
		http.Error(ctx.ResponseWriter, err.Error(), http.StatusInternalServerError)
		return
	}
	siteUrl := getSiteUrl(ctx)
	siteTitle := object.GetSiteSetting("title")
	channel := &rssChannel{
		Title:       siteTitle,
		Link:        siteUrl,
		Description: object.GetSiteSetting("site_description"),
		Language:    object.GetSiteSetting("default_locale"),
	}
	if title != "" {
		channel.Title = fmt.Sprintf("%s - %s", title, siteTitle)
	}
	for _, topic := range list.Topics {
		if topic.IsPinned() && query.Filter == "latest" {
			continue
		}
		post, err := object.GetPostByNumber(topic.Id, 1)
		if err != nil || post == nil {
			continue
		}
		creator := ""
		if user := list.Users[topic.UserId]; user != nil {
			creator = "@" + user.Username
		}
		category, _ := object.GetCategory(topic.CategoryId)
		item := &rssItem{
			Title:       topic.Title,
			Link:        topicUrl(siteUrl, topic),
			Description: post.Cooked,
			Creator:     creator,
			PubDate:     topic.CreatedAt.Format(time.RFC1123Z),
			Guid:        fmt.Sprintf("%s-topic-%d", ctx.Request.Host, topic.Id),
		}
		if category != nil {
			item.Category = category.Name
		}
		channel.Items = append(channel.Items, item)
	}
	writeXml(ctx, "application/rss+xml; charset=utf-8", &rssDoc{Version: "2.0", DC: "http://purl.org/dc/elements/1.1/", Channel: channel})
}

func serveTopicFeed(ctx *context.Context, id int64) {
	topic, err := object.GetTopic(id)
	canSee, _ := object.CanSeeTopic(nil, topic)
	if err != nil || !canSee || object.GetSiteSettingBool("login_required") {
		http.NotFound(ctx.ResponseWriter, ctx.Request)
		return
	}
	stream, err := object.GetPostStream(nil, topic.Id)
	if err != nil {
		http.Error(ctx.ResponseWriter, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(stream) > 25 {
		stream = stream[len(stream)-25:]
	}
	posts, err := object.GetPostsByNumbers(nil, topic, stream)
	if err != nil {
		http.Error(ctx.ResponseWriter, err.Error(), http.StatusInternalServerError)
		return
	}
	userIds := []int64{}
	for _, post := range posts {
		userIds = append(userIds, post.UserId)
	}
	users, _ := object.GetUserMap(userIds)

	siteUrl := getSiteUrl(ctx)
	channel := &rssChannel{
		Title:       topic.Title,
		Link:        topicUrl(siteUrl, topic),
		Description: topic.Excerpt,
	}
	for i := len(posts) - 1; i >= 0; i-- {
		post := posts[i]
		if post.PostType != object.PostTypeRegular || post.Hidden {
			continue
		}
		creator := ""
		if user := users[post.UserId]; user != nil {
			creator = "@" + user.Username
		}
		channel.Items = append(channel.Items, &rssItem{
			Title:       topic.Title,
			Link:        fmt.Sprintf("%s/%d", topicUrl(siteUrl, topic), post.PostNumber),
			Description: post.Cooked,
			Creator:     creator,
			PubDate:     post.CreatedAt.Format(time.RFC1123Z),
			Guid:        fmt.Sprintf("%s-post-%d", ctx.Request.Host, post.Id),
		})
	}
	writeXml(ctx, "application/rss+xml; charset=utf-8", &rssDoc{Version: "2.0", DC: "http://purl.org/dc/elements/1.1/", Channel: channel})
}

func serveRobots(ctx *context.Context) {
	siteUrl := getSiteUrl(ctx)
	lines := []string{"User-agent: *"}
	if object.GetSiteSettingBool("login_required") {
		lines = append(lines, "Disallow: /")
	} else {
		for _, path := range []string{"/admin/", "/review", "/search", "/callback", "/new", "/unread", "/my/", "/u/*/preferences", "/u/*/notifications", "/u/*/messages", "/u/*/activity/bookmarks", "/api/"} {
			lines = append(lines, "Disallow: "+path)
		}
		lines = append(lines, "", "Sitemap: "+siteUrl+"/sitemap.xml")
	}
	ctx.ResponseWriter.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = ctx.ResponseWriter.Write([]byte(strings.Join(lines, "\n") + "\n"))
}

type sitemapIndex struct {
	XMLName  xml.Name      `xml:"sitemapindex"`
	Xmlns    string        `xml:"xmlns,attr"`
	Sitemaps []*sitemapLoc `xml:"sitemap"`
}

type sitemapLoc struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
}

type urlSet struct {
	XMLName xml.Name      `xml:"urlset"`
	Xmlns   string        `xml:"xmlns,attr"`
	Urls    []*sitemapLoc `xml:"url"`
}

func serveSitemapIndex(ctx *context.Context) {
	if object.GetSiteSettingBool("login_required") {
		http.NotFound(ctx.ResponseWriter, ctx.Request)
		return
	}
	count, err := object.CountPublicTopics()
	if err != nil {
		http.Error(ctx.ResponseWriter, err.Error(), http.StatusInternalServerError)
		return
	}
	siteUrl := getSiteUrl(ctx)
	index := &sitemapIndex{Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9"}
	pages := int(count+sitemapPageSize-1) / sitemapPageSize
	if pages == 0 {
		pages = 1
	}
	for i := 1; i <= pages; i++ {
		index.Sitemaps = append(index.Sitemaps, &sitemapLoc{Loc: fmt.Sprintf("%s/sitemap_%d.xml", siteUrl, i)})
	}
	writeXml(ctx, "application/xml; charset=utf-8", index)
}

func serveSitemapPage(ctx *context.Context, page int) {
	if object.GetSiteSettingBool("login_required") || page < 1 {
		http.NotFound(ctx.ResponseWriter, ctx.Request)
		return
	}
	topics, err := object.GetTopicsForSitemap(page-1, sitemapPageSize)
	if err != nil {
		http.Error(ctx.ResponseWriter, err.Error(), http.StatusInternalServerError)
		return
	}
	siteUrl := getSiteUrl(ctx)
	set := &urlSet{Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9"}
	if page == 1 {
		set.Urls = append(set.Urls, &sitemapLoc{Loc: siteUrl + "/"}, &sitemapLoc{Loc: siteUrl + "/categories"})
		categories, _ := object.GetCategories(nil)
		for _, category := range categories {
			set.Urls = append(set.Urls, &sitemapLoc{Loc: fmt.Sprintf("%s/c/%s/%d", siteUrl, url.PathEscape(category.Slug), category.Id)})
		}
	}
	for _, topic := range topics {
		set.Urls = append(set.Urls, &sitemapLoc{Loc: topicUrl(siteUrl, topic), LastMod: topic.BumpedAt.Format(time.RFC3339)})
		// the crawler pages of long topics
		for p := 2; p <= (topic.PostsCount+object.PostChunkSize-1)/object.PostChunkSize; p++ {
			set.Urls = append(set.Urls, &sitemapLoc{Loc: fmt.Sprintf("%s?page=%d", topicUrl(siteUrl, topic), p)})
		}
	}
	writeXml(ctx, "application/xml; charset=utf-8", set)
}
