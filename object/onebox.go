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
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"html"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/beego/beego/v2/core/logs"
	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/charset"
)

// A link on a line of its own becomes a preview of the page, like Discourse's onebox: the title, description and
// image the page gives for sharing. Pages are fetched once and kept in onebox_cache; the post is cooked again when
// its previews are fetched.

type OneboxCache struct {
	UrlHash     string    `xorm:"varchar(64) pk" json:"-"`
	Url         string    `xorm:"varchar(1000)" json:"url"`
	Ok          bool      `json:"ok"`
	Kind        string    `xorm:"varchar(20)" json:"kind"`
	Title       string    `xorm:"varchar(500)" json:"title"`
	Description string    `xorm:"varchar(1000)" json:"description"`
	Image       string    `xorm:"varchar(1000)" json:"image"`
	SiteName    string    `xorm:"varchar(200)" json:"siteName"`
	Favicon     string    `xorm:"varchar(1000)" json:"favicon"`
	FetchedAt   time.Time `json:"fetchedAt"`
}

const (
	oneboxMaxBytes    = 1 << 20
	oneboxTimeout     = 6 * time.Second
	oneboxRetryFailed = 24 * time.Hour
	oneboxRefresh     = 30 * 24 * time.Hour
)

var internalTopicRegex = regexp.MustCompile(`^/t/[^/?#]+/(\d+)(?:/(\d+))?/?$`)

func urlHash(u string) string {
	sum := sha256.Sum256([]byte(u))
	return hex.EncodeToString(sum[:])
}

func getOneboxCache(u string) *OneboxCache {
	cache := &OneboxCache{}
	has, err := adapter.Engine.ID(urlHash(u)).Get(cache)
	if err != nil || !has {
		return nil
	}
	return cache
}

// oneboxCandidate is the link of a paragraph that holds nothing else, written as a bare URL.
func oneboxCandidate(p *xhtml.Node) string {
	var link *xhtml.Node
	for c := p.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == xhtml.TextNode && strings.TrimSpace(c.Data) == "" {
			continue
		}
		if c.Type != xhtml.ElementNode || c.Data != "a" || link != nil {
			return ""
		}
		link = c
	}
	if link == nil || link.FirstChild == nil || link.FirstChild != link.LastChild || link.FirstChild.Type != xhtml.TextNode {
		return ""
	}
	href := ""
	for _, attr := range link.Attr {
		if attr.Key == "href" {
			href = attr.Val
		}
	}
	text := strings.TrimSpace(link.FirstChild.Data)
	if href == "" || (text != href && "http://"+text != href && "https://"+text != href) {
		return ""
	}
	return href
}

// applyOneboxes puts the previews already fetched in place of the links, and returns the links still to fetch.
func applyOneboxes(cooked string) (string, []string) {
	if !strings.Contains(cooked, "<a ") {
		return cooked, nil
	}
	doc, err := xhtml.Parse(strings.NewReader("<div id=\"cooked\">" + cooked + "</div>"))
	if err != nil {
		return cooked, nil
	}
	root := findElement(doc, "div")
	if root == nil {
		return cooked, nil
	}

	pending := []string{}
	changed := false
	for p := root.FirstChild; p != nil; {
		next := p.NextSibling
		if p.Type == xhtml.ElementNode && p.Data == "p" {
			if href := oneboxCandidate(p); href != "" {
				box, fetch := renderOnebox(href)
				if box != "" {
					nodes, err := xhtml.ParseFragment(strings.NewReader(box), root)
					if err == nil {
						for _, node := range nodes {
							root.InsertBefore(node, p)
						}
						root.RemoveChild(p)
						changed = true
					}
				} else if fetch && len(pending) < 10 {
					pending = append(pending, href)
				}
			}
		}
		p = next
	}
	if !changed {
		return cooked, pending
	}
	var buf bytes.Buffer
	for c := root.FirstChild; c != nil; c = c.NextSibling {
		_ = xhtml.Render(&buf, c)
	}
	return buf.String(), pending
}

// renderOnebox is the preview of a link, or "" with whether it should be fetched first.
func renderOnebox(href string) (string, bool) {
	parsed, err := url.Parse(href)
	if err != nil {
		return "", false
	}
	if box := renderInternalOnebox(parsed); box != "" {
		return box, false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", false
	}
	cache := getOneboxCache(href)
	if cache == nil {
		return "", true
	}
	if !cache.Ok {
		return "", time.Since(cache.FetchedAt) > oneboxRetryFailed
	}
	return oneboxHtml(cache), false
}

