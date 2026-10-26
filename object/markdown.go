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
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	goldmarkHtml "github.com/yuin/goldmark/renderer/html"
	xhtml "golang.org/x/net/html"
)

// CookedPost is a post's Markdown turned into the HTML shown to readers, with what was found on the way.
type CookedPost struct {
	Html          string
	Excerpt       string
	ImageUrl      string
	Mentions      []string
	GroupMentions []string
	Quotes        []string
	Polls         []*ParsedPoll
	// links on a line of their own whose preview is not fetched yet
	OneboxUrls []string
}

var markdown = goldmark.New(
	goldmark.WithExtensions(extension.GFM, extension.Footnote),
	goldmark.WithParserOptions(parser.WithAutoHeadingID()),
	goldmark.WithRendererOptions(goldmarkHtml.WithHardWraps(), goldmarkHtml.WithUnsafe()),
)

var sanitizer = newSanitizer()

func newSanitizer() *bluemonday.Policy {
	p := bluemonday.UGCPolicy()
	p.AllowAttrs("class").Matching(regexp.MustCompile(`^(quote|title|quote-title|mention|lang-[\w-]+|language-[\w-]+|footnote-ref|footnote-backref|footnotes|onebox|spoiler|poll)(\s[\w-]+)*$`)).Globally()
	p.AllowAttrs("data-username", "data-post", "data-topic").OnElements("aside")
	p.AllowAttrs("data-poll-name").Matching(regexp.MustCompile(`^[\w.\-]{1,100}$`)).OnElements("div")
	p.AllowElements("aside", "details", "summary", "mark", "kbd", "sup", "sub", "ins", "del")
	p.AllowAttrs("open").OnElements("details")
	p.AllowAttrs("width", "height", "loading", "alt", "title").OnElements("img")
	p.AllowAttrs("id").Matching(regexp.MustCompile(`^[\p{L}\p{N}_-]+$`)).OnElements("h1", "h2", "h3", "h4", "h5", "h6", "li", "sup")
	p.AllowAttrs("checked", "disabled", "type").OnElements("input")
	p.AllowAttrs("align").OnElements("th", "td")
	p.RequireNoReferrerOnLinks(false)
	p.AddTargetBlankToFullyQualifiedLinks(false)
	return p
}

var (
	quoteOpenRegex   = regexp.MustCompile(`(?m)^\[quote(?:="([^"\]]*)")?\]\s*$`)
	quoteCloseRegex  = regexp.MustCompile(`(?m)^\[/quote\]\s*$`)
	detailsOpenRegex = regexp.MustCompile(`(?m)^\[details(?:="([^"\]]*)")?\]\s*$`)
	detailsClose     = regexp.MustCompile(`(?m)^\[/details\]\s*$`)
	mentionRegex     = regexp.MustCompile(`(^|[^\w@/.])@([\w][\w.\-]{0,59}[\w]|[\w])`)
	inlineQuoteRegex = regexp.MustCompile(`\[quote(?:="([^"\]]*)")?\]`)
	emojiRegex       = regexp.MustCompile(`:([a-z0-9_+\-]+):`)
)

var emojis = map[string]string{
	"heart": "❤️", "smile": "😄", "slight_smile": "🙂", "grinning": "😀", "laughing": "😆", "joy": "😂", "wink": "😉",
	"blush": "😊", "thinking": "🤔", "confused": "😕", "cry": "😢", "sob": "😭", "sweat_smile": "😅", "rofl": "🤣",
	"+1": "👍", "thumbsup": "👍", "-1": "👎", "thumbsdown": "👎", "clap": "👏", "pray": "🙏", "wave": "👋", "ok_hand": "👌",
	"tada": "🎉", "rocket": "🚀", "fire": "🔥", "100": "💯", "eyes": "👀", "star": "⭐", "sparkles": "✨", "bulb": "💡",
	"white_check_mark": "✅", "heavy_check_mark": "✔️", "x": "❌", "warning": "⚠️", "question": "❓", "exclamation": "❗",
	"bug": "🐛", "wrench": "🔧", "lock": "🔒", "key": "🔑", "memo": "📝", "link": "🔗", "point_right": "👉", "muscle": "💪",
	"heart_eyes": "😍", "sunglasses": "😎", "scream": "😱", "neutral_face": "😐", "upside_down_face": "🙃", "coffee": "☕",
}

