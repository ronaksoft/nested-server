package main

import (
	"git.ronaksoftware.com/nested/cli-ctl/cli/cmd"
	"os"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
