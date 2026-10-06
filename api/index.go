// Package handler is the Vercel Function entry point. vercel.json rewrites
// every /api/* and /health request here, and the whole Go API serves it.
// The API itself lives in ./backend (see backend/server).
package handler

import (
	"encoding/json"
	"net/http"
	"runtime"

	"github.com/myroutine/backend/server"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	// TEMPORARY — feasibility spike only (docs/MVP_PLAN.md, Etapa 5). Shows
	// which path the function receives after the rewrite. Remove before launch.
	if r.URL.Path == "/api/__spike" {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"path":        r.URL.Path,
			"request_uri": r.RequestURI,
			"go":          runtime.Version(),
			"has_real_ip": r.Header.Get("X-Real-Ip") != "",
		})
		return
	}
	server.ServeHTTP(w, r)
}
