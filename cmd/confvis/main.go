// confvis generates confidence visualization badges and dashboards.
package main

import (
	"runtime/debug"

	"github.com/boinger/confvis/internal/cli"
)

var version = "dev"

func main() {
	cli.SetVersion(resolveVersion(version, defaultBuildInfo))
	cli.Execute()
}

// resolveVersion returns v unchanged unless it is "dev", in which case it
// attempts to read a version from Go module build info via buildInfoFn.
func resolveVersion(v string, buildInfoFn func() (string, bool)) string {
	if v == "dev" {
		if ver, ok := buildInfoFn(); ok && ver != "" {
			return ver
		}
	}
	return v
}

// defaultBuildInfo reads the version from runtime/debug build info.
func defaultBuildInfo() (string, bool) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "", false
	}
	return info.Main.Version, true
}
