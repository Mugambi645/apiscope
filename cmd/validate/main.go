// Command validate checks every catalog manifest. CI runs it on each pull request.
package main

import (
	"fmt"
	"os"

	"apiscope/internal/catalog"
)

func main() {
	dir := "catalog"
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	apis, err := catalog.LoadDir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "INVALID: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("OK: %d manifests valid\n", len(apis))
}
