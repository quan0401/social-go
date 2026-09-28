package main

import "net/http"

// healthCheckHandler answers GET /v1/health so a load balancer, Docker or
// k8s can ask "is the process up?" without touching real data.
//
// Nothing sets the status or Content-Type, so net/http fills them in on the
// first Write: status 200, and Content-Type sniffed from the bytes
// ("text/plain; charset=utf-8" for "ok").
//
// Suggestion: GopherSocial returns JSON here ({"status":"ok","env":...,
// "version":...}) through a writeJSON helper. Worth it once there's a
// second JSON endpoint, see ../GopherSocial/cmd/api/health.go and json.go.
func (app *application) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	// NOTE: golangci-lint (errcheck) flags this ignored error. Once Write
	// fails the client is already gone, so there's nothing useful to send.
	// Ignoring it is normal; `_, _ = w.Write(...)` makes that explicit.
	w.Write([]byte("ok"))
}
