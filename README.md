# Project 1 - URL Shortener with Analytics

A lightweight, persistent URL shortening microservice and redirection engine built in Go. It enables users, marketers, and developers to convert lengthy URLs into compact identifiers, navigate through them via fast HTTP redirects, and track visit analytics in an embedded SQLite database.

- **Student:** Canceal Cristian-Anton, group DMWD
- **Project:** Project 1 - URL Shortener with Analytics
- **Language:** Go

## What it does

This application is a REST API that lets you create and manage short links. You can send a long web link to the service, and it will give you back a short code (or you can pick your own custom name). When someone visits that short link, the server immediately redirects them to the destination site and adds 1 to the visit count.

All links and click counts are stored locally in an SQLite database file, so your links stay saved even if the app restarts. The service also meets all the course requirements, providing health check endpoints, version info, and a clean reset command.

The project is currently at Milestone 1: custom and auto-generated link creation, the redirection hot path, contract endpoints, and multi-architecture Docker deployment are fully operational. The visit statistics endpoint (`GET /links/{code}/stats`) and link expiration support will be implemented in Milestone 2.

## How to run it

Pull and run the pre-built multi-arch image from GitHub Container Registry:

```bash
docker pull ghcr.io/notna4/url-shortener:latest
docker run --rm -p 8080:8080 -v $(pwd)/data:/data ghcr.io/notna4/url-shortener:latest
```

To run the version check flag inside the container:

```bash
docker run --rm ghcr.io/notna4/url-shortener:latest --version
```

To build and run locally without Docker:

```bash
go run ./cmd/app
go run ./cmd/app --version
```

By default, running locally creates a shortener.db SQLite database in the current directory. You can specify a different location via the DB_PATH environment variable:

```bash
DB_PATH=/tmp/shortener.db go run ./cmd/app
```

## What you should see

1. Health check verification

```bash
curl -i http://localhost:8080/health
```

Output:

```http
HTTP/1.1 200 OK
Content-Type: application/json
Date: Sun, 01 Nov 2026 17:00:00 GMT
Content-Length: 35

{"status":"ok","uptime_seconds":12}
```

2. Creating a shortened URL

```bash
curl -i -X POST http://localhost:8080/links \
  -H "Content-Type: application/json" \
  -d '{"url":"https://google.com"}'
```

Output:

```http
HTTP/1.1 201 Created
Content-Type: application/json
Date: Sun, 01 Nov 2026 17:00:05 GMT
Content-Length: 115

{"code":"CkcpapY","url":"https://google.com","visits":0,"last_visit_at":null,"created_at":"2026-11-01T17:00:05Z"}
```

3. Redirection hot path

```bash
curl -i http://localhost:8080/CkcpapY
```

Output:

```http
HTTP/1.1 302 Found
Location: https://google.com
Date: Sun, 01 Nov 2026 17:00:10 GMT
Content-Length: 42
Content-Type: text/html; charset=utf-8

<a href="https://google.com">Found</a>.
```

4. Binary version contract flag
   Run inside the container:

```bash
docker run --rm ghcr.io/notna4/url-shortener:latest --version
```

Output:

```json
{
  "app": "atad-project",
  "version": "0.0.0-dev",
  "commit": "116af0d29cf24da9d77087e0d9e65af4f19f308b",
  "built_at": "2026-11-01T00:21:33+02:00"
}
```

5. Storage reset contract endpoint

```bash
curl -i -X POST http://localhost:8080/reset
```

Output:

```http
HTTP/1.1 204 No Content
```

6. Student landing page contract endpoint

```bash
curl -i http://localhost:8080/
```

```http
HTTP/1.1 200 OK
Content-Type: text/html; charset=utf-8
Date: Sun, 01 Nov 2026 18:02:39 GMT
Content-Length: 213

<!doctype html><html lang="en"><meta charset="utf-8">
<title>atad-project</title><body>
<h1>atad-project</h1>
<p>Canceal Cristian-Anton, group DMWD</p>
<p>Project 1: URL Shortener with Analytics</p>
</body></html>%
```
