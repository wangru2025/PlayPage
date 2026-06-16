# AI Static Host

Single-machine MVP for hosting static AI-generated projects with a Go API, a Next.js control panel, PostgreSQL metadata, and Angie/Nginx in front of the app.

## Layout

- `apps/api`: Go service bound to `127.0.0.1`
- `apps/web`: Next.js control panel
- `config/angie`: reverse-proxy and static site examples
- `db/migrations`: PostgreSQL schema

## Runtime shape

- Angie serves public traffic on `80/443`
- Go API listens on `127.0.0.1:8080`
- Static releases are served from disk under `/data/apps/{project_id}/releases/{release_id}/public`
- The active public path for a project is exposed through `/data/public/@username/project-slug`
- PostgreSQL stores users, projects, releases, collections, records, subscriptions, and moderation metadata

## Local setup

### Web

```bash
cd apps/web
npm install
npm run dev
```

### API

Install Go 1.24+ first, then:

```bash
cd apps/api
go run ./cmd/server
```

The API expects a PostgreSQL database URL in `DATABASE_URL`.

## Debian deployment shape

- `Angie` serves `@username/project-slug` public paths from `/data/public`
- `Go` service runs behind `127.0.0.1:8080`
- Each publish creates a release under `/data/apps/{project_id}/releases/{release_id}/public`
- The current live release is linked to `/data/public/@username/project-slug`

Example services to install on the Debian box:

- `angie`
- `postgresql`
- `golang` 1.24+
- `nodejs` 22+

## MVP status

This repository contains the initial product skeleton:

- control panel shell
- API routing and JSON contracts
- PostgreSQL migration for core entities
- local-disk release layout helper
- Angie sample configuration

The next implementation step is wiring the handlers to PostgreSQL and adding upload/publish execution.
