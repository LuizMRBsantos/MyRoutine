// Package buildinfo carries values stamped at build time.
package buildinfo

import "os"

// Version is set with -ldflags "-X github.com/myroutine/backend/internal/buildinfo.Version=<git sha>"
// by the Docker build. Vercel builds without ldflags but exposes the commit as
// VERCEL_GIT_COMMIT_SHA, read at startup. Local `go run` builds report "dev".
var Version = "dev"

func init() {
	Version = resolve(Version, os.Getenv("VERCEL_GIT_COMMIT_SHA"))
}

// resolve keeps a stamped version; otherwise uses the short Vercel commit SHA.
func resolve(stamped, vercelSHA string) string {
	if stamped != "dev" || len(vercelSHA) < 7 {
		return stamped
	}
	return vercelSHA[:7]
}
