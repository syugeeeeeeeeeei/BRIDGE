package buildinfo

import "runtime"

var (
	Version   = "0.16.0"
	Commit    = "unknown"
	BuildTime = "unknown"
	Dirty     = "unknown"
)

func GoVersion() string { return runtime.Version() }
