// confvis generates confidence visualization badges and dashboards.
package main

import "github.com/boinger/confvis/internal/cli"

var version = "dev"

func main() {
	cli.SetVersion(version)
	cli.Execute()
}
