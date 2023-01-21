package main

import (
	"os"

	"github.com/goozt/seashell/cli"
)

func main() {
	SeashellCli()
}

func SeashellCli() {
	defer os.Exit(0)
	cmd := cli.CommandLine{}
	cmd.Run()
}
