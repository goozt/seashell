package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/goozt/seashell/api"
	"github.com/goozt/seashell/cli"
	"github.com/goozt/seashell/config"
)

func main() {
	apiMode := flag.Bool("api", false, "Start the HTTP API server instead of the CLI")
	flag.Parse()

	if *apiMode {
		cfg := config.Load()
		if err := api.Start(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "API server error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	SeashellCli()
}

func SeashellCli() {
	defer os.Exit(0)
	cmd := cli.CommandLine{}
	cmd.Run()
}
