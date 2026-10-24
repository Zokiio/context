package factorycli

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/Zokiio/context/internal/factory"
	cli "github.com/urfave/cli/v3"
)

func workCommand(out io.Writer) *cli.Command {
	draftFlags := []cli.Flag{
		&cli.StringFlag{Name: "title"},
		&cli.StringFlag{Name: "description", Usage: "Prose every agent reads before the requirements"},
		&cli.StringSliceFlag{Name: "criterion", Usage: "One requirement; repeat for each"},
		&cli.StringSliceFlag{Name: "blocked-by", Usage: "A work item ID that must be done first; repeat for each"},
		&cli.StringSliceFlag{Name: "spec", Usage: "A specification path relative to the work item; repeat for each"},
		&cli.StringSliceFlag{Name: "context", Usage: "A context path relative to the work item; repeat for each"},
	}
	itemPath := func(c *cli.Command) string {
		return "/v1/projects/" + esc(c.String("project")) + "/work-items/" + esc(c.String("work-item"))
	}
	command := &cli.Command{Name: "work", Usage: "Manage tasks, queue membership, and retained runs", DisableSliceFlagSeparator: true, Commands: []*cli.Command{
		{Name: "list", Usage: "List tasks, including drafts and archived tasks", Flags: projectFlags(), Action: func(ctx context.Context, c *cli.Command) error {
			return call(ctx, c, out, http.MethodGet, "/v1/projects/"+esc(c.String("project"))+"/work-items", nil)
		}},
		{Name: "reconcile", Usage: "Resume retained task saves after an interruption", Flags: projectFlags(), Action: func(ctx context.Context, c *cli.Command) error {
			return call(ctx, c, out, http.MethodPost, "/v1/projects/"+esc(c.String("project"))+"/work-items/reconcile", struct{}{})
		}},
		{Name: "show", Usage: "Show a work item's title, description, requirements and dependencies", Flags: append(projectFlags(), &cli.StringFlag{Name: "work-item", Required: true}), Action: func(ctx context.Context, c *cli.Command) error {
			return call(ctx, c, out, http.MethodGet, itemPath(c), nil)
		}},
		{Name: "add", Usage: "Write a new work item; --draft saves it without queueing", DisableSliceFlagSeparator: true, Flags: append(append(append(requestFlags(), projectFlags()...), draftFlags...), &cli.BoolFlag{Name: "draft", Usage: "Save without making the task eligible for automatic pickup"}), Action: func(ctx context.Context, c *cli.Command) error {
			request := factory.CreateWorkItemRequest{RequestID: reqID(c), ProjectID: factory.ProjectID(c.String("project")), Draft: factory.WorkItemDraft{Title: c.String("title"), Description: c.String("description"), Criteria: c.StringSlice("criterion"), BlockedBy: c.StringSlice("blocked-by"), Spec: c.StringSlice("spec"), Context: c.StringSlice("context")}}
			if c.Bool("draft") {
				queue := false
				request.Queue = &queue
			}
			return call(ctx, c, out, http.MethodPost, "/v1/projects/"+esc(c.String("project"))+"/work-items", request)
		}},
		{Name: "edit", Usage: "Rewrite a work item; a flag left out keeps its current value", DisableSliceFlagSeparator: true, Flags: append(append(append(requestFlags(), projectFlags()...), &cli.StringFlag{Name: "work-item", Required: true}), draftFlags...), Action: func(ctx context.Context, c *cli.Command) error {
			cl, err := client(c)
			if err != nil {
				return err
			}
			var record factory.WorkItemRecord
			if err := cl.DoJSON(ctx, http.MethodGet, itemPath(c), nil, &record); err != nil {
				return err
			}
			draft := record.Draft
			if c.IsSet("title") {
				draft.Title = c.String("title")
			}
			if c.IsSet("description") {
				draft.Description = c.String("description")
			}
			if c.IsSet("criterion") {
				draft.Criteria = c.StringSlice("criterion")
			}
			if c.IsSet("blocked-by") {
				draft.BlockedBy = c.StringSlice("blocked-by")
			}
			if c.IsSet("spec") {
				draft.Spec = workDocumentPaths(c.StringSlice("spec"))
			}
			if c.IsSet("context") {
				draft.Context = workDocumentPaths(c.StringSlice("context"))
			}
			request := factory.UpdateWorkItemRequest{RequestID: reqID(c), ProjectID: record.ProjectID, WorkItemID: record.WorkItemID, IfTicketSHA256: record.TicketSHA256, Draft: draft}
			return call(ctx, c, out, http.MethodPut, itemPath(c), request)
		}},
		{Name: "activity", Usage: "Read retained jobs, attempts, refinement and test-author history", Flags: append(projectFlags(), &cli.StringFlag{Name: "work-item", Required: true}), Action: func(ctx context.Context, c *cli.Command) error {
			return call(ctx, c, out, http.MethodGet, itemPath(c)+"/activity", nil)
		}},
	}}
	for _, action := range []string{"queue", "unqueue", "archive", "restore", "delete"} {
		command.Commands = append(command.Commands, &cli.Command{Name: action, Usage: action + " a task using its current revision", Flags: append(append(requestFlags(), projectFlags()...), &cli.StringFlag{Name: "work-item", Required: true}, &cli.StringFlag{Name: "if-ticket-sha256", Usage: "Expected revision; omitted reads the current revision first"}), Action: func(ctx context.Context, c *cli.Command) error {
			digest := c.String("if-ticket-sha256")
			if digest == "" {
				cl, err := client(c)
				if err != nil {
					return err
				}
				var record factory.WorkItemRecord
				if err := cl.DoJSON(ctx, http.MethodGet, itemPath(c), nil, &record); err != nil {
					return err
				}
				digest = record.TicketSHA256
			}
			return call(ctx, c, out, http.MethodPost, itemPath(c)+"/disposition", factory.SetWorkItemDispositionRequest{RequestID: reqID(c), ProjectID: factory.ProjectID(c.String("project")), WorkItemID: c.String("work-item"), IfTicketSHA256: digest, Action: action})
		}})
	}
	return command
}

func workDocumentPaths(values []string) []string {
	paths := []string{}
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			paths = append(paths, value)
		}
	}
	return paths
}
