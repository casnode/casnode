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
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/beego/beego/v2/server/web/context"
	"github.com/casnode/casnode/object"
)

var frontendProxy *httputil.ReverseProxy

// InitFrontendProxy points the pages at the Next.js server (StartFrontend) listening on port.
func InitFrontendProxy(port int) {
	target := &url.URL{Scheme: "http", Host: "127.0.0.1:" + strconv.Itoa(port)}
	frontendProxy = httputil.NewSingleHostReverseProxy(target)
	director := frontendProxy.Director
	frontendProxy.Director = func(r *http.Request) {
		// the original Host keeps the absolute URLs Next writes right; compression is done here
		host := r.Host
		director(r)
		r.Host = host
		r.Header.Set("X-Accept-Encoding", r.Header.Get("Accept-Encoding"))
		r.Header.Del("Accept-Encoding")
	}
	frontendProxy.ModifyResponse = compressFrontendResponse
	frontendProxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Retry-After", "10")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("The site is being updated, please refresh in a moment."))
	}
}

// StaticFilter leaves /api/ to the controllers, serves uploads, feeds and sitemaps itself, and sends every other
// request to the Next.js server, which renders the pages.
func StaticFilter(ctx *context.Context) {
	path := ctx.Request.URL.Path
	if strings.HasPrefix(path, "/api/") {
		return
	}
	if strings.HasPrefix(path, "/uploads/") {
		serveUpload(ctx)
		return
	}
	if serveFeed(ctx) {
		return
	}
	frontendProxy.ServeHTTP(ctx.ResponseWriter, ctx.Request)
}

func serveUpload(ctx *context.Context) {
	path := object.GetUploadPath(ctx.Request.URL.Path)
	info, err := os.Stat(path)
	if path == "" || err != nil || info.IsDir() {
		http.NotFound(ctx.ResponseWriter, ctx.Request)
		return
	}
	file, err := os.Open(path)
	if err != nil {
		http.Error(ctx.ResponseWriter, err.Error(), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	header := ctx.ResponseWriter.Header()
	// files are named by their hash, so they never change; an uploaded SVG or HTML must not run scripts on our origin
	header.Set("Cache-Control", "public, max-age=31536000, immutable")
	header.Set("Content-Security-Policy", "default-src 'none'; img-src 'self' data:; style-src 'unsafe-inline'; sandbox")
	header.Set("X-Content-Type-Options", "nosniff")
	if contentType, ok := uploadContentTypes[strings.ToLower(path[strings.LastIndex(path, ".")+1:])]; ok {
		header.Set("Content-Type", contentType)
	} else {
		header.Set("Content-Type", "application/octet-stream")
		header.Set("Content-Disposition", "attachment")
	}
	http.ServeContent(ctx.ResponseWriter, ctx.Request, info.Name(), info.ModTime(), file)
}

// Set explicitly: on Windows, Go reads MIME types from the registry, which is not always right.
var uploadContentTypes = map[string]string{
	"jpg":  "image/jpeg",
	"jpeg": "image/jpeg",
	"png":  "image/png",
	"gif":  "image/gif",
	"webp": "image/webp",
	"svg":  "image/svg+xml",
	"pdf":  "application/pdf",
	"txt":  "text/plain; charset=utf-8",
}

func compressFrontendResponse(resp *http.Response) error {
	contentType := resp.Header.Get("Content-Type")
	if !isCompressible(contentType) || resp.Header.Get("Content-Encoding") != "" {
		return nil
	}
	resp.Header.Add("Vary", "Accept-Encoding")
	if !acceptsGzip(resp.Request.Header.Get("X-Accept-Encoding")) {
		return nil
	}
	// the payloads of client-side navigation are streamed, send them on as they come
	if strings.HasPrefix(contentType, "text/x-component") {
		return nil
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()

	if len(data) > 1024 {
		var buf bytes.Buffer
		zw := gzip.NewWriter(&buf)
		_, _ = zw.Write(data)
		_ = zw.Close()
		data = buf.Bytes()
		resp.Header.Set("Content-Encoding", "gzip")
	}
	resp.Header.Set("Content-Length", strconv.Itoa(len(data)))
	resp.Body = io.NopCloser(bytes.NewReader(data))
	return nil
}

func isCompressible(contentType string) bool {
	return strings.HasPrefix(contentType, "text/") || strings.HasPrefix(contentType, "application/json") ||
		strings.HasPrefix(contentType, "application/xml") || strings.HasPrefix(contentType, "image/svg+xml") ||
		strings.HasPrefix(contentType, "application/javascript")
}

func acceptsGzip(acceptEncoding string) bool {
	for _, part := range strings.Split(acceptEncoding, ",") {
		coding, params, _ := strings.Cut(strings.TrimSpace(part), ";")
		if strings.EqualFold(strings.TrimSpace(coding), "gzip") && strings.ReplaceAll(strings.TrimSpace(params), " ", "") != "q=0" {
			return true
		}
	}
	return false
}
