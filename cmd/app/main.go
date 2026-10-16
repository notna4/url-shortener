// Entry point for your project.
//
// Every ATAD project is a web service in a container. The four routes below are
// the common contract and are checked automatically: keep them working exactly
// as specified and add your own routes alongside them.
//
// --version must also keep working. It prints one line of JSON and exits 0.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"syscall"
	"time"

	"atad-project/internal/storage"
	"atad-project/internal/version"
)

var (
	started     = time.Now()
	codePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,32}$`)
)

type createLinkRequest struct {
	URL  string `json:"url"`
	Code string `json:"code,omitempty"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	for _, a := range os.Args[1:] {
		if a == "--version" || a == "-v" {
			return version.WriteJSON(os.Stdout)
		}
	}

	log := slog.New(slog.NewTextHandler(os.Stderr, nil))

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "shortener.db"
	}

	store, err := storage.New(dbPath)
	if err != nil {
		return fmt.Errorf("initialize storage: %w", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			log.Error("failed closing storage", "error", err)
		}
	}()

	mux := http.NewServeMux()

	// ---- the common contract: four routes, identical in every project -------

	mux.HandleFunc("GET /version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = version.WriteJSON(w)
	})

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":         "ok",
			"uptime_seconds": int(time.Since(started).Seconds()),
		})
	})

	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<!doctype html><html lang="en"><meta charset="utf-8">
<title>%s</title><body>
<h1>%s</h1>
<p>Canceal Cristian-Anton, group DMWD</p>
<p>Project 1: URL Shortener with Analytics</p>
</body></html>`, version.Current().App, version.Current().App)
	})

	mux.HandleFunc("POST /reset", func(w http.ResponseWriter, r *http.Request) {
		if err := store.Reset(); err != nil {
			log.Error("failed to reset store", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to reset database state")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	// ---- your own routes go below -------------------------------------------

	mux.HandleFunc("POST /links", func(w http.ResponseWriter, r *http.Request) {
		var req createLinkRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
			return
		}

		if req.URL == "" {
			writeError(w, http.StatusBadRequest, "missing_url", "the url field is required")
			return
		}

		parsedURL, err := url.ParseRequestURI(req.URL)
		if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.Host == "" {
			writeError(w, http.StatusBadRequest, "invalid_url", "url must be a valid absolute http or https link")
			return
		}

		if req.Code != "" && !codePattern.MatchString(req.Code) {
			writeError(w, http.StatusBadRequest, "invalid_code", "code must be 3-32 alphanumeric characters, dashes, or underscores")
			return
		}

		link, err := store.CreateLink(req.URL, req.Code)
		if err != nil {
			if errors.Is(err, storage.ErrConflict) {
				writeError(w, http.StatusConflict, "conflict", "specified code already exists")
				return
			}
			log.Error("failed creating link", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to persist link")
			return
		}

		writeJSON(w, http.StatusCreated, link)
	})

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Shut down cleanly on Ctrl-C or on the signal Docker sends when stopping
	// the container, so in-flight requests are allowed to finish.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", srv.Addr, "commit", version.Current().Commit)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		log.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}

// writeJSON is the one place that sets the content type, so every response in
// the service is consistent about it.
func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// writeError produces the error shape the common contract requires. Use it for
// every failure rather than writing the JSON by hand at each call site.
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]string{"error": code, "message": message})
}
