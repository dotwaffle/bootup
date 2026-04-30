package main

import (
	"fmt"
	"os"

	"github.com/dotwaffle/bootup/internal/catalog"
)

func main() {
	generated, err := catalog.GenerateDefault()
	if err != nil {
		fmt.Fprintf(os.Stderr, "generate default catalog: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile("default.json", generated, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write default catalog: %v\n", err)
		os.Exit(1)
	}
}
