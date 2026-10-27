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
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/core/logs"
)

// Notifications are emailed like Discourse does: a few minutes after they happen, only if still unread then, and
// for replies and mentions only when the user is away from the site (each user can change that).

// How often a user wants the emails of replies and mentions (EmailLevel) and of messages (EmailMessagesLevel).
const (
	EmailWhenAway = "away"
	EmailAlways   = "always"
	EmailNever    = "never"
)

const awayAfter = 10 * time.Minute

func emailLevelOf(level int, messages bool) string {
	// the zero value is the default: when away for replies, always for messages
	switch level {
	case 1:
		if messages {
			return EmailWhenAway
		}
		return EmailAlways
	case 2:
		return EmailNever
	default:
		if messages {
			return EmailAlways
		}
		return EmailWhenAway
	}
}

func EmailLevelName(level int, messages bool) string {
	return emailLevelOf(level, messages)
}

// EmailLevelValue turns a level name into what is stored.
func EmailLevelValue(name string, messages bool) int {
	switch name {
	case EmailNever:
		return 2
	case EmailAlways:
		if messages {
			return 0
		}
		return 1
	default:
		if messages {
			return 1
		}
		return 0
	}
}

var emailedTypes = map[int]bool{
	NotificationMentioned:               true,
	NotificationReplied:                 true,
	NotificationQuoted:                  true,
	NotificationPosted:                  true,
	NotificationGroupMentioned:          true,
	NotificationPrivateMessage:          true,
	NotificationInvitedToPrivateMessage: true,
	NotificationChatMention:             true,
	NotificationChatMessage:             true,
}

func isMessageNotification(notificationType int) bool {
	return notificationType == NotificationPrivateMessage || notificationType == NotificationInvitedToPrivateMessage || notificationType == NotificationChatMessage
}

func IsEmailConfigured() bool {
	return GetSiteSettingBool("email_enabled") && GetSiteSetting("smtp_host") != "" && GetSiteSetting("notification_email") != ""
}

// sendNotificationEmails mails the notifications that waited long enough, and marks the others as skipped.
func sendNotificationEmails() {
	if !IsEmailConfigured() {
		return
	}
	delay := GetSiteSettingInt("email_delay_minutes")
	if delay < 0 {
		delay = 0
	}
	notifications := []*Notification{}
	err := adapter.Engine.Where("email_status = ? AND created_at <= ? AND created_at >= ?", EmailPending,
		dbTime(time.Now().Add(-time.Duration(delay)*time.Minute)), dbTime(time.Now().AddDate(0, 0, -1))).
		Asc("id").Limit(100).Find(&notifications)
	if err != nil {
		logs.Error("sendNotificationEmails(): %v", err)
		return
	}
	for _, notification := range notifications {
		status := EmailSkipped
		sent, err := emailNotification(notification)
		if err != nil {
			logs.Error("emailNotification(%d): %v", notification.Id, err)
		}
		if sent {
			status = EmailSent
		}
		_, err = adapter.Engine.ID(notification.Id).Cols("email_status").Update(&Notification{EmailStatus: status})
		if err != nil {
			logs.Error("sendNotificationEmails(): %v", err)
		}
	}
	// the ones too old to send any more
	_, _ = adapter.Engine.Where("email_status = ? AND created_at < ?", EmailPending, dbTime(time.Now().AddDate(0, 0, -1))).
		Cols("email_status").Update(&Notification{EmailStatus: EmailSkipped})
}

