package main

import (
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// application is the server's dependency container: everything a handler
// needs lives here. Methods use a pointer receiver (*application) so every
// handler shares the one app built in main, not a copy of it.
type application struct {
	config config
}

type config struct {
	addr string
}

// mount builds the router: middleware first, then routes.
// It returns http.Handler (the interface) rather than *chi.Mux, so run()
// doesn't care which router you picked. The commented-out ServeMux below
// plugs into run() unchanged.
func (app *application) mount() http.Handler {
	// Version 1: the stdlib router (Go 1.22+ method patterns).
	// mux := http.NewServeMux()

	// mux.HandleFunc("GET /v1/users", app.healthCheckHandler)

	// Version 2: chi. Same http.Handler, plus middleware (r.Use) and
	// route groups (r.Route) that ServeMux doesn't have.
	r := chi.NewRouter()

	// Middleware wraps every request, outermost first:
	//   RequestID → ClientIP → Logger → Recoverer → Timeout → your handler
	//
	// Good: this order matters and it's right. RequestID and ClientIP run
	// before Logger, so the log line can print both:
	//   [host/abc123-000001] "GET .../v1/health HTTP/1.1" from ::1 - 200 2B
	// Recoverer sits inside Logger, so a panic becomes a logged 500
	// instead of a dropped connection.
	//
	// NOTE: Logger writes to stdout, while log.Printf/log.Fatal write to
	// stderr. `go run ./cmd/api 2>err.log` splits them into two places.
	r.Use(middleware.RequestID)
	r.Use(middleware.ClientIPFromRemoteAddr) // client IP = TCP peer address. Right with no proxy in front; behind one use ClientIPFromXFF or ClientIPFromHeader
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Set a timeout value on the request context (ctx), that will signal
	// through ctx.Done() that the request has timed out and further
	// processing should be stopped.
	//
	// NOTE: Timeout only cancels ctx. It can't stop a handler that never
	// looks at r.Context().Done() (like healthCheckHandler). On the timeout
	// it answers 504 only if the handler returns early and wrote nothing.
	// ctx cancellation, see go-concurrency L4 §1.1.
	//
	// QUESTION: this is 60s, and run() sets WriteTimeout to 30s. What does
	// a client see when a handler takes 45s? Try it scaled down: Timeout(5s),
	// WriteTimeout 1s, and a handler that sleeps 3s. curl gets
	// "Empty reply from server" (exit 52), yet the Logger line says 200.
	// Hint: which of the two timeouts should fire first so the client gets
	// a real 504?
	r.Use(middleware.Timeout(60 * time.Second))

	// Good: a /v1 prefix from day one. A breaking change later can live
	// at /v2 while old clients keep working.
	//
	// NOTE: chi matches paths exactly. GET /v1/health/ (trailing slash) and
	// GET /health are both 404, POST is 405 with "Allow: GET".
	// HEAD is 405 too: r.Get doesn't answer HEAD, while ServeMux's
	// "GET /x" pattern does. That matters if a load balancer health-checks
	// with HEAD (see middleware.GetHead).
	r.Route("/v1", func(r chi.Router) {
		r.Get("/health", app.healthCheckHandler)
	})

	// Suggestion: these groups belong INSIDE the r.Route("/v1", ...) block
	// above, as nested r.Route("/posts", ...) calls, to get the /v1 prefix.
	// See ../GopherSocial/cmd/api/api.go, the r.Route("/posts", ...) block.

	// posts

	// users

	// auth

	return r
}

// run starts the server and blocks until it fails.
func (app *application) run(mux http.Handler) error {

	// Good: an explicit http.Server instead of http.ListenAndServe(addr, h).
	// The shortcut has no timeouts at all, so a client that sends its
	// headers one byte per minute (slowloris) holds a connection forever.
	//   ReadTimeout   whole request (headers + body) must arrive within 10s
	//   WriteTimeout  handler + response must finish within 30s after the
	//                 request headers are read, or the connection is cut
	//   IdleTimeout   keep-alive connection closes after 1 min of silence
	srv := &http.Server{
		Addr:         app.config.addr,
		Handler:      mux,
		WriteTimeout: time.Second * 30,
		ReadTimeout:  time.Second * 10,
		IdleTimeout:  time.Minute,
	}

	// NOTE: this prints before the port is bound. With :8080 already taken
	// you get "server has started at :8080" and then, on the next line,
	// "listen tcp :8080: bind: address already in use".
	log.Printf("server has started at %s", app.config.addr)

	// Suggestion: ListenAndServe never returns nil. After a Ctrl-C the
	// process just dies mid-request. Graceful shutdown (catch SIGINT/SIGTERM,
	// call srv.Shutdown(ctx)) is in ../GopherSocial/cmd/api/api.go, and is
	// "next" in your go-concurrency notes.
	return srv.ListenAndServe()
}
