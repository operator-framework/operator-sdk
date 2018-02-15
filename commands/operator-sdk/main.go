package main

import (
	"fmt"
	"os"

	"github.com/coreos/operator-sdk/commands/operator-sdk/cmd"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(-1)
	}
}