func replaceEmoji(text string) string {
	return emojiRegex.ReplaceAllStringFunc(text, func(s string) string {
		if emoji, ok := emojis[s[1:len(s)-1]]; ok {
			return emoji
		}
		return s
	})
}

// expandBlocks turns Discourse's [quote="name, post:2, topic:5"] and [details="Summary"] blocks into HTML whose
// inside is still Markdown (the blank lines make goldmark parse it as Markdown).
func expandBlocks(raw string) string {
	raw = quoteOpenRegex.ReplaceAllStringFunc(raw, func(s string) string {
		m := quoteOpenRegex.FindStringSubmatch(s)
		username, post, topic := parseQuoteParams(m[1])
		title := ""
		if username != "" {
			title = fmt.Sprintf("<div class=\"title\">%s:</div>", html.EscapeString(username))
		}
		return fmt.Sprintf("<aside class=\"quote\" data-username=\"%s\" data-post=\"%s\" data-topic=\"%s\">\n%s\n<blockquote>\n\n",
			html.EscapeString(username), html.EscapeString(post), html.EscapeString(topic), title)
	})
	raw = quoteCloseRegex.ReplaceAllString(raw, "\n\n</blockquote>\n</aside>\n")
	raw = detailsOpenRegex.ReplaceAllStringFunc(raw, func(s string) string {
		m := detailsOpenRegex.FindStringSubmatch(s)
		summary := m[1]
		if summary == "" {
			summary = "Summary"
		}
		return fmt.Sprintf("<details>\n<summary>%s</summary>\n\n", html.EscapeString(summary))
	})
	raw = detailsClose.ReplaceAllString(raw, "\n\n</details>\n")
	return raw
}

func parseQuoteParams(params string) (string, string, string) {
	username, post, topic := "", "", ""
	for i, part := range strings.Split(params, ",") {
		part = strings.TrimSpace(part)
		switch {
		case i == 0:
			username = part
		case strings.HasPrefix(part, "post:"):
			post = strings.TrimPrefix(part, "post:")
		case strings.HasPrefix(part, "topic:"):
			topic = strings.TrimPrefix(part, "topic:")
		}
	}
	return username, post, topic
}

func CookMarkdown(raw string) *CookedPost {
	res := &CookedPost{}
	expanded, polls := expandPolls(raw)
	var buf bytes.Buffer
	err := markdown.Convert([]byte(expandBlocks(expanded)), &buf)
	if err != nil {
		res.Html = "<p>" + html.EscapeString(raw) + "</p>"
		return res
	}
	cooked := sanitizer.Sanitize(buf.String())
	res.Html, res.Mentions, res.GroupMentions, res.Quotes = linkMentions(cooked)
	res.Polls = collectPollOptions(res.Html, polls)
	res.Html, res.OneboxUrls = applyOneboxes(res.Html)
	res.Excerpt, res.ImageUrl = makeExcerpt(res.Html)
	return res
}

type mentionTarget struct {
	href  string
	class string
}

