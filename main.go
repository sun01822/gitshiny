// GitShiny - Git contribution statistics for the terminal.
package main

import (
	"os"
	"runtime/debug"

	"github.com/sun01822/gitshiny/cmd"
)

// version is set at release time: -ldflags "-X main.version=v1.0.0".
var version = "dev"

func resolveVersion() string {
	if version != "dev" {
		return version
	}
	// Installed with `go install ...@vX.Y.Z`: use the module version.
	if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		return bi.Main.Version
	}
	return version
}

func main() {
	os.Exit(cmd.Execute(cmd.App{
		In: os.Stdin, Out: os.Stdout, Err: os.Stderr, Version: resolveVersion(),
	}, os.Args[1:]))
}
