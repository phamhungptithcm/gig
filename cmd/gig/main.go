package main

import (
	"context"
	"fmt"
	"os"

	"gig/internal/cli"
)

func main() {
	app, err := cli.NewApp(os.Stdout, os.Stderr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize gig: %v\n", err)
		os.Exit(1)
	}

	os.Exit(app.Run(context.Background(), os.Args[1:]))
}
