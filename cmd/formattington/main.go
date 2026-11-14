// Command formattington is a cross-platform, multi-language formatting and
// code-quality tool orchestrator.
//
// It provides a unified command-line interface for invoking existing tools
// (40 registered tools — formatters, linters, and other quality tools)
// across multidisciplinary projects.
package main

import (
	"os"

	"github.com/fordaj/formattington/internal/formattington"
)

func main() {
	os.Exit(formattington.Main(os.Args[1:]))
}
