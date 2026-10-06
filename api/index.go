// Package handler is the Vercel Function entry point. vercel.json rewrites
// every /api/* and /health request here, and the whole Go API serves it.
// The API itself lives in ./backend (see backend/server).
package handler

import (
	"net/http"

	"github.com/myroutine/backend/server"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	server.ServeHTTP(w, r)
}
