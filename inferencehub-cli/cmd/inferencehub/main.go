package main

import (
	"fmt"
	"os"

	"github.com/Vinay-Venkatesh/inferencehub-cli/internal/cli"
)

func main() {
	err := cli.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
