// Package buildinfo carries values stamped at build time.
package buildinfo

// Version is set with -ldflags "-X github.com/myroutine/backend/internal/buildinfo.Version=<git sha>"
// by the Docker build; local `go run` builds report "dev".
var Version = "dev"
