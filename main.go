package main

import (
	"os"

	"mini-coin-go/cmd"
)

func main() {
	defer os.Exit(0)

	cli := cmd.CLI{}
	cli.Run()
}
