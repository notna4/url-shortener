# Architecture

## 1. The Shape

The project is set up as a straightforward Go microservice.

- **How a request flows**: Incoming requests hit the server's built-in router. From there, they split into two paths: standard management routes (like `/health`, `/version`, `/`, and `POST /reset`) which handle basic service checks and maintenance, and the core application routes (`POST /links` and `GET /{code}`) which handle creating links and redirecting visitors
- **The redirect hot path**: When someone requests a short code via `GET /{code}`, the server immediately looks it up in the database, atomically updates the click counter, and fires back an HTTP `302 Found` redirect with the destination URL in the header
- **Storage layer**: All handlers talk to a database package that wraps around SQLite

## 2. The Data

- **What is stored**: A single `links` table holding the auto-incrementing ID, the unique short code, the target URL, total visit count, last visit timestamp, and creation timestamp
- **Where it lives**: It’s saved in an embedded SQLite database file on disk (defaulting to `/data/shortener.db` inside the container volume)
- **Surviving restarts**: Yes, it fully survives restarts. Because the database file is stored inside a Docker volume (`-v $(pwd)/data:/data`), all your links and click counts remain intact even if the container stops or reboots

## 3. Three Decisions

1. **Using SQLite instead of an external database like PostgreSQL**
   - _What I chose_: An embedded SQLite database
   - _What I rejected_: Running a separate Postgres container
   - _Why_: It keeps everything inside a single Go binary and a simple Docker container with zero extra network overhead or setup complexity
2. **Counting visits synchronously during the redirect**
   - _What I chose_: Incrementing the visit counter right inside the request path (`visits = visits + 1`) before sending the 302 redirect
   - _What I rejected_: A background worker queue or asynchronous goroutine to log clicks later
   - _Why_: The project rules state that a visit only counts when the redirect is actually served. Doing it synchronously ensures we don't lose data if the app crashes right after redirecting
3. **Enabling WAL (Write-Ahead Logging) mode**
   - _What I chose_: Turning on SQLite's WAL mode (`PRAGMA journal_mode=WAL;`)
   - _What I rejected_: Leaving SQLite on its default journal mode
   - _Why_: WAL mode lets multiple readers (handling fast redirection lookups) run at the same time without getting blocked by a write updating the click counter.

## 4. What Is Not Built Yet

- **Statistics endpoint (`GET /links/{code}/stats`)**: Reading individual link analytics (returning total visits and last visit time, which stays null until someone actually visits it)
- **Link deletion (`DELETE /links/{code}`)**: Removing specific links
- **Link listing (`GET /links`)**: Viewing a paginated list of all created links
- **Link expiration**: Automatic time-to-live (TTL) cleanups
