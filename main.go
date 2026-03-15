package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/goozt/seashell/api"
	"github.com/goozt/seashell/cli"
	"github.com/goozt/seashell/config"
	"github.com/joho/godotenv"
)

func main() {
	// Handle plain subcommands before flag parsing so they don't interfere.
	if len(os.Args) >= 2 {
		switch os.Args[1] {
		case "genenv":
			out := ".env"
			if len(os.Args) >= 3 {
				out = os.Args[2]
			}
			cli.RunGenEnv(out)
			return
}
	}

	_ = godotenv.Load() // load .env if present; ignore error if missing

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
