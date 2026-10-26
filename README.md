<h1 align="center">Casnode</h1>

<p align="center">
  <b>An open-source forum with the look and features of Discourse, built with Go and Next.js.</b><br>
  Fast server-rendered pages for search engines, sign-in through <a href="https://casdoor.ai">Casdoor</a>, and your choice of MySQL, PostgreSQL, SQLite or SQL Server.
</p>

<p align="center">
  <a href="https://github.com/casnode/casnode/actions/workflows/build.yml"><img alt="Build" src="https://github.com/casnode/casnode/actions/workflows/build.yml/badge.svg"></a>
  <a href="https://github.com/casnode/casnode/releases/latest"><img alt="Release" src="https://img.shields.io/github/v/release/casnode/casnode?sort=semver"></a>
  <a href="https://goreportcard.com/report/github.com/casnode/casnode"><img alt="Go Report Card" src="https://goreportcard.com/badge/github.com/casnode/casnode"></a>
  <a href="https://github.com/casnode/casnode/blob/master/go.mod"><img alt="Go version" src="https://img.shields.io/github/go-mod/go-version/casnode/casnode?logo=go"></a>
  <a href="https://nextjs.org"><img alt="Next.js" src="https://img.shields.io/badge/Next.js-16-black?logo=next.js"></a>
  <a href="https://github.com/casnode/casnode/blob/master/LICENSE"><img alt="License" src="https://img.shields.io/github/license/casnode/casnode"></a>
</p>

<p align="center">
  <a href="https://forum.casbin.com"><img alt="Live demo" src="https://img.shields.io/badge/demo-forum.casbin.com-0088cc"></a>
  <a href="https://github.com/casnode/casnode/stargazers"><img alt="Stars" src="https://img.shields.io/github/stars/casnode/casnode?style=flat"></a>
  <a href="https://github.com/casnode/casnode/network/members"><img alt="Forks" src="https://img.shields.io/github/forks/casnode/casnode?style=flat"></a>
  <a href="https://github.com/casnode/casnode/issues"><img alt="Issues" src="https://img.shields.io/github/issues/casnode/casnode"></a>
  <a href="https://github.com/casnode/casnode/commits/master"><img alt="Last commit" src="https://img.shields.io/github/last-commit/casnode/casnode"></a>
  <a href="https://discord.gg/d8dsAWgn43"><img alt="Discord" src="https://img.shields.io/discord/1022748306096537660?logo=discord&label=discord&color=5865F2"></a>
</p>

<p align="center">
  <a href="https://forum.casbin.com">Live demo</a> |
  <a href="#quick-start">Quick start</a> |
  <a href="#configuration">Configuration</a> |
  <a href="#deployment">Deployment</a> |
  <a href="https://discord.gg/d8dsAWgn43">Discord</a>
</p>

![Latest topics](docs/images/latest.png)

## Why Casnode

Discourse is the forum most people know how to use, but running it means Ruby, Ember, PostgreSQL, Redis and a Docker
image of several gigabytes. Casnode gives your community the same experience with a single Go binary and a Next.js
server:

- **Looks and works like Discourse.** The topic list, the topic page with its timeline, the composer docked at the bottom,
  the user menu, notifications and the admin pages follow Discourse's default theme closely, so your members feel at home.
- **Made for search engines.** Every topic, category, tag and profile page is rendered on the server with canonical URLs,
  Open Graph, JSON-LD (`DiscussionForumPosting`), sitemaps, RSS feeds and crawler-friendly pagination.
