# unjbr

A Go web server using only the standard library.

## Layout

```
cmd/server/        entrypoint: config, graceful shutdown
internal/server/   routes, handlers, request logging
web/templates/     HTML templates (layout.html + one file per page)
web/static/        static assets, served at /static/
```

Templates and static files are embedded into the binary with `embed`.

## Usage

```sh
make run      # http://localhost:8080 (override with PORT=...)
make test
make build    # outputs bin/server
```

## Adding a page

1. Create `web/templates/<name>.html` defining a `content` block.
2. Register a route in `internal/server/server.go` and call `s.render(w, "<name>.html", data)`.

## Endpoints

- `GET /` home page
- `GET /healthz` health check
- `GET /static/...` static assets