func emailNotification(notification *Notification) (bool, error) {
	fresh := &Notification{}
	has, err := adapter.Engine.ID(notification.Id).Get(fresh)
	if err != nil || !has || fresh.IsRead || !emailedTypes[fresh.NotificationType] {
		return false, err
	}
	user, err := GetUser(fresh.UserId)
	if err != nil || user == nil || user.Email == "" || user.IsSuspended() {
		return false, err
	}
	messages := isMessageNotification(fresh.NotificationType)
	level := user.EmailLevel
	if messages {
		level = user.EmailMessagesLevel
	}
	switch emailLevelOf(level, messages) {
	case EmailNever:
		return false, nil
	case EmailWhenAway:
		if time.Since(user.LastSeenAt) < awayAfter {
			return false, nil
		}
	}
	actor, err := GetUser(fresh.ActingUserId)
	if err != nil {
		return false, err
	}

	siteUrl := strings.TrimSuffix(GetSiteSetting("site_url"), "/")
	title := GetSiteSetting("title")
	var subject, link, body string
	if fresh.NotificationType == NotificationChatMention || fresh.NotificationType == NotificationChatMessage {
		data := map[string]interface{}{}
		_ = json.Unmarshal([]byte(fresh.Data), &data)
		channelId, _ := data["chatChannelId"].(float64)
		messageId, _ := data["chatMessageId"].(float64)
		channel, err := GetChatChannel(int64(channelId))
		if err != nil || channel == nil || !canSeeChannel(user, channel) {
			return false, err
		}
		message, err := getChatMessage(int64(messageId))
		if err != nil || message == nil || message.Deleted {
			return false, err
		}
		name := channel.Name
		if channel.ChannelType == ChatChannelDirect {
			name = actorName(actor)
		}
		link = fmt.Sprintf("%s/chat/c/%s/%d/%d", siteUrl, chatSlug(channel), channel.Id, message.Id)
		if fresh.NotificationType == NotificationChatMention {
			subject = fmt.Sprintf("[%s] %s mentioned you in %s", title, actorName(actor), name)
		} else {
			subject = fmt.Sprintf("[%s] New message from %s", title, actorName(actor))
		}
		body = renderEmailPost(actor, message.Cooked, message.CreatedAt, siteUrl)
	} else {
		topic, err := GetTopic(fresh.TopicId)
		if err != nil || topic == nil {
			return false, err
		}
		canSee, err := CanSeeTopic(user, topic)
		if err != nil || !canSee {
			return false, err
		}
		post, err := GetPostByNumber(topic.Id, fresh.PostNumber)
		if err != nil || post == nil || post.Deleted {
			return false, err
		}
		subject = fmt.Sprintf("[%s] %s", title, topic.Title)
		if topic.IsPrivateMessage() {
			subject = fmt.Sprintf("[%s] [PM] %s", title, topic.Title)
		}
		link = fmt.Sprintf("%s/t/%s/%d/%d", siteUrl, topic.Slug, topic.Id, post.PostNumber)
		body = renderEmailPost(actor, post.Cooked, post.CreatedAt, siteUrl)
	}

	preferences := fmt.Sprintf("%s/u/%s/preferences/emails", siteUrl, user.Username)
	htmlBody := fmt.Sprintf(`<div style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif;font-size:14px;color:#222;max-width:680px">%s
<p style="margin:24px 0"><a href="%s" style="background:#08c;color:#fff;padding:8px 16px;border-radius:3px;text-decoration:none">%s</a></p>
<hr style="border:none;border-top:1px solid #ddd">
<p style="color:#919191;font-size:12px">%s <a href="%s" style="color:#919191">%s</a></p></div>`,
		body, html.EscapeString(link), emailText("visit"), emailText("footer"), html.EscapeString(preferences), emailText("preferences"))
	textBody := fmt.Sprintf("%s\n\n%s: %s\n\n%s %s", htmlToText(body), emailText("visit"), link, emailText("footer"), preferences)
	err = SendEmail(user.Email, subject, htmlBody, textBody, preferences)
	return err == nil, err
}

func chatSlug(channel *ChatChannel) string {
	if channel.Slug != "" {
		return channel.Slug
	}
	return "dm"
}

func actorName(user *User) string {
	if user == nil {
		return "someone"
	}
	return user.Username
}

var emailTexts = map[string]map[string]string{
	"en": {
		"visit":         "View it on the site",
		"footer":        "You are receiving this because of your notification settings.",
		"preferences":   "Change your email settings",
		"testSubject":   "Test email",
		"testBody":      "This is a test email from %s. Email notifications work.",
		"watcherFooter": "You get every new topic and reply because this address is in the site setting new_post_notification_emails.",
		"siteSettings":  "Site settings",
	},
	"zh": {
		"visit":         "到网站上查看",
		"footer":        "你收到这封邮件是因为你的通知设置。",
		"preferences":   "修改邮件设置",
		"testSubject":   "测试邮件",
		"testBody":      "这是来自 %s 的测试邮件，邮件通知可以正常发送。",
		"watcherFooter": "你收到每个新话题和回复，是因为这个地址在站点设置 new_post_notification_emails 里。",
		"siteSettings":  "站点设置",
	},
}

func emailText(key string) string {
	locale := GetSiteSetting("default_locale")
	if texts, ok := emailTexts[locale]; ok {
		return texts[key]
	}
	return emailTexts["en"][key]
}

var relativeLinkRegex = regexp.MustCompile(`(href|src)="/([^/"])`)

func renderEmailPost(author *User, cooked string, at time.Time, siteUrl string) string {
	cooked = relativeLinkRegex.ReplaceAllString(cooked, `$1="`+siteUrl+`/$2`)
	name := actorName(author)
	if author != nil && author.Name != "" && !strings.EqualFold(author.Name, author.Username) {
		name = fmt.Sprintf("%s (%s)", author.Username, author.Name)
	}
	return fmt.Sprintf(`<p style="color:#646464;margin:0 0 8px"><strong style="color:#222">%s</strong> %s</p><div style="line-height:1.5">%s</div>`,
		html.EscapeString(name), at.Format("2006-01-02 15:04"), cooked)
}

var tagRegex = regexp.MustCompile(`<[^>]*>`)

