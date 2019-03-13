package main

import (
  "os"
  "git.ronaksoftware.com/nested/cli-ctl/cli/cmd"
)

func main() {
  if err := cmd.RootCmd.Execute(); err != nil {
    os.Exit(1)
  }
}
