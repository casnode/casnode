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
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/beego/beego/v2/core/logs"
)

// A CDN can serve the frontend's static files (scripts, styles, fonts). CasWAF uploads a build's .next-build/static
// to its storage provider and writes the CDN's URL of the `_next/` folder, like https://cdn.example.com/x/_next/, to
// cdn.txt beside the build's server.js; the pages then point their `/_next/static/` URLs there. Without cdn.txt (no
// CDN, or `yarn dev`) everything is served from this site.
var frontendCdnUrl string

var cdnUrlRegex = regexp.MustCompile(`^https?://[^\s"'<>\\]+/$`)

func loadFrontendCdnUrl(frontendDir string) {
	data, err := os.ReadFile(filepath.Join(frontendDir, "cdn.txt"))
	if err != nil {
		frontendCdnUrl = ""
		return
	}
	cdnUrl := strings.TrimSpace(string(data))
	if !strings.HasSuffix(cdnUrl, "/") {
		cdnUrl += "/"
	}
	if !cdnUrlRegex.MatchString(cdnUrl) {
		logs.Warn("loadFrontendCdnUrl(): ignoring the invalid CDN URL %q", cdnUrl)
		frontendCdnUrl = ""
		return
	}
	frontendCdnUrl = cdnUrl
	logs.Info("the frontend's static files are served from %s", cdnUrl)
}

// The pages and the router payloads quote the static files as "/_next/static/..." (escaped as \"/_next/static/...\"
// inside the inline payloads, which this matches too).
var nextStaticPattern = []byte(`"/_next/static/`)

func cdnStaticPrefix() []byte {
	return []byte(`"` + frontendCdnUrl + "static/")
}

// rewriteHtmlForCdn points the page's static files at the CDN, and tells Turbopack's runtime to load the chunks it
// adds later from there too.
func rewriteHtmlForCdn(html []byte) []byte {
	html = bytes.ReplaceAll(html, nextStaticPattern, cdnStaticPrefix())
	script := []byte(`<script>self.TURBOPACK_CHUNK_BASE_PATH="` + frontendCdnUrl + `"</script>`)
	if index := bytes.Index(html, []byte("<head>")); index >= 0 {
		end := index + len("<head>")
		return append(html[:end:end], append(script, html[end:]...)...)
	}
	return html
}

// rewriteLinkHeaderForCdn points the preloads Next lists in the Link header (</_next/static/...>; rel=preload) at
// the CDN too, or the browser would fetch those files from both places.
func rewriteLinkHeaderForCdn(header http.Header) {
	values := header.Values("Link")
	if len(values) == 0 {
		return
	}
	header.Del("Link")
	for _, value := range values {
		header.Add("Link", strings.ReplaceAll(value, "</_next/static/", "<"+frontendCdnUrl+"static/"))
	}
}

// cdnStreamRewriter does the same replacement on a streamed router payload, holding back the end of each read in
// case the pattern is split between two reads.
type cdnStreamRewriter struct {
	src     io.ReadCloser
	pending []byte
	out     []byte
	eof     bool
}

func newCdnStreamRewriter(src io.ReadCloser) io.ReadCloser {
	return &cdnStreamRewriter{src: src}
}

func (r *cdnStreamRewriter) Read(p []byte) (int, error) {
	for len(r.out) == 0 {
		if r.eof {
			if len(r.pending) == 0 {
				return 0, io.EOF
			}
			r.out, r.pending = r.pending, nil
			break
		}
		buf := make([]byte, 32*1024)
		n, err := r.src.Read(buf)
		r.pending = append(r.pending, buf[:n]...)
		if err == io.EOF {
			r.eof = true
		} else if err != nil {
			return 0, err
		}
		r.pending = bytes.ReplaceAll(r.pending, nextStaticPattern, cdnStaticPrefix())
		keep := len(nextStaticPattern) - 1
		if r.eof || len(r.pending) <= keep {
			if r.eof {
				r.out, r.pending = r.pending, nil
			}
			continue
		}
		cut := len(r.pending) - keep
		r.out = append(r.out[:0], r.pending[:cut]...)
		r.pending = append([]byte(nil), r.pending[cut:]...)
	}
	n := copy(p, r.out)
	r.out = r.out[n:]
	return n, nil
}

func (r *cdnStreamRewriter) Close() error {
	return r.src.Close()
}
