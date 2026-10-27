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
	"html"
	"regexp"
	"strings"

	"github.com/beego/beego/v2/core/logs"
)

// The site setting new_post_notification_emails lists addresses that get every new topic and reply at once, like
// Discourse's mailing list mode, for whoever runs the forum. Personal messages are left out.

var newPostEmails = make(chan *Post, 500)

var emailListSeparator = regexp.MustCompile(`[|,;\s]+`)

func GetNewPostNotificationEmails() []string {
	res := []string{}
	for _, address := range emailListSeparator.Split(GetSiteSetting("new_post_notification_emails"), -1) {
		if strings.Contains(address, "@") {
			res = append(res, address)
		}
	}
	return res
}

func canSendEmail() bool {
	return GetSiteSetting("smtp_host") != "" && GetSiteSetting("notification_email") != ""
}

// queueNewPostEmail hands a new post to the worker; a full queue drops it rather than slow down posting.
func queueNewPostEmail(topic *Topic, post *Post) {
	if topic.IsPrivateMessage() || len(GetNewPostNotificationEmails()) == 0 || !canSendEmail() {
		return
	}
	select {
	case newPostEmails <- post:
	default:
		logs.Warn("queueNewPostEmail(): the queue is full, post %d is not emailed", post.Id)
	}
}

func runNewPostEmailWorker() {
	for post := range newPostEmails {
		for _, address := range GetNewPostNotificationEmails() {
			err := emailNewPost(post, address)
			if err != nil {
				logs.Error("emailNewPost(%d, %s): %v", post.Id, address, err)
			}
		}
	}
}

func emailNewPost(post *Post, address string) error {
	topic, err := GetTopic(post.TopicId)
	if err != nil || topic == nil || topic.Deleted {
		return err
	}
	author, err := GetUser(post.UserId)
	if err != nil {
		return err
	}
	siteUrl := strings.TrimSuffix(GetSiteSetting("site_url"), "/")
	title := GetSiteSetting("title")
	subject := fmt.Sprintf("[%s] %s", title, topic.Title)
	if category, err := GetCategory(topic.CategoryId); err == nil && category != nil {
		subject = fmt.Sprintf("[%s] [%s] %s", title, category.Name, topic.Title)
	}
	if post.PostNumber > 1 {
		subject = strings.Replace(subject, topic.Title, "Re: "+topic.Title, 1)
	}
	link := fmt.Sprintf("%s/t/%s/%d/%d", siteUrl, topic.Slug, topic.Id, post.PostNumber)
	body := renderEmailPost(author, post.Cooked, post.CreatedAt, siteUrl)
	settings := siteUrl + "/admin/site_settings"
	htmlBody := fmt.Sprintf(`<div style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif;font-size:14px;color:#222;max-width:680px">%s
<p style="margin:24px 0"><a href="%s" style="background:#08c;color:#fff;padding:8px 16px;border-radius:3px;text-decoration:none">%s</a></p>
<hr style="border:none;border-top:1px solid #ddd">
<p style="color:#919191;font-size:12px">%s <a href="%s" style="color:#919191">%s</a></p></div>`,
		body, html.EscapeString(link), emailText("visit"), emailText("watcherFooter"), html.EscapeString(settings), emailText("siteSettings"))
	textBody := fmt.Sprintf("%s\n\n%s: %s\n\n%s %s", htmlToText(body), emailText("visit"), link, emailText("watcherFooter"), settings)
	return SendEmail(address, subject, htmlBody, textBody, "")
}
