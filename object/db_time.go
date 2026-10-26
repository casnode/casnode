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

import "time"

// dbTime is a time for a WHERE clause, written the way xorm stores times (local time, to the second). Passing a
// time.Time directly lets each driver format it its own way (MySQL in UTC, SQLite with a zone), which then fails to
// compare with the stored values.
func dbTime(t time.Time) string {
	return t.In(time.Local).Format("2006-01-02 15:04:05")
}
