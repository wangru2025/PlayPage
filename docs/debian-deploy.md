# Debian Single-Machine Deploy

## Services

- `angie`
- `postgresql`
- `nodejs` 22+
- `golang` 1.24+

## Process layout

- Web control panel: `127.0.0.1:3000`
- Go API: `127.0.0.1:8080`
- Angie: public `80/443`

## Directories

- `/data/apps/{project_id}/uploads`
- `/data/apps/{project_id}/runtime`
- `/data/apps/{project_id}/releases/{release_id}/public`
- `/data/public/@username/project-slug`

## Publish flow

1. User uploads ZIP to the Go API.
2. Go stores the archive under the project uploads directory.
3. Go extracts validated static files into a new release directory.
4. Go updates the live path under `/data/public/@username/project-slug`.
5. Angie serves the published site directly from disk.
