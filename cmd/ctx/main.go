package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/Zokiio/context/internal/cli"
	"github.com/Zokiio/context/internal/orientation"
	"github.com/Zokiio/context/internal/resumption"
	"github.com/Zokiio/context/internal/taskcontext"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := cli.Run(ctx, os.Args, os.Stdout, os.Stderr, cli.Operations{
		Assemble: taskcontext.Assemble,
		Orient:   orientation.Orient,
		Resume:   resumption.Resume,
	})
	stop()
	os.Exit(code)
}
