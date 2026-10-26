# Casnode

Casnode is a forum with the look and features of [Discourse](https://www.discourse.org), written in Go and Next.js and
signed in through [Casdoor](https://casdoor.ai).

- **Backend**: Go + [beego](https://github.com/beego/beego), with [xorm](https://github.com/xorm-io/xorm) on SQLite, MySQL,
  PostgreSQL or SQL Server; the database and tables are created on first start.
- **Frontend**: [Next.js](https://nextjs.org) (App Router) + [shadcn/ui](https://ui.shadcn.com), rendered on the server
  so that every topic, category, tag and profile page is complete HTML for search engines.

## Features

- Topic lists: Latest, New, Unread, Hot, Top (by year, quarter, month, week, day), Categories, per category and per tag,
  with pinned topics, unread counts, poster avatars and endless scrolling
- Topics: the post stream with endless loading both ways, the timeline, the topic map (views, likes, users, reading
  time), replies to a post shown in place, likes, bookmarks, copy link, quoting selected text, edit history, small action
  posts (closed, pinned, archived, unlisted), suggested topics, reading position and New / Unread tracking, notification
  levels (Watching, Tracking, Normal, Muted)
- The composer docked at the bottom: Markdown with live preview, toolbar, image and file uploads (paste or drag),
  @mentions, categories and tags, drafts kept in the browser; Discourse's `[quote]` and `[details]` blocks
- Personal messages, notifications (mentions, replies, quotes, likes, messages) with the user menu
- Users: profiles with summary and activity, preferences, the Users directory
- Search with Discourse's filters: `@user`, `#category`, `tags:name`, `in:title`, `in:messages`, `status:open`,
  `order:latest|likes|views`
- Moderation: flags and the review queue, closing, pinning, archiving, unlisting, deleting and recovering topics and posts,
  suspending and silencing users, staff-only categories
- Admin: dashboard, site settings, users, categories and tags
- SEO: server-rendered pages, canonical URLs, Open Graph, JSON-LD (`DiscussionForumPosting`), `?page=N` pages for crawlers,
  `sitemap.xml`, `robots.txt`, RSS feeds (`/latest.rss`, `/top.rss`, `/c/<slug>/<id>.rss`, `/tag/<name>.rss`,
  `/t/<slug>/<id>.rss`)
- English and Chinese, light and dark (follows the system)

## Development

Requirements: Go 1.26+, Node.js 22+, Yarn 1.

```bash
# backend on http://localhost:16900, SQLite file casnode.db in the repository root
go run .
```

```bash
# frontend on http://localhost:13900, with hot reload; /api goes to the backend
cd web
yarn install
yarn dev
```

Without Casdoor, sign in locally with any user name: put `devSignin = true` in `conf/app.local.conf` and open
http://localhost:13900/login. Users named in `admins` (default `admin`) are administrators.

## Configuration

`conf/app.conf` has the defaults. A server's own settings and secrets go in `conf/app.local.conf` (same format, not in
git), and any item can also be set as an environment variable of the same name.

| Item | Meaning |
| --- | --- |
| `httpport` | the Go backend; the Next.js server runs on `httpport + 1` |
| `driverName`, `dataSourceName`, `dbName` | `sqlite` (dbName is the file), `mysql` (`root:123456@tcp(localhost:3306)/`), `postgres` (`user=postgres password=123456 host=localhost port=5432 sslmode=disable`) or `mssql` (`sqlserver://sa:123456@localhost:1433`) |
| `casdoorEndpoint`, `clientId`, `clientSecret`, `casdoorOrganization`, `casdoorApplication` | sign-in through Casdoor; add `https://<your site>/callback` to the application's redirect URLs |
| `admins` | user names that administer the forum, besides Casdoor's own admins |
| `uploadsDir` | where uploaded files are kept |
| `trustProxyHeader` | behind a reverse proxy, take the client IP from `X-Forwarded-For` |

Everything else (site name, logo, menus, posting limits, analytics IDs, about / FAQ / terms / privacy texts) is set by
admins on `/admin/site_settings`.

## Production

```bash
cd web && yarn install && yarn build && cd ..
go build -o casnode . && ./casnode
```

`yarn build` puts the Next.js server in `web/build/server-<time>` and names it in `web/build/server.txt`; the Go
backend starts it, sends the pages to it and answers `/api/`, uploads, feeds and sitemaps itself. Or use the
`Dockerfile`: data (SQLite file and uploads) is kept in `/app/data`.

## License

[Apache-2.0](LICENSE). Casnode is an independent implementation; it contains no code from Discourse.
