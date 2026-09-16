package main

import (
	"context"
	"os"

	"github.com/Zokiio/context/internal/cli"
	"github.com/Zokiio/context/internal/orientation"
	"github.com/Zokiio/context/internal/taskcontext"
)

func main() {
	os.Exit(cli.Run(context.Background(), os.Args, os.Stdout, os.Stderr, cli.Operations{
		Assemble: taskcontext.Assemble,
		Orient:   orientation.Orient,
	}))
}
