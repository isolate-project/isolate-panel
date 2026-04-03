package version

// Build-time variables injected via -ldflags.
var (
	Version   = "dev"
	BuildDate = "unknown"
	GitCommit = "unknown"
)