func htmlToText(s string) string {
	s = strings.NewReplacer("<br>", "\n", "</p>", "\n\n", "</li>", "\n", "</div>", "\n").Replace(s)
	s = html.UnescapeString(tagRegex.ReplaceAllString(s, ""))
	lines := []string{}
	for _, line := range strings.Split(s, "\n") {
		lines = append(lines, strings.TrimSpace(line))
	}
	return strings.TrimSpace(regexp.MustCompile(`\n{3,}`).ReplaceAllString(strings.Join(lines, "\n"), "\n\n"))
}

// SendTestEmail checks the mail settings by sending to the address given.
func SendTestEmail(to string) error {
	if GetSiteSetting("smtp_host") == "" || GetSiteSetting("notification_email") == "" {
		return fmt.Errorf("Set smtp_host and notification_email first")
	}
	title := GetSiteSetting("title")
	body := fmt.Sprintf(emailText("testBody"), title)
	return SendEmail(to, fmt.Sprintf("[%s] %s", title, emailText("testSubject")), "<p>"+html.EscapeString(body)+"</p>", body, "")
}

func encodeHeader(s string) string {
	return mime.BEncoding.Encode("UTF-8", s)
}

func randomHex(n int) string {
	buf := make([]byte, n)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}

// SendEmail sends a message with an HTML and a text part through the SMTP server in the site settings.
func SendEmail(to string, subject string, htmlBody string, textBody string, unsubscribeUrl string) error {
	if _, err := mail.ParseAddress(to); err != nil {
		return fmt.Errorf("Invalid email address %s", to)
	}
	fromAddress := GetSiteSetting("notification_email")
	from := mail.Address{Name: GetSiteSetting("title"), Address: fromAddress}
	domain := fromAddress[strings.LastIndex(fromAddress, "@")+1:]
	boundary := "casnode-" + randomHex(12)

	var msg bytes.Buffer
	headers := [][2]string{
		{"From", from.String()},
		{"To", to},
		{"Subject", encodeHeader(subject)},
		{"Date", time.Now().Format(time.RFC1123Z)},
		{"Message-ID", fmt.Sprintf("<%s@%s>", randomHex(16), domain)},
		{"MIME-Version", "1.0"},
		{"Content-Type", fmt.Sprintf("multipart/alternative; boundary=%q", boundary)},
		{"Auto-Submitted", "auto-generated"},
	}
	if unsubscribeUrl != "" {
		headers = append(headers, [2]string{"List-Unsubscribe", "<" + unsubscribeUrl + ">"})
	}
	for _, header := range headers {
		fmt.Fprintf(&msg, "%s: %s\r\n", header[0], header[1])
	}
	msg.WriteString("\r\n")
	for _, part := range [][2]string{{"text/plain", textBody}, {"text/html", htmlBody}} {
		fmt.Fprintf(&msg, "--%s\r\nContent-Type: %s; charset=UTF-8\r\nContent-Transfer-Encoding: base64\r\n\r\n", boundary, part[0])
		encoded := base64.StdEncoding.EncodeToString([]byte(part[1]))
		for len(encoded) > 76 {
			msg.WriteString(encoded[:76] + "\r\n")
			encoded = encoded[76:]
		}
		msg.WriteString(encoded + "\r\n")
	}
	fmt.Fprintf(&msg, "--%s--\r\n", boundary)
	return smtpSend(fromAddress, to, msg.Bytes())
}

func smtpSend(from string, to string, msg []byte) error {
	host := GetSiteSetting("smtp_host")
	port := GetSiteSettingInt("smtp_port")
	if port == 0 {
		port = 587
	}
	address := net.JoinHostPort(host, strconv.Itoa(port))
	security := GetSiteSetting("smtp_security")
	tlsConfig := &tls.Config{ServerName: host}

	var conn net.Conn
	var err error
	dialer := &net.Dialer{Timeout: 15 * time.Second}
	if security == "ssl" || (security == "" && port == 465) {
		conn, err = tls.DialWithDialer(dialer, "tcp", address, tlsConfig)
	} else {
		conn, err = dialer.Dial("tcp", address)
	}
	if err != nil {
		return err
	}
	_ = conn.SetDeadline(time.Now().Add(60 * time.Second))
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		conn.Close()
		return err
	}
	defer client.Close()
	if security != "ssl" && security != "none" {
		if ok, _ := client.Extension("STARTTLS"); ok {
			err = client.StartTLS(tlsConfig)
			if err != nil {
				return err
			}
		} else if security == "starttls" {
			return fmt.Errorf("the SMTP server does not offer STARTTLS")
		}
	}
	if username := GetSiteSetting("smtp_username"); username != "" {
		err = client.Auth(smtp.PlainAuth("", username, GetSiteSetting("smtp_password"), host))
		if err != nil {
			return err
		}
	}
	err = client.Mail(from)
	if err != nil {
		return err
	}
	err = client.Rcpt(to)
	if err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	_, err = writer.Write(msg)
	if err != nil {
		return err
	}
	err = writer.Close()
	if err != nil {
		return err
	}
	return client.Quit()
}
