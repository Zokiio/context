package factorycli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/Zokiio/context/internal/factory"
	"github.com/Zokiio/context/internal/factoryworker"
	cli "github.com/urfave/cli/v3"
)

func refinerFlags() []cli.Flag {
	return []cli.Flag{&cli.StringFlag{Name: "refiner-command-file", Usage: "JSON argv of the command that proposes refined acceptance criteria"}}
}

func refinerCommandFile(c *cli.Command) ([]string, error) {
	path := c.String("refiner-command-file")
	if path == "" {
		return nil, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var argv []string
	if err := json.Unmarshal(raw, &argv); err != nil || len(argv) == 0 || argv[0] == "" {
		return nil, errors.New("refiner-command-file must contain a nonempty JSON argv array")
	}
	return argv, nil
}

func refinerCommand(stdout, stderr io.Writer) *cli.Command {
	return &cli.Command{Name: "refiner", Usage: "Propose refined criteria for the refinement input on stdin using read-only Claude Code", Hidden: true,
		Flags: []cli.Flag{&cli.StringFlag{Name: "binary"}, &cli.StringFlag{Name: "model"}, &cli.Float64Flag{Name: "max-budget-usd"}},
		Action: func(ctx context.Context, c *cli.Command) error {
			return factoryworker.Refiner(ctx, os.Stdin, stdout, stderr, factoryworker.RefinerOptions{Binary: c.String("binary"), Model: c.String("model"), MaxBudgetUSD: c.Float64("max-budget-usd")})
		}}
}

func refineCommand(out io.Writer) *cli.Command {
	idFlag := &cli.StringFlag{Name: "id", Required: true, Usage: "Refinement ID"}
	return &cli.Command{Name: "refine", Usage: "Refine a work item's acceptance criteria before admission", Commands: []*cli.Command{
		{Name: "start", Usage: "Ask the refiner for rewritten criteria and up to three blocking questions", DisableSliceFlagSeparator: true,
			Flags: append(requestFlags(), &cli.StringFlag{Name: "project", Required: true}, &cli.StringFlag{Name: "work-item", Required: true}, &cli.StringFlag{Name: "previous", Usage: "Refine again from this proposal"}, &cli.StringSliceFlag{Name: "answer", Usage: "Answer to the previous proposal's questions, in order"}, &cli.BoolFlag{Name: "wait", Usage: "Wait for the proposal"}),
			Action: func(ctx context.Context, c *cli.Command) error {
				cl, err := client(c)
				if err != nil {
					return err
				}
				request := factory.StartRefinementRequest{RequestID: reqID(c), ProjectID: factory.ProjectID(c.String("project")), WorkItemID: c.String("work-item"), PreviousID: factory.ID(c.String("previous"))}
				if len(c.StringSlice("answer")) > 0 {
					if request.PreviousID == "" {
						return errors.New("--answer requires --previous")
					}
					var previous factory.Refinement
					if err := cl.DoJSON(ctx, http.MethodGet, "/v1/refinements/"+esc(c.String("previous")), nil, &previous); err != nil {
						return err
					}
					request.Answers, err = pairAnswers(previous, c.StringSlice("answer"))
					if err != nil {
						return err
					}
				}
				var refinement factory.Refinement
				if err := cl.DoJSON(ctx, http.MethodPost, "/v1/projects/"+esc(c.String("project"))+"/refinements", request, &refinement); err != nil {
					return err
				}
				for c.Bool("wait") && refinement.Status == factory.RefinementRunning {
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-time.After(2 * time.Second):
					}
					if err := cl.DoJSON(ctx, http.MethodGet, "/v1/refinements/"+esc(string(refinement.ID)), nil, &refinement); err != nil {
						return err
					}
				}
				return encode(out, refinement)
			}},
		{Name: "show", Usage: "Inspect a refinement", Flags: []cli.Flag{idFlag}, Action: func(ctx context.Context, c *cli.Command) error {
			return call(ctx, c, out, http.MethodGet, "/v1/refinements/"+esc(c.String("id")), nil)
		}},
		{Name: "list", Usage: "List a project's refinements", Flags: []cli.Flag{&cli.StringFlag{Name: "project", Required: true}, &cli.StringFlag{Name: "work-item"}}, Action: func(ctx context.Context, c *cli.Command) error {
			path := "/v1/projects/" + esc(c.String("project")) + "/refinements"
			if c.String("work-item") != "" {
				path += "?workItemId=" + esc(c.String("work-item"))
			}
			return call(ctx, c, out, http.MethodGet, path, nil)
		}},
		{Name: "accept", Usage: "Accept the proposal, optionally edited, and rewrite the ticket", DisableSliceFlagSeparator: true,
			Flags: append(requestFlags(), &cli.StringFlag{Name: "id", Required: true}, &cli.StringSliceFlag{Name: "criterion", Usage: "Replacement criterion; defaults to the proposal's"}, &cli.StringSliceFlag{Name: "answer", Usage: "Answer to the proposal's questions, in order"}),
			Action: func(ctx context.Context, c *cli.Command) error {
				cl, err := client(c)
				if err != nil {
					return err
				}
				var refinement factory.Refinement
				if err := cl.DoJSON(ctx, http.MethodGet, "/v1/refinements/"+esc(c.String("id")), nil, &refinement); err != nil {
					return err
				}
				if refinement.Proposal == nil {
					return fmt.Errorf("refinement %s has no proposal (status %s)", refinement.ID, refinement.Status)
				}
				criteria := c.StringSlice("criterion")
				if len(criteria) == 0 {
					criteria = refinement.Proposal.Criteria
				}
				answers, err := pairAnswers(refinement, c.StringSlice("answer"))
				if err != nil {
					return err
				}
				request := factory.AcceptRefinementRequest{RequestID: reqID(c), RefinementID: refinement.ID, Criteria: criteria, Answers: answers}
				return call(ctx, c, out, http.MethodPost, "/v1/refinements/"+esc(string(refinement.ID))+"/accept", request)
			}},
		{Name: "reject", Usage: "Reject the proposal and leave the ticket unchanged", Flags: append(requestFlags(), idFlag), Action: func(ctx context.Context, c *cli.Command) error {
			request := factory.RejectRefinementRequest{RequestID: reqID(c), RefinementID: factory.ID(c.String("id"))}
			return call(ctx, c, out, http.MethodPost, "/v1/refinements/"+esc(c.String("id"))+"/reject", request)
		}},
	}}
}

// pairAnswers matches answers, in order, to a proposal's questions.
func pairAnswers(refinement factory.Refinement, answers []string) ([]factory.RefinementAnswer, error) {
	if len(answers) == 0 {
		return nil, nil
	}
	if refinement.Proposal == nil || len(answers) > len(refinement.Proposal.Questions) {
		return nil, fmt.Errorf("refinement %s asked fewer questions than the answers given", refinement.ID)
	}
	paired := make([]factory.RefinementAnswer, 0, len(answers))
	for i, answer := range answers {
		paired = append(paired, factory.RefinementAnswer{Question: refinement.Proposal.Questions[i].Question, Answer: answer})
	}
	return paired, nil
}
