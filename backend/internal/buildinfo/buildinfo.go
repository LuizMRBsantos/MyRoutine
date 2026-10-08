// Package buildinfo carries values stamped at build time.
package buildinfo

import "os"

// Version is set with -ldflags "-X github.com/myroutine/backend/internal/buildinfo.Version=<git sha>"
// by the Docker build. On Vercel there are no ldflags: the deploy workflow
// passes APP_VERSION, and Git-triggered builds expose VERCEL_GIT_COMMIT_SHA.
// Local `go run` builds report "dev".
var Version = "dev"

func init() {
	Version = resolve(Version, os.Getenv("APP_VERSION"), os.Getenv("VERCEL_GIT_COMMIT_SHA"))
}

// resolve keeps a stamped version; otherwise uses APP_VERSION, then the
// short Vercel commit SHA.
func resolve(stamped, appVersion, vercelSHA string) string {
	if stamped != "dev" {
		return stamped
	}
	if appVersion != "" {
		return shortSHA(appVersion)
	}
	if len(vercelSHA) >= 7 {
		return vercelSHA[:7]
	}
	return stamped
}

func shortSHA(v string) string {
	if len(v) > 7 {
		return v[:7]
	}
	return v
}
