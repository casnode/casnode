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

package sessions

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web"
	webcontext "github.com/beego/beego/v2/server/web/context"
	"github.com/beego/beego/v2/server/web/session"
)

const FileProviderName = "casnode_file"

// fileProvider keeps the existing Beego file format. It serializes each
// read-modify-write so that requests only persist the keys they changed.
type fileProvider struct {
	session.Provider
	mu sync.Mutex
}

type change struct {
	value   any
	deleted bool
}

type fileStore struct {
	provider *fileProvider
	sid      string

	mu            sync.Mutex
	base          session.Store
	changes       map[any]change
	flushed       bool
	pendingCookie string
	response      *webcontext.Response
	header        http.Header
}

func init() {
	provider, err := session.GetProvider("file")
	if err != nil {
		panic(err)
	}
	session.Register(FileProviderName, &fileProvider{Provider: provider})
}

func (p *fileProvider) SessionRead(ctx context.Context, sid string) (session.Store, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	exists, err := p.Provider.SessionExist(ctx, sid)
	if err != nil {
		return nil, err
	}
	store := &fileStore{provider: p, sid: sid}
	if exists {
		store.base, err = p.Provider.SessionRead(ctx, sid)
	}
	return store, err
}

func (p *fileProvider) SessionRegenerate(ctx context.Context, oldSID, newSID string) (session.Store, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	exists, err := p.Provider.SessionExist(ctx, oldSID)
	if err != nil {
		return nil, err
	}
	store := &fileStore{provider: p, sid: newSID}
	if exists {
		store.base, err = p.Provider.SessionRegenerate(ctx, oldSID, newSID)
	}
	return store, err
}

func (p *fileProvider) SessionDestroy(ctx context.Context, sid string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.Provider.SessionDestroy(ctx, sid)
}

func (p *fileProvider) SessionGC(ctx context.Context) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Provider.SessionGC(ctx)
}

func (s *fileStore) Set(ctx context.Context, key, value any) error {
	s.record(ctx, key, change{value: value})
	return nil
}

func (s *fileStore) Delete(ctx context.Context, key any) error {
	s.record(ctx, key, change{deleted: true})
	return nil
}

func (s *fileStore) record(ctx context.Context, key any, update change) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.changes == nil {
		s.changes = make(map[any]change)
	}
	s.changes[key] = update
	s.saveBeforeResponse(ctx)
}

func (s *fileStore) Get(ctx context.Context, key any) any {
	s.mu.Lock()
	defer s.mu.Unlock()
	if update, ok := s.changes[key]; ok {
		return update.value
	}
	if s.flushed || s.base == nil {
		return nil
	}
	return s.base.Get(ctx, key)
}

func (s *fileStore) Flush(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.changes = nil
	s.flushed = true
	s.saveBeforeResponse(ctx)
	return nil
}

func (s *fileStore) SessionID(context.Context) string {
	return s.sid
}

func (s *fileStore) SessionRelease(ctx context.Context, _ http.ResponseWriter) {
	if err := s.commit(ctx, false); err != nil {
		logs.Error("could not save forum session: %v", err)
	}
}

func (s *fileStore) SessionReleaseIfPresent(ctx context.Context, _ http.ResponseWriter) {
	if err := s.commit(ctx, true); err != nil {
		logs.Error("could not save forum session: %v", err)
	}
}

func (s *fileStore) commit(ctx context.Context, existingOnly bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.flushed && len(s.changes) == 0 {
		return nil
	}
	updates, flushed := s.changes, s.flushed
	s.changes, s.flushed = nil, false

	s.provider.mu.Lock()
	defer s.provider.mu.Unlock()
	provider := s.provider.Provider
	exists, err := provider.SessionExist(ctx, s.sid)
	if err != nil {
		return err
	}
	if !exists {
		if existingOnly || !hasNewValue(updates) {
			s.base = nil
			return nil
		}
		if s.base != nil {
			return errors.New("forum session no longer exists")
		}
	}

	latest, err := provider.SessionRead(ctx, s.sid)
	if err != nil {
		return err
	}
	if flushed {
		if err := latest.Flush(ctx); err != nil {
			return err
		}
	}
	for key, update := range updates {
		if update.deleted {
			err = latest.Delete(ctx, key)
		} else {
			err = latest.Set(ctx, key, update.value)
		}
		if err != nil {
			return err
		}
	}
	latest.SessionReleaseIfPresent(ctx, nil)
	s.base = latest
	if s.pendingCookie != "" {
		s.header.Add("Set-Cookie", s.pendingCookie)
		s.pendingCookie = ""
	}
	return nil
}

func hasNewValue(updates map[any]change) bool {
	for _, update := range updates {
		if !update.deleted {
			return true
		}
	}
	return false
}

// PrepareResponse gives session changes a chance to save before headers are
// sent. Keep new cookies pending so an anonymous response cannot overwrite
// a cookie issued by sign-in.
func PrepareResponse(ctx *webcontext.Context) {
	store, ok := ctx.Input.CruSession.(*fileStore)
	if !ok {
		return
	}
	store.response = ctx.ResponseWriter
	if store.base != nil {
		return
	}
	header := ctx.ResponseWriter.Header()
	cookies := header.Values("Set-Cookie")
	want := web.BConfig.WebConfig.Session.SessionName + "=" + url.QueryEscape(store.sid)
	for i, cookie := range cookies {
		pair, _, _ := strings.Cut(cookie, ";")
		if pair == want {
			store.pendingCookie = cookie
			store.header = header
			header["Set-Cookie"] = append(cookies[:i:i], cookies[i+1:]...)
			return
		}
	}
}

// Read-only requests use Beego's original writer.
func (s *fileStore) saveBeforeResponse(ctx context.Context) {
	response := s.response
	if response == nil || response.Started {
		return
	}
	response.ResponseWriter = &sessionResponse{ResponseWriter: response.ResponseWriter, store: s, ctx: ctx}
	s.response = nil
}

type sessionResponse struct {
	http.ResponseWriter
	store *fileStore
	ctx   context.Context
	saved bool
	err   error
}

func (w *sessionResponse) WriteHeader(code int) {
	if w.save() == nil {
		w.ResponseWriter.WriteHeader(code)
	}
}

func (w *sessionResponse) Write(data []byte) (int, error) {
	if err := w.save(); err != nil {
		return 0, err
	}
	return w.ResponseWriter.Write(data)
}

func (w *sessionResponse) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *sessionResponse) save() error {
	if !w.saved {
		w.saved = true
		w.err = w.store.commit(w.ctx, false)
		if w.err != nil {
			logs.Error("could not save forum session: %v", w.err)
			w.Header().Del("Location")
			w.Header().Del("Content-Encoding")
			http.Error(w.ResponseWriter, "Could not save the forum session", http.StatusServiceUnavailable)
		}
	}
	return w.err
}
