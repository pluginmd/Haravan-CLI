// Package build carries values injected at link time via -ldflags.
package build

var (
	Version = "dev"
	Date    = "unknown"
	Commit  = "none"
)
