package factorycli

import (
	"context"
	"fmt"
	"io"
	"strconv"

	"github.com/Zokiio/context/internal/factory"
	cli "github.com/urfave/cli/v3"
)

func disputesCommand(out io.Writer) *cli.Command {
	return &cli.Command{Name: "disputes", Usage: "Report a blocked contract, inspect the judge's decision, and route the next agent", Commands: []*cli.Command{
		{Name: "list", Usage: "List retained disputes for a work item", Flags: []cli.Flag{&cli.StringFlag{Name: "project", Required: true}, &cli.StringFlag{Name: "work-item", Required: true}}, Action: func(ctx context.Context, c *cli.Command) error {
			cl, err := client(c)
			if err != nil {
				return err
			}
			v, err := cl.ListDisputes(ctx, factory.ProjectID(c.String("project")), c.String("work-item"))
			if err != nil {
				return err
			}
			return encode(out, v)
		}},
		{Name: "show", Usage: "Read the report, decision, next owner, and available actions", Flags: []cli.Flag{&cli.StringFlag{Name: "id", Required: true}}, Action: func(ctx context.Context, c *cli.Command) error {
			cl, err := client(c)
			if err != nil {
				return err
			}
			v, err := cl.GetDispute(ctx, factory.ID(c.String("id")))
			if err != nil {
				return err
			}
			return encode(out, v)
		}},
		{Name: "report", Usage: "Send evidence of a requirement or test conflict to the judge", DisableSliceFlagSeparator: true,
			Flags: append(requestFlags(), &cli.StringFlag{Name: "job", Required: true}, &cli.Int64Flag{Name: "if-version", Required: true}, &cli.StringFlag{Name: "summary", Required: true}, &cli.StringFlag{Name: "observed-failure", Required: true}, &cli.StringSliceFlag{Name: "criterion", Usage: "Affected criterion number, repeatable"}, &cli.StringSliceFlag{Name: "test-path", Usage: "Affected repository test path, repeatable"}, &cli.StringSliceFlag{Name: "evidence", Required: true, Usage: "Evidence reference, repeatable"}, &cli.StringFlag{Name: "proposed-change"}),
			Action: func(ctx context.Context, c *cli.Command) error {
				input := factory.DisputeInput{Summary: c.String("summary"), ObservedFailure: c.String("observed-failure"), TestPaths: c.StringSlice("test-path"), EvidenceRefs: c.StringSlice("evidence"), ProposedChange: c.String("proposed-change")}
				for _, value := range c.StringSlice("criterion") {
					n, err := strconv.Atoi(value)
					if err != nil || n < 1 {
						return fmt.Errorf("criterion %q must be a positive number", value)
					}
					input.CriterionRefs = append(input.CriterionRefs, n)
				}
				if err := factory.ValidateDisputeInput(input); err != nil {
					return err
				}
				cl, err := client(c)
				if err != nil {
					return err
				}
				v, err := cl.ReportDispute(ctx, factory.ReportDisputeRequest{RequestID: reqID(c), JobID: factory.JobID(c.String("job")), IfVersion: c.Int64("if-version"), Dispute: input})
				if err != nil {
					return err
				}
				return encode(out, v)
			}},
		{Name: "resolve", Usage: "Apply an available dispute action against the decision version you inspected", Flags: append(requestFlags(), &cli.StringFlag{Name: "id", Required: true}, &cli.Int64Flag{Name: "if-version", Required: true}, &cli.StringFlag{Name: "action", Required: true, Usage: "An action returned by disputes show"}, &cli.StringFlag{Name: "rationale", Required: true}), Action: func(ctx context.Context, c *cli.Command) error {
			cl, err := client(c)
			if err != nil {
				return err
			}
			v, err := cl.ResolveDispute(ctx, factory.ResolveDisputeRequest{RequestID: reqID(c), DisputeID: factory.ID(c.String("id")), IfVersion: c.Int64("if-version"), Action: c.String("action"), Rationale: c.String("rationale")})
			if err != nil {
				return err
			}
			return encode(out, v)
		}},
	}}
}
