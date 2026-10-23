package factorycli

import (
	"context"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/Zokiio/context/internal/factory"
	"github.com/Zokiio/context/internal/factoryworker"
	cli "github.com/urfave/cli/v3"
)

func expectationFlags() []cli.Flag {
	return []cli.Flag{&cli.StringFlag{Name: "expectation-command-file", Usage: "JSON argv of the command that writes holdout tests"}}
}

func expectationAuthorCommand(stdout, stderr io.Writer) *cli.Command {
	return &cli.Command{Name: "expectation-author", Usage: "Write holdout tests for the expectation input on stdin using Claude Code", Hidden: true, DisableSliceFlagSeparator: true,
		Flags: []cli.Flag{&cli.StringFlag{Name: "binary"}, &cli.StringFlag{Name: "model"}, &cli.Float64Flag{Name: "max-budget-usd"}, &cli.StringSliceFlag{Name: "allow-tool"}},
		Action: func(ctx context.Context, c *cli.Command) error {
			return factoryworker.ExpectationAuthor(ctx, os.Stdin, stdout, stderr, factoryworker.ExpectationAuthorOptions{Binary: c.String("binary"), Model: c.String("model"), MaxBudgetUSD: c.Float64("max-budget-usd"), AllowedTools: c.StringSlice("allow-tool")})
		}}
}

func expectationsCommand(out io.Writer) *cli.Command {
	idFlag := &cli.StringFlag{Name: "id", Required: true, Usage: "Expectation ID"}
	decide := func(name string) *cli.Command {
		return &cli.Command{Name: name, Usage: "Record a decision on proposed holdout expectations", Flags: append(requestFlags(), idFlag), Action: func(ctx context.Context, c *cli.Command) error {
			request := factory.DecideExpectationsRequest{RequestID: reqID(c), ExpectationID: factory.ID(c.String("id"))}
			return call(ctx, c, out, http.MethodPost, "/v1/expectations/"+esc(c.String("id"))+"/"+name, request)
		}}
	}
	return &cli.Command{Name: "expectations", Usage: "Write holdout tests for a work item before implementation", Commands: []*cli.Command{
		{Name: "start", Usage: "Ask the expectation author for tests under the project's protected paths",
			Flags: append(requestFlags(), &cli.StringFlag{Name: "project", Required: true}, &cli.StringFlag{Name: "work-item", Required: true}, &cli.StringFlag{Name: "base", Usage: "Base revision; defaults to HEAD"}, &cli.BoolFlag{Name: "wait"}),
			Action: func(ctx context.Context, c *cli.Command) error {
				cl, err := client(c)
				if err != nil {
					return err
				}
				request := factory.StartExpectationsRequest{RequestID: reqID(c), ProjectID: factory.ProjectID(c.String("project")), WorkItemID: c.String("work-item"), BaseRevision: c.String("base")}
				var expectation factory.Expectation
				if err := cl.DoJSON(ctx, http.MethodPost, "/v1/projects/"+esc(c.String("project"))+"/expectations", request, &expectation); err != nil {
					return err
				}
				for c.Bool("wait") && expectation.Status == factory.ExpectationRunning {
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-time.After(2 * time.Second):
					}
					if err := cl.DoJSON(ctx, http.MethodGet, "/v1/expectations/"+esc(string(expectation.ID)), nil, &expectation); err != nil {
						return err
					}
				}
				return encode(out, expectation)
			}},
		{Name: "show", Usage: "Inspect proposed expectations and their patch", Flags: []cli.Flag{idFlag}, Action: func(ctx context.Context, c *cli.Command) error {
			return call(ctx, c, out, http.MethodGet, "/v1/expectations/"+esc(c.String("id")), nil)
		}},
		{Name: "list", Usage: "List a project's expectations", Flags: []cli.Flag{&cli.StringFlag{Name: "project", Required: true}, &cli.StringFlag{Name: "work-item"}}, Action: func(ctx context.Context, c *cli.Command) error {
			path := "/v1/projects/" + esc(c.String("project")) + "/expectations"
			if c.String("work-item") != "" {
				path += "?workItemId=" + esc(c.String("work-item"))
			}
			return call(ctx, c, out, http.MethodGet, path, nil)
		}},
		decide("accept"), decide("reject"),
	}}
}
