package main

import (
	"context"
	"os"

	"github.com/Zokiio/context/internal/cli"
	"github.com/Zokiio/context/internal/taskcontext"
)

func main() {
	os.Exit(cli.Run(context.Background(), os.Args, os.Stdout, os.Stderr, taskcontext.Assemble))
}