- **Runs on the database you already have.** [xorm](https://github.com/xorm-io/xorm) talks to MySQL, PostgreSQL, SQLite
  and SQL Server; the database and tables are created on first start.
- **Single sign-on out of the box.** Members sign in with [Casdoor](https://github.com/casdoor/casdoor), which brings
  GitHub, Google, WeChat, SAML, LDAP and 100+ other providers.

## Features

| | |
| --- | --- |
| **Topic lists** | Latest, New, Unread, Hot, Top (year, quarter, month, week, day), Categories; per category and per tag; pinned topics, unread counts, poster avatars, endless scrolling |
| **Topics** | Post stream that loads in both directions, timeline, topic map (views, likes, users, reading time), replies shown in place, likes, bookmarks, quoting selected text, edit history, closed / pinned / archived / unlisted notices, suggested topics |
| **Composer** | Markdown with live preview and toolbar, image and file uploads (paste or drag), `@mentions`, categories and tags, drafts, Discourse's `[quote]` and `[details]` blocks, emoji shortcodes |
| **Tracking** | Reading position, New and Unread topics, notification levels (Watching, Tracking, Normal, Muted), notifications for mentions, replies, quotes, likes and messages |
| **People** | Personal messages, profiles with summary and activity, preferences, the Users directory |
| **Search** | Full-text search with `@user`, `#category`, `tags:name`, `in:title`, `in:messages`, `status:open`, `order:latest\|likes\|views`, and quick results in the header |
| **Moderation** | Flags and the review queue; closing, pinning, archiving, unlisting, deleting and recovering topics and posts; suspending and silencing users; staff-only categories |
| **Admin** | Dashboard, site settings (name, logo, menus, posting limits, guidelines, analytics), users, categories and tags |
| **Interface** | English and Chinese, light and dark (follows the system), works on phones |

<table>
  <tr>
    <td><img alt="A topic" src="docs/images/topic.png"></td>
    <td><img alt="Categories" src="docs/images/categories.png"></td>
  </tr>
</table>

## Tech stack

| Part | Built with |
| --- | --- |
| Backend | Go, [beego](https://github.com/beego/beego), [xorm](https://github.com/xorm-io/xorm), [goldmark](https://github.com/yuin/goldmark) + [bluemonday](https://github.com/microcosm-cc/bluemonday) for Markdown |
| Frontend | [Next.js](https://nextjs.org) App Router (React Server Components), [shadcn/ui](https://ui.shadcn.com) on Radix, Tailwind CSS |
| Sign-in | [Casdoor](https://casdoor.ai) (OAuth 2.0 / OIDC) |
| Database | MySQL, PostgreSQL, SQLite, SQL Server |

```
browser ──> Go backend (httpport) ──> /api, /uploads, RSS, sitemaps
                    │
                    └──> Next.js server (httpport + 1) ──> server-rendered pages
```

## Quick start

### Docker

```bash
docker build -t casnode .
docker run -d -p 16900:16900 -v casnode-data:/app/data \
  -e casdoorEndpoint=https://door.casdoor.com -e clientId=<client id> -e clientSecret=<client secret> \
  -e casdoorOrganization=<organization> -e casdoorApplication=<application> -e admins=<your user name> \
  casnode
```

Open http://localhost:16900. The SQLite database and the uploads are kept in the `casnode-data` volume; see
[Configuration](#configuration) for MySQL or PostgreSQL.

### From source

Requirements: Go 1.25+, Node.js 20+, Yarn 1.

```bash
git clone https://github.com/casnode/casnode
cd casnode
echo "devSignin = true" > conf/app.local.conf
go run .                         # backend on http://localhost:16900, SQLite file casnode.db
```

```bash
cd web
yarn install
yarn dev                         # frontend on http://localhost:13900 with hot reload
```

Open http://localhost:13900/login and sign in as `admin` (the users in `admins` administer the forum).

## Configuration

`conf/app.conf` holds the defaults. Put your own settings and secrets in `conf/app.local.conf` (same format, ignored by
git); every item can also be set as an environment variable of the same name.

| Item | Default | Meaning |
| --- | --- | --- |
| `httpport` | `16900` | Port of the Go backend; the Next.js server listens on `httpport + 1` |
| `driverName` | `sqlite` | `sqlite`, `mysql`, `postgres` or `mssql` |
| `dataSourceName` | | `root:123456@tcp(localhost:3306)/` (MySQL), `user=postgres password=123456 host=localhost port=5432 sslmode=disable` (PostgreSQL), `sqlserver://sa:123456@localhost:1433` (SQL Server) |
| `dbName` | `casnode.db` | Database name, or the file name for SQLite |
| `casdoorEndpoint` | | Your Casdoor, e.g. `https://door.casdoor.com` |
| `clientId`, `clientSecret` | | Of the Casdoor application; add `https://<your forum>/callback` to its redirect URLs |
| `casdoorOrganization`, `casdoorApplication` | | Only users of this organization can sign in |
| `admins` | `admin` | User names that administer the forum, besides Casdoor's own admins |
| `uploadsDir` | `files/uploads` | Where uploaded images and files are kept |
| `trustProxyHeader` | `false` | Behind a reverse proxy, take the client IP from `X-Forwarded-For` |
| `devSignin` | `false` | Sign in without Casdoor, from the same machine only; never turn it on in production |

Everything else (site name, logo, top menu, posting limits, guidelines, terms, privacy policy, analytics IDs) is changed
by admins on `/admin/site_settings`.

## Deployment

```bash
cd web && yarn install && yarn build && cd ..
go build -o casnode .
./casnode
```

`yarn build` writes the Next.js server to `web/build/server-<time>` and names it in `web/build/server.txt`; the Go
backend starts it on `httpport + 1`, forwards page requests to it and serves `/api/`, uploads, RSS feeds and sitemaps
itself. Each build goes to a new folder, so you can build while the forum is running and restart the backend afterwards.
Put Nginx, Caddy or [CasWAF](https://github.com/caswaf/caswaf) in front for HTTPS, and keep `uploadsDir` and the
database backed up.

## Development

```bash
gofmt -l . && go vet ./...       # backend checks, run by CI
cd web && yarn lint && yarn build
```

The backend is in the repository root (`object/` for the data model, `controllers/` for the API, `routers/` for routing,
feeds and the proxy to Next.js); the frontend is in `web/src` (`app/` for pages, `components/` for the UI,
`lib/locales/` for the translations). Pull requests are welcome; commit messages follow
[Conventional Commits](https://www.conventionalcommits.org) so that releases are made automatically.

## Community

- Live demo and support forum: [forum.casbin.com](https://forum.casbin.com)
- Chat: [Discord](https://discord.gg/d8dsAWgn43)
- Bugs and feature requests: [GitHub Issues](https://github.com/casnode/casnode/issues)

## License

[Apache-2.0](LICENSE). Casnode is an independent implementation inspired by Discourse's user interface; it contains no
code from Discourse.
