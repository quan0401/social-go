// Command api is the HTTP server for social-go, a rebuild of ../GopherSocial
// from scratch. This is step 1: a chi router with one route, /v1/health.
// Compare ../GopherSocial/cmd/api/api.go to see where the same mount()
// ends up (posts, users, auth routes, CORS, rate limiter, graceful shutdown).
//
// Run it from social-go/ (every .go file in cmd/api is one package main,
// so run the folder, never a single file):
//
//	go run ./cmd/api
//	curl -i localhost:8080/v1/health     # 200 OK, body "ok"
//
// Or send the requests in cmd/api/client.http (VS Code REST Client).
//
// Reading order: main.go (wiring) → api.go (router + server) → health.go.
package main

import (
	"log"

	"github.com/quan0401/social-go/internal/env"
)

func main() {
	// Good: config is plain data built in main and handed down. Later the
	// course reads these values from env vars; only this block changes.
	cfg := config{
		addr: env.GetString("ADDR", ":8080"),
	}

	// Good: one *application holds every dependency (config now; DB store,
	// logger, mailer later). Handlers are methods on it, so they reach the
	// dependencies through `app.` instead of global variables.
	app := &application{
		config: cfg,
	}

	mux := app.mount()

	// Good: run returns the error and main decides to exit. A port already
	// in use prints "bind: address already in use" and exits 1, instead
	// of the silent exit 0 you get when ListenAndServe's error is ignored.
	log.Fatal(app.run(mux))
}
