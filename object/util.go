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
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

func uniqueIds(ids []int64) []int64 {
	seen := map[int64]bool{}
	res := []int64{}
	for _, id := range ids {
		if id == 0 || seen[id] {
			continue
		}
		seen[id] = true
		res = append(res, id)
	}
	return res
}

func containsId(ids []int64, id int64) bool {
	for _, item := range ids {
		if item == id {
			return true
		}
	}
	return false
}

// escapeLike drops the LIKE wildcard "%" from a search term; databases disagree on how to escape it.
func escapeLike(s string) string {
	return strings.ReplaceAll(s, "%", "")
}

var slugInvalid = regexp.MustCompile(`[^\p{L}\p{N}]+`)

// Slugify makes the URL part of a title: lower case, letters and digits joined by "-". A title without any (emoji
// only, say) gets "topic".
func Slugify(s string) string {
	s = norm.NFKC.String(strings.ToLower(s))
	s = slugInvalid.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	runes := []rune(s)
	if len(runes) > 80 {
		s = strings.TrimRight(string(runes[:80]), "-")
	}
	if s == "" {
		return "topic"
	}
	return s
}

func isAsciiWord(s string) bool {
	for _, r := range s {
		if r > unicode.MaxASCII {
			return false
		}
	}
	return true
}

func truncateRunes(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "…"
}

func errorf(format string, a ...interface{}) error {
	return fmt.Errorf(format, a...)
}

func itoa(n int64) string {
	return strconv.FormatInt(n, 10)
}

func toJson(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(data)
}
