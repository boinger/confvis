// confvis generates confidence visualization badges and dashboards.
package main

import (
	"runtime/debug"

	"github.com/boinger/confvis/internal/cli"
)

var version = "dev"

func main() {
	if version == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
			version = info.Main.Version
		}
	}
	cli.SetVersion(version)
	cli.Execute()
}