// renderInternalOnebox previews a link to a topic of this forum that anyone may read.
func renderInternalOnebox(parsed *url.URL) string {
	if parsed.Host != "" {
		site, err := url.Parse(GetSiteSetting("site_url"))
		if err != nil || site.Host == "" || !strings.EqualFold(site.Host, parsed.Host) {
			return ""
		}
	} else if parsed.Scheme != "" {
		return ""
	}
	m := internalTopicRegex.FindStringSubmatch(parsed.Path)
	if m == nil {
		return ""
	}
	topicId, _ := strconv.ParseInt(m[1], 10, 64)
	topic, err := GetTopic(topicId)
	if err != nil || topic == nil {
		return ""
	}
	canSee, err := CanSeeTopic(nil, topic)
	if err != nil || !canSee {
		return ""
	}
	link := fmt.Sprintf("/t/%s/%d", url.PathEscape(topic.Slug), topic.Id)
	if m[2] != "" {
		link += "/" + m[2]
	}
	categoryHtml := ""
	if category, err := GetCategory(topic.CategoryId); err == nil && category != nil {
		categoryHtml = fmt.Sprintf("<span class=\"badge-category\" style=\"--category-badge-color: #%s\">%s</span>",
			html.EscapeString(category.Color), html.EscapeString(category.Name))
	}
	return fmt.Sprintf("<aside class=\"quote quote-onebox\" data-topic=\"%d\"><div class=\"title\"><div class=\"quote-title__text-content\"><a href=\"%s\">%s</a> %s</div></div><blockquote><p>%s</p></blockquote></aside>",
		topic.Id, html.EscapeString(link), html.EscapeString(topic.Title), categoryHtml, html.EscapeString(truncateRunes(topic.Excerpt, 300)))
}

func oneboxHtml(cache *OneboxCache) string {
	href := html.EscapeString(cache.Url)
	if cache.Kind == "image" {
		return fmt.Sprintf("<p><a href=\"%s\" class=\"onebox\" target=\"_blank\" rel=\"noopener nofollow ugc\"><img src=\"%s\" alt=\"\" loading=\"lazy\"></a></p>", href, href)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "<aside class=\"onebox allowlistedgeneric\" data-onebox-src=\"%s\"><header class=\"source\">", href)
	if cache.Favicon != "" {
		fmt.Fprintf(&b, "<img src=\"%s\" class=\"site-icon\" width=\"16\" height=\"16\" alt=\"\" loading=\"lazy\">", html.EscapeString(cache.Favicon))
	}
	siteName := cache.SiteName
	if siteName == "" {
		if parsed, err := url.Parse(cache.Url); err == nil {
			siteName = parsed.Host
		}
	}
	fmt.Fprintf(&b, "<a href=\"%s\" target=\"_blank\" rel=\"noopener nofollow ugc\">%s</a></header><article class=\"onebox-body\">", href, html.EscapeString(siteName))
	if cache.Image != "" {
		fmt.Fprintf(&b, "<div class=\"aspect-image\"><img src=\"%s\" class=\"thumbnail\" alt=\"\" loading=\"lazy\"></div>", html.EscapeString(cache.Image))
	}
	fmt.Fprintf(&b, "<h3><a href=\"%s\" target=\"_blank\" rel=\"noopener nofollow ugc\">%s</a></h3>", href, html.EscapeString(cache.Title))
	if cache.Description != "" {
		fmt.Fprintf(&b, "<p>%s</p>", html.EscapeString(truncateRunes(cache.Description, 300)))
	}
	b.WriteString("</article><div class=\"onebox-metadata\"></div><div style=\"clear: both\"></div></aside>")
	return b.String()
}

// The previews are fetched only from public addresses, so a post can't make the server read its own network.
var errPrivateAddress = errors.New("the address is not public")

func isPublicIp(ip net.IP) bool {
	return !(ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsUnspecified() || ip.IsMulticast() || ip.IsInterfaceLocalMulticast() ||
		// carrier-grade NAT
		(ip.To4() != nil && ip.To4()[0] == 100 && ip.To4()[1]&0xc0 == 64))
}

var oneboxClient = &http.Client{
	Timeout: oneboxTimeout,
	Transport: &http.Transport{
		Proxy: nil,
		DialContext: (&net.Dialer{
			Timeout: oneboxTimeout,
			Control: func(network, address string, c syscall.RawConn) error {
				host, _, err := net.SplitHostPort(address)
				if err != nil {
					return err
				}
				ip := net.ParseIP(host)
				if ip == nil || !isPublicIp(ip) {
					return errPrivateAddress
				}
				return nil
			},
		}).DialContext,
		TLSHandshakeTimeout:   oneboxTimeout,
		ResponseHeaderTimeout: oneboxTimeout,
		MaxIdleConns:          10,
		IdleConnTimeout:       30 * time.Second,
	},
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return errors.New("too many redirects")
		}
		if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
			return errors.New("not a web address")
		}
		return nil
	},
}

