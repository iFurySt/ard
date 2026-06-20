package main

import (
	"fmt"
	"os"

	"github.com/ifuryst/ard/internal/cli"
)

func main() {
	if err := cli.NewCLICommand().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
