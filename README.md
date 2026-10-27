# unjbr

A simple browser-based video chat with a single meeting and no
authentication. Built with the Go standard library and plain JavaScript.

## How it works

- Browsers connect directly to each other with WebRTC. Every participant
  connects to every other one, so audio and video never pass through the
  server.
- The server only relays the messages browsers use to set up those
  connections. It pushes them to each browser over Server-Sent Events
  (`GET /events`) and receives them as `POST /signal` requests.
- A STUN server (`stun.l.google.com`) helps browsers find each other. There
  is no TURN server, so peers behind restrictive NATs or firewalls may fail
  to connect.
- Because everyone connects to everyone, it works best with a small group
  (roughly up to 4–6 people).

## Layout

```
cmd/server/        entrypoint: config, TLS, graceful shutdown
internal/server/   routes, page rendering, meeting room and signaling relay
web/templates/     HTML templates
web/static/        client script (app.js) and styles
```

## Usage

```sh
make run      # http://localhost:8080 (override with PORT=...)
make test
make build    # outputs bin/server
```

Browsers only allow camera and microphone access on `localhost` or over
HTTPS. To join from other machines, serve with a certificate, for example
one made with [mkcert](https://github.com/FiloSottile/mkcert):

```sh
mkcert 192.168.1.10 localhost
TLS_CERT=192.168.1.10+1.pem TLS_KEY=192.168.1.10+1-key.pem make run
```

## Endpoints

- `GET /` meeting page
- `GET /events` signaling event stream (joins the meeting)
- `POST /signal` relay a signaling message to another participant
- `GET /healthz` health check
- `GET /static/...` static assets