// linkMentions turns @name in the text (not in code or links) into a link to the profile or the group, and collects
// the users and groups mentioned and the users quoted.
func linkMentions(cooked string) (string, []string, []string, []string) {
	doc, err := xhtml.Parse(strings.NewReader("<div id=\"cooked\">" + cooked + "</div>"))
	if err != nil {
		return cooked, nil, nil, nil
	}
	root := findElement(doc, "div")
	if root == nil {
		return cooked, nil, nil, nil
	}

	mentions := []string{}
	groupMentions := []string{}
	quotes := []string{}
	seen := map[string]*mentionTarget{}
	var walk func(n *xhtml.Node, skip bool)
	walk = func(n *xhtml.Node, skip bool) {
		if n.Type == xhtml.ElementNode {
			switch n.Data {
			case "code", "pre", "a":
				skip = true
			case "aside":
				for _, attr := range n.Attr {
					if attr.Key == "data-username" && attr.Val != "" {
						quotes = append(quotes, attr.Val)
					}
				}
			}
		}
		if n.Type == xhtml.TextNode && !skip && strings.Contains(n.Data, ":") {
			n.Data = replaceEmoji(n.Data)
		}
		if n.Type == xhtml.TextNode && !skip && strings.Contains(n.Data, "@") {
			replaceMentions(n, func(name string) *mentionTarget {
				key := strings.ToLower(name)
				if target, ok := seen[key]; ok {
					return target
				}
				var target *mentionTarget
				user, err := GetUserByUsername(name)
				if err == nil && user != nil {
					mentions = append(mentions, user.Username)
					target = &mentionTarget{href: "/u/" + url.PathEscape(user.Username), class: "mention"}
				} else if group, err := GetGroupByName(name); err == nil && group != nil {
					groupMentions = append(groupMentions, group.Name)
					target = &mentionTarget{href: "/g/" + url.PathEscape(group.Name), class: "mention-group"}
				}
				seen[key] = target
				return target
			})
			return
		}
		for c := n.FirstChild; c != nil; {
			next := c.NextSibling
			walk(c, skip)
			c = next
		}
	}
	walk(root, false)

	var buf bytes.Buffer
	for c := root.FirstChild; c != nil; c = c.NextSibling {
		_ = xhtml.Render(&buf, c)
	}
	return buf.String(), mentions, groupMentions, quotes
}

func replaceMentions(n *xhtml.Node, found func(name string) *mentionTarget) {
	text := n.Data
	matches := mentionRegex.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return
	}
	parent := n.Parent
	last := 0
	for _, m := range matches {
		nameStart, nameEnd := m[4], m[5]
		atPos := nameStart - 1
		name := text[nameStart:nameEnd]
		target := found(name)
		if target == nil {
			continue
		}
		if atPos > last {
			parent.InsertBefore(&xhtml.Node{Type: xhtml.TextNode, Data: text[last:atPos]}, n)
		}
		link := &xhtml.Node{Type: xhtml.ElementNode, Data: "a", Attr: []xhtml.Attribute{
			{Key: "class", Val: target.class},
			{Key: "href", Val: target.href},
		}}
		link.AppendChild(&xhtml.Node{Type: xhtml.TextNode, Data: "@" + name})
		parent.InsertBefore(link, n)
		last = nameEnd
	}
	if last < len(text) {
		parent.InsertBefore(&xhtml.Node{Type: xhtml.TextNode, Data: text[last:]}, n)
	}
	parent.RemoveChild(n)
}

func findElement(n *xhtml.Node, tag string) *xhtml.Node {
	if n.Type == xhtml.ElementNode && n.Data == tag {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if res := findElement(c, tag); res != nil {
			return res
		}
	}
	return nil
}

// makeExcerpt is the plain text of the post without its quotes, cut to about 300 characters, and its first image.
func makeExcerpt(cooked string) (string, string) {
	doc, err := xhtml.Parse(strings.NewReader(cooked))
	if err != nil {
		return "", ""
	}
	var text strings.Builder
	imageUrl := ""
	var walk func(n *xhtml.Node)
	walk = func(n *xhtml.Node) {
		if n.Type == xhtml.ElementNode {
			switch n.Data {
			case "aside", "pre", "details":
				return
			case "img":
				if imageUrl == "" {
					for _, attr := range n.Attr {
						if attr.Key == "src" {
							imageUrl = attr.Val
						}
					}
				}
			case "p", "li", "br", "h1", "h2", "h3", "h4", "h5", "h6", "blockquote", "tr":
				text.WriteString(" ")
			}
		}
		if n.Type == xhtml.TextNode {
			text.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	excerpt := strings.Join(strings.Fields(text.String()), " ")
	return truncateRunes(excerpt, 300), imageUrl
}

// StripQuoteTags is the raw text of a post without the quotes it contains, for quoting it in turn.
func StripQuoteTags(raw string) string {
	return strings.TrimSpace(inlineQuoteRegex.ReplaceAllString(quoteCloseRegex.ReplaceAllString(raw, ""), ""))
}