func fetchOnebox(ctx context.Context, href string) *OneboxCache {
	res := &OneboxCache{UrlHash: urlHash(href), Url: truncateRunes(href, 990), FetchedAt: time.Now()}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, href, nil)
	if err != nil {
		return res
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Casnode Onebox; +https://github.com/casnode/casnode)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,image/*;q=0.8,*/*;q=0.5")
	resp, err := oneboxClient.Do(req)
	if err != nil {
		logs.Info("onebox %s: %v", href, err)
		return res
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return res
	}

	mediaType, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if strings.HasPrefix(mediaType, "image/") && mediaType != "image/svg+xml" {
		res.Ok = true
		res.Kind = "image"
		return res
	}
	if mediaType != "text/html" && mediaType != "application/xhtml+xml" {
		return res
	}
	body, err := charset.NewReader(io.LimitReader(resp.Body, oneboxMaxBytes), resp.Header.Get("Content-Type"))
	if err != nil {
		return res
	}
	doc, err := xhtml.Parse(body)
	if err != nil {
		return res
	}

	meta := map[string]string{}
	title := ""
	icon := ""
	var walk func(n *xhtml.Node)
	walk = func(n *xhtml.Node) {
		if n.Type == xhtml.ElementNode {
			switch n.Data {
			case "body":
				return
			case "title":
				if n.FirstChild != nil && title == "" {
					title = n.FirstChild.Data
				}
			case "meta":
				key, content := "", ""
				for _, attr := range n.Attr {
					switch attr.Key {
					case "property", "name":
						key = strings.ToLower(attr.Val)
					case "content":
						content = attr.Val
					}
				}
				if key != "" && content != "" && meta[key] == "" {
					meta[key] = content
				}
			case "link":
				rel, linkHref := "", ""
				for _, attr := range n.Attr {
					switch attr.Key {
					case "rel":
						rel = strings.ToLower(attr.Val)
					case "href":
						linkHref = attr.Val
					}
				}
				if icon == "" && (rel == "icon" || rel == "shortcut icon") && linkHref != "" {
					icon = linkHref
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	pick := func(keys ...string) string {
		for _, key := range keys {
			if value := strings.TrimSpace(meta[key]); value != "" {
				return value
			}
		}
		return ""
	}
	base := resp.Request.URL
	absolute := func(u string) string {
		if u == "" {
			return ""
		}
		parsed, err := base.Parse(u)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return ""
		}
		return parsed.String()
	}
	res.Title = truncateRunes(strings.Join(strings.Fields(html.UnescapeString(firstNonEmpty(pick("og:title", "twitter:title"), title))), " "), 300)
	res.Description = truncateRunes(strings.Join(strings.Fields(pick("og:description", "twitter:description", "description")), " "), 500)
	res.Image = truncateRunes(absolute(pick("og:image", "og:image:url", "twitter:image", "twitter:image:src")), 990)
	res.SiteName = truncateRunes(pick("og:site_name", "application-name"), 190)
	res.Favicon = truncateRunes(absolute(icon), 990)
	res.Kind = "page"
	res.Ok = res.Title != ""
	return res
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func saveOneboxCache(cache *OneboxCache) {
	has, err := adapter.Engine.ID(cache.UrlHash).Exist(&OneboxCache{})
	if err != nil {
		return
	}
	if has {
		_, err = adapter.Engine.ID(cache.UrlHash).AllCols().Update(cache)
	} else {
		_, err = adapter.Engine.Insert(cache)
	}
	if err != nil {
		logs.Error("saveOneboxCache(): %v", err)
	}
}

var oneboxFetching sync.Map

// FetchOneboxes fetches the previews of the links, a few at a time, waiting at most timeout.
func FetchOneboxes(urls []string, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	var wg sync.WaitGroup
	limit := make(chan struct{}, 4)
	for _, u := range urls {
		if _, busy := oneboxFetching.LoadOrStore(u, true); busy {
			continue
		}
		wg.Add(1)
		go func(u string) {
			defer wg.Done()
			defer oneboxFetching.Delete(u)
			limit <- struct{}{}
			defer func() { <-limit }()
			saveOneboxCache(fetchOnebox(ctx, u))
		}(u)
	}
	wg.Wait()
}

// RebakePostAfterOneboxes fetches the previews of a new or edited post in the background, then cooks it again and
// tells the readers of the topic.
func RebakePostAfterOneboxes(postId int64, urls []string) {
	if len(urls) == 0 {
		return
	}
	go func() {
		FetchOneboxes(urls, 20*time.Second)
		post, err := GetPost(postId)
		if err != nil || post == nil {
			return
		}
		cooked := CookMarkdown(post.Raw)
		if cooked.Html == post.Cooked {
			return
		}
		_, err = adapter.Engine.ID(post.Id).Cols("cooked").Update(&Post{Cooked: cooked.Html})
		if err != nil {
			logs.Error("RebakePostAfterOneboxes(): %v", err)
			return
		}
		topic, err := GetTopic(post.TopicId)
		if err == nil && topic != nil {
			if post.PostNumber == 1 && cooked.ImageUrl != topic.ImageUrl {
				_, _ = adapter.Engine.ID(topic.Id).Cols("image_url").Update(&Topic{ImageUrl: cooked.ImageUrl})
			}
			PublishToTopic(topic, map[string]interface{}{"type": "rebaked", "postId": post.Id, "postNumber": post.PostNumber})
		}
	}()
}

// refreshOldOneboxes drops the previews fetched long ago, so they are fetched again when a post is cooked.
func refreshOldOneboxes() {
	_, _ = adapter.Engine.Where("fetched_at < ?", dbTime(time.Now().Add(-oneboxRefresh))).Delete(&OneboxCache{})
}
