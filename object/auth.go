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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html"
	"net/mail"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// How a user signs in. Users from Casdoor (and devSignin) have none, which is also what tells that the user names in
// the `admins` config may make them administrators: anyone could sign up locally under such a name.
const (
	LoginTypeLocal = "local"
	LoginTypeOAuth = "oauth"
)

// UserToken is a one-time link sent by email, such as for resetting the password; only the hash is kept.
type UserToken struct {
	Id        int64     `xorm:"pk autoincr" json:"id"`
	UserId    int64     `xorm:"index" json:"userId"`
	Purpose   string    `xorm:"varchar(20)" json:"purpose"`
	TokenHash string    `xorm:"varchar(64) unique" json:"-"`
	ExpiresAt time.Time `json:"expiresAt"`
	CreatedAt time.Time `xorm:"created" json:"createdAt"`
}

const tokenPasswordReset = "password_reset"

var (
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.\-]{2,19}$`)
	// like Discourse's reserved_usernames, plus the names of pages under /u/
	reservedUsernames = []string{
		"admin", "administrator", "moderator", "mod", "staff", "sys", "system", "root", "support", "help", "info",
		"community", "casnode", "discourse", "user", "username", "users", "nickname", "name", "you", "me", "anonymous",
		"guest", "null", "undefined", "webmaster", "postmaster", "noreply", "no-reply", "password-reset", "signup", "login",
	}
	dummyPasswordHash, _ = bcrypt.GenerateFromPassword([]byte("casnode dummy password"), bcrypt.DefaultCost)
)

func validateUsername(username string) error {
	if !usernameRegex.MatchString(username) {
		return fmt.Errorf("A user name is 3 to 20 letters, digits, dots, dashes or underscores, starting with a letter or a digit")
	}
	if slices.Contains(reservedUsernames, strings.ToLower(username)) {
		return fmt.Errorf("The user name %s is reserved", username)
	}
	return nil
}

func validatePassword(password string, username string, email string) error {
	minLength := GetSiteSettingInt("min_password_length")
	if minLength < 6 {
		minLength = 6
	}
	if len([]rune(password)) < minLength {
		return fmt.Errorf("The password must be at least %d characters", minLength)
	}
	// bcrypt only looks at the first 72 bytes
	if len(password) > 72 {
		return fmt.Errorf("The password must be at most 72 bytes")
	}
	lower := strings.ToLower(password)
	if lower == strings.ToLower(username) || (email != "" && lower == strings.ToLower(email)) {
		return fmt.Errorf("The password must not be your user name or email")
	}
	return nil
}

func normalizeEmail(email string) (string, error) {
	email = strings.TrimSpace(email)
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email || !strings.Contains(email[strings.LastIndex(email, "@")+1:], ".") {
		return "", fmt.Errorf("Invalid email address")
	}
	return email, nil
}

// getPasswordUserByEmail finds the local account with this email; accounts from single sign-on have no password and
// may share an email with others.
func getPasswordUserByEmail(email string) (*User, error) {
	user := &User{}
	has, err := adapter.Engine.Where("LOWER(email) = ? AND password_hash <> ?", strings.ToLower(email), "").Get(user)
	if err != nil || !has {
		return nil, err
	}
	return user, nil
}

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

// isFirstUser tells whether the forum has nobody yet: the first account becomes its administrator.
func isFirstUser() (bool, error) {
	count, err := adapter.Engine.Where("username <> ?", SystemUsername).Count(&User{})
	return count == 0, err
}

func insertNewUser(user *User) error {
	first, err := isFirstUser()
	if err != nil {
		return err
	}
	if first {
		user.Admin = true
	}
	user.TrustLevel = 1
	user.LastSeenAt = time.Now()
	_, err = adapter.Engine.Insert(user)
	if err == nil && user.Admin {
		err = RefreshAutomaticGroups()
	}
	return err
}

// SignUp creates a local account with a password.
func SignUp(username string, email string, name string, password string) (*User, error) {
	if !GetSiteSettingBool("enable_local_logins") || !GetSiteSettingBool("allow_new_registrations") {
		return nil, fmt.Errorf("New registrations are not allowed")
	}
	username = strings.TrimSpace(username)
	name = strings.TrimSpace(name)
	err := validateUsername(username)
	if err != nil {
		return nil, err
	}
	email, err = normalizeEmail(email)
	if err != nil {
		return nil, err
	}
	err = validatePassword(password, username, email)
	if err != nil {
		return nil, err
	}
	if len([]rune(name)) > 100 {
		return nil, fmt.Errorf("The name is too long")
	}

	existing, err := GetUserByUsername(username)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("The user name %s is taken", username)
	}
	existing, err = getPasswordUserByEmail(email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("An account with this email already exists")
	}

	hash, err := hashPassword(password)
	if err != nil {
		return nil, err
	}
	if name == "" {
		name = username
	}
	user := &User{
		Username:     username,
		Name:         name,
		Email:        email,
		LoginType:    LoginTypeLocal,
		PasswordHash: hash,
	}
	err = insertNewUser(user)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// LogIn checks a user name or email and a password.
func LogIn(login string, password string) (*User, error) {
	if !GetSiteSettingBool("enable_local_logins") {
		return nil, fmt.Errorf("Logging in with a password is turned off")
	}
	login = strings.TrimSpace(login)
	var user *User
	var err error
	if strings.Contains(login, "@") {
		user, err = getPasswordUserByEmail(login)
	} else {
		user, err = GetUserByUsername(login)
	}
	if err != nil {
		return nil, err
	}

	hash := dummyPasswordHash
	if user != nil && user.PasswordHash != "" {
		hash = []byte(user.PasswordHash)
	}
	// compare even when there is no such user, so that the time taken does not tell
	err = bcrypt.CompareHashAndPassword(hash, []byte(password))
	if err != nil || user == nil || user.PasswordHash == "" {
		return nil, fmt.Errorf("Incorrect user name, email or password")
	}
	return user, nil
}

// ChangePassword sets a new password for a local account, after checking the current one.
func ChangePassword(user *User, currentPassword string, newPassword string) error {
	if user.LoginType != LoginTypeLocal || user.PasswordHash == "" {
		return fmt.Errorf("This account signs in elsewhere and has no password here")
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)) != nil {
		return fmt.Errorf("The current password is incorrect")
	}
	return setPassword(user, newPassword)
}

func setPassword(user *User, password string) error {
	err := validatePassword(password, user.Username, user.Email)
	if err != nil {
		return err
	}
	hash, err := hashPassword(password)
	if err != nil {
		return err
	}
	user.PasswordHash = hash
	_, err = adapter.Engine.ID(user.Id).Cols("password_hash").Update(user)
	if err != nil {
		return err
	}
	_, err = adapter.Engine.Where("user_id = ? AND purpose = ?", user.Id, tokenPasswordReset).Delete(&UserToken{})
	return err
}

// NewRandomToken is 32 hex characters for the state of a login and such.
func NewRandomToken() string {
	return randomHex(16)
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func CanResetPassword() bool {
	return GetSiteSettingBool("enable_local_logins") && GetSiteSetting("smtp_host") != "" && GetSiteSetting("notification_email") != ""
}

// SendPasswordResetEmail mails a link for setting a new password to the local account with this user name or email.
// It says nothing about whether there is such an account.
func SendPasswordResetEmail(login string, siteUrl string) error {
	if !CanResetPassword() {
		return fmt.Errorf("Resetting the password by email is not set up on this site")
	}
	login = strings.TrimSpace(login)
	var user *User
	var err error
	if strings.Contains(login, "@") {
		user, err = getPasswordUserByEmail(login)
	} else {
		user, err = GetUserByUsername(login)
	}
	if err != nil || user == nil || user.PasswordHash == "" || user.Email == "" {
		return err
	}

	token := randomHex(32)
	_, err = adapter.Engine.Insert(&UserToken{
		UserId:    user.Id,
		Purpose:   tokenPasswordReset,
		TokenHash: hashToken(token),
		ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		return err
	}

	title := GetSiteSetting("title")
	link := fmt.Sprintf("%s/password-reset/%s", siteUrl, token)
	text := fmt.Sprintf(emailText("resetBody"), user.Username, title)
	htmlBody := fmt.Sprintf(`<p>%s</p><p><a href="%s">%s</a></p><p>%s</p>`, html.EscapeString(text), link, html.EscapeString(link), html.EscapeString(emailText("resetIgnore")))
	textBody := fmt.Sprintf("%s\n\n%s\n\n%s\n", text, link, emailText("resetIgnore"))
	return SendEmail(user.Email, fmt.Sprintf("[%s] %s", title, emailText("resetSubject")), htmlBody, textBody, "")
}

// ResetPassword sets the password through a link from SendPasswordResetEmail and returns whose it was.
func ResetPassword(token string, password string) (*User, error) {
	record := &UserToken{}
	has, err := adapter.Engine.Where("token_hash = ? AND purpose = ?", hashToken(token), tokenPasswordReset).Get(record)
	if err != nil {
		return nil, err
	}
	if !has || record.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("This link has expired or was already used; ask for a new one")
	}
	user, err := GetUser(record.UserId)
	if err != nil {
		return nil, err
	}
	if user == nil || user.PasswordHash == "" {
		return nil, fmt.Errorf("This link has expired or was already used; ask for a new one")
	}
	err = setPassword(user, password)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// RateLimiter counts recent attempts per key, such as failed logins per address, in memory.
type RateLimiter struct {
	max      int
	window   time.Duration
	lock     sync.Mutex
	attempts map[string][]time.Time
}

func NewRateLimiter(max int, window time.Duration) *RateLimiter {
	return &RateLimiter{max: max, window: window, attempts: map[string][]time.Time{}}
}

func (l *RateLimiter) recent(key string, now time.Time) []time.Time {
	kept := l.attempts[key][:0]
	for _, at := range l.attempts[key] {
		if now.Sub(at) < l.window {
			kept = append(kept, at)
		}
	}
	if len(kept) == 0 {
		delete(l.attempts, key)
		return nil
	}
	l.attempts[key] = kept
	return kept
}

// Allowed tells whether key has attempts left.
func (l *RateLimiter) Allowed(key string) bool {
	l.lock.Lock()
	defer l.lock.Unlock()
	return len(l.recent(key, time.Now())) < l.max
}

func (l *RateLimiter) Add(key string) {
	l.lock.Lock()
	defer l.lock.Unlock()
	now := time.Now()
	l.attempts[key] = append(l.recent(key, now), now)
}
