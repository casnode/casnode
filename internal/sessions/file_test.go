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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/beego/beego/v2/server/web"
	webcontext "github.com/beego/beego/v2/server/web/context"
	"github.com/beego/beego/v2/server/web/session"
)

func testManager(t *testing.T) *session.Manager {
	t.Helper()
	manager, err := session.NewManager(FileProviderName, &session.ManagerConfig{
		CookieName:      web.BConfig.WebConfig.Session.SessionName,
		EnableSetCookie: true,
		ProviderConfig:  t.TempDir(),
		Gclifetime:      3600,
		Secure:          true,
		CookieSameSite:  http.SameSiteLaxMode,
	})
	if err != nil {
		t.Fatal(err)
	}
	return manager
}

func readStore(t *testing.T, provider session.Provider, sid string) session.Store {
	t.Helper()
	store, err := provider.SessionRead(t.Context(), sid)
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestCommitPreservesUnchangedKeys(t *testing.T) {
	manager := testManager(t)
	provider := manager.GetProvider()
	const sid = "1234567890abcdef"

	// A session already written by an older Casnode release remains readable.
	oldProvider, err := session.GetProvider("file")
	must(t, err)
	old := readStore(t, oldProvider, sid)
	must(t, old.Set(t.Context(), "language", "en"))
	old.SessionRelease(t.Context(), nil)

	first := readStore(t, provider, sid)
	second := readStore(t, provider, sid)
	anonymousReader := readStore(t, provider, sid)
	must(t, first.Set(t.Context(), "displayMode", "compact"))
	first.SessionRelease(t.Context(), nil)
	must(t, second.Set(t.Context(), "language", "zh"))
	second.SessionRelease(t.Context(), nil)
	anonymousReader.SessionRelease(t.Context(), nil)

	persisted := readStore(t, provider, sid)
	if actual := persisted.Get(t.Context(), "displayMode"); actual != "compact" {
		t.Fatalf("display mode was %v, want compact", actual)
	}
	if actual := persisted.Get(t.Context(), "language"); actual != "zh" {
		t.Fatalf("language was %v, want zh", actual)
	}

	must(t, persisted.Delete(t.Context(), "displayMode"))
	persisted.SessionRelease(t.Context(), nil)
	if actual := readStore(t, provider, sid).Get(t.Context(), "displayMode"); actual != nil {
		t.Fatalf("deleted display mode was still present: %v", actual)
	}
}

func TestSessionCookieRequiresSavedData(t *testing.T) {
	manager := testManager(t)
	ctx := context.Background()
	cookieName := web.BConfig.WebConfig.Session.SessionName
	startRequest := func(cookie *http.Cookie) (*httptest.ResponseRecorder, *webcontext.Context) {
		t.Helper()
		request := httptest.NewRequest(http.MethodGet, "https://forum.example.test/", nil)
		if cookie != nil {
			request.AddCookie(cookie)
		}
		recorder := httptest.NewRecorder()
		http.SetCookie(recorder, &http.Cookie{Name: "unrelated", Value: "keep"})
		store, err := manager.SessionStart(recorder, request)
		must(t, err)
		webCtx := webcontext.NewContext()
		webCtx.Reset(recorder, request)
		webCtx.Input.CruSession = store
		PrepareResponse(webCtx)
		return recorder, webCtx
	}

	recorder, anonymous := startRequest(nil)
	anonymous.ResponseWriter.WriteHeader(http.StatusOK)
	reader := anonymous.Input.CruSession
	reader.SessionRelease(ctx, recorder)
	cookies := recorder.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != "unrelated" {
		t.Fatalf("anonymous response cookies = %v, want only unrelated", cookies)
	}
	exists, err := manager.GetProvider().SessionExist(ctx, reader.SessionID(ctx))
	must(t, err)
	if exists {
		t.Fatal("an anonymous request created a file session")
	}

	recorder, signin := startRequest(nil)
	writer := signin.Input.CruSession
	must(t, writer.Set(ctx, "language", "en"))
	signin.ResponseWriter.Header().Set("Location", "/account")
	signin.ResponseWriter.WriteHeader(http.StatusFound)
	if got := readStore(t, manager.GetProvider(), writer.SessionID(ctx)).Get(ctx, "language"); got != "en" {
		t.Fatalf("language at redirect = %v, want en", got)
	}
	cookies = recorder.Result().Cookies()
	if len(cookies) != 2 || cookies[0].Name != "unrelated" || cookies[1].Name != cookieName {
		t.Fatalf("stateful response cookies = %v, want unrelated and forum session", cookies)
	}
	if !cookies[1].HttpOnly || !cookies[1].Secure || cookies[1].SameSite != http.SameSiteLaxMode {
		t.Fatal("session cookie attributes changed")
	}

	_, signout := startRequest(cookies[1])
	loggedIn := signout.Input.CruSession
	must(t, loggedIn.Delete(ctx, "language"))
	_, err = signout.ResponseWriter.Write([]byte("signed out"))
	must(t, err)
	if got := readStore(t, manager.GetProvider(), loggedIn.SessionID(ctx)).Get(ctx, "language"); got != nil {
		t.Fatalf("language was still saved after writing the sign-out response: %v", got)
	}
}

func TestFailedSaveStopsRedirect(t *testing.T) {
	manager := testManager(t)
	ctx := t.Context()
	provider := manager.GetProvider()
	const sid = "1234567890abcdef"
	session := readStore(t, provider, sid)
	must(t, session.Set(ctx, "language", "en"))
	session.SessionRelease(ctx, nil)

	recorder := httptest.NewRecorder()
	webCtx := webcontext.NewContext()
	webCtx.Reset(recorder, httptest.NewRequest(http.MethodGet, "https://forum.example.test/", nil))
	webCtx.Input.CruSession = readStore(t, provider, sid)
	PrepareResponse(webCtx)
	must(t, webCtx.Input.CruSession.Set(ctx, "language", "zh"))
	must(t, provider.SessionDestroy(ctx, sid))

	webCtx.ResponseWriter.Header().Set("Location", "/account")
	webCtx.ResponseWriter.WriteHeader(http.StatusFound)
	if response := recorder.Result(); response.StatusCode != http.StatusServiceUnavailable || response.Header.Get("Location") != "" {
		t.Fatalf("response = %d with redirect %q after a failed save", response.StatusCode, response.Header.Get("Location"))
	}
}
