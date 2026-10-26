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
	"strings"
	"time"

	"github.com/casnode/casnode/conf"
	_ "github.com/go-sql-driver/mysql"  // db = mysql
	_ "github.com/lib/pq"               // db = postgres
	_ "github.com/microsoft/go-mssqldb" // db = mssql
	"github.com/xorm-io/xorm"
	"github.com/xorm-io/xorm/names"
	_ "modernc.org/sqlite" // db = sqlite
)

var adapter *Adapter

type Adapter struct {
	driverName     string
	dataSourceName string
	dbName         string
	Engine         *xorm.Engine
}

// InitAdapter connects to the database in app.conf, creating the database and tables if needed.
func InitAdapter() {
	var err error
	adapter, err = NewAdapter(conf.GetConfigString("driverName"), conf.GetConfigString("dataSourceName"), conf.GetConfigString("dbName"))
	if err != nil {
		panic(err)
	}

	err = adapter.createTables()
	if err != nil {
		panic(err)
	}
}

func NewAdapter(driverName string, dataSourceName string, dbName string) (*Adapter, error) {
	a := &Adapter{driverName: driverName, dataSourceName: dataSourceName, dbName: dbName}
	err := a.createDatabase()
	if err != nil {
		return nil, err
	}

	a.Engine, err = xorm.NewEngine(a.driverName, a.getSource(a.dbName))
	if err != nil {
		return nil, err
	}
	a.Engine.SetMapper(names.SnakeMapper{})
	a.Engine.SetConnMaxLifetime(5 * time.Minute)
	if a.driverName == "sqlite" {
		a.Engine.SetMaxOpenConns(1)
	}
	return a, nil
}

// getSource is the connection string for the database dbName ("" = the server without a database).
func (a *Adapter) getSource(dbName string) string {
	switch a.driverName {
	case "sqlite":
		// one writer at a time: wait for the lock instead of failing with "database is locked"
		return fmt.Sprintf("file:%s?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)", dbName)
	case "mysql":
		source := a.dataSourceName
		if !strings.HasSuffix(source, "/") {
			source += "/"
		}
		return source + dbName
	case "postgres":
		if dbName == "" {
			dbName = "postgres"
		}
		return fmt.Sprintf("%s dbname=%s", a.dataSourceName, dbName)
	case "mssql":
		source := strings.TrimSuffix(a.dataSourceName, "/")
		if dbName == "" {
			return source
		}
		separator := "?"
		if strings.Contains(source, "?") {
			separator = "&"
		}
		return source + separator + "database=" + dbName
	default:
		return a.dataSourceName + dbName
	}
}

func (a *Adapter) createDatabase() error {
	if a.driverName == "sqlite" {
		return nil
	}

	engine, err := xorm.NewEngine(a.driverName, a.getSource(""))
	if err != nil {
		return err
	}
	defer engine.Close()

	switch a.driverName {
	case "mysql":
		_, err = engine.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci", a.dbName))
	case "postgres":
		var count int64
		_, err = engine.SQL("SELECT COUNT(*) FROM pg_database WHERE datname = ?", a.dbName).Get(&count)
		if err == nil && count == 0 {
			_, err = engine.Exec(fmt.Sprintf(`CREATE DATABASE "%s" ENCODING 'UTF8'`, a.dbName))
		}
	case "mssql":
		_, err = engine.Exec(fmt.Sprintf("IF DB_ID('%s') IS NULL CREATE DATABASE [%s]", a.dbName, a.dbName))
	}
	return err
}

func (a *Adapter) createTables() error {
	tables := []interface{}{
		new(User),
		new(Category),
		new(Topic),
		new(Post),
		new(PostRevision),
		new(PostAction),
		new(TopicUser),
		new(TopicAllowedUser),
		new(Tag),
		new(TopicTag),
		new(Notification),
		new(Reviewable),
		new(Upload),
		new(SiteSetting),
		new(UserVisit),
	}
	for _, table := range tables {
		err := a.Engine.Sync2(table)
		if err != nil {
			return err
		}
	}
	return nil
}

func IsSqlite() bool {
	return adapter.driverName == "sqlite"
}
