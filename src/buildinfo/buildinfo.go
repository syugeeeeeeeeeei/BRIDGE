package buildinfo

import "runtime"

var (
	Version   = "0.15.3"
	Commit    = "unknown"
	BuildTime = "unknown"
	Dirty     = "unknown"
)

func GoVersion() string { return runtime.Version() }
