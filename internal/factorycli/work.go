package factorycli

import (
	"context"
	"io"
	"net/http"

	"github.com/Zokiio/context/internal/factory"
	cli "github.com/urfave/cli/v3"
)

func workCommand(out io.Writer) *cli.Command {
	draftFlags := []cli.Flag{
		&cli.StringFlag{Name: "title"},
		&cli.StringFlag{Name: "description", Usage: "Prose every agent reads before the requirements"},
		&cli.StringSliceFlag{Name: "criterion", Usage: "One requirement; repeat for each"},
		&cli.StringSliceFlag{Name: "blocked-by", Usage: "A work item ID that must be done first; repeat for each"},
	}
	itemPath := func(c *cli.Command) string {
		return "/v1/projects/" + esc(c.String("project")) + "/work-items/" + esc(c.String("work-item"))
	}
	return &cli.Command{Name: "work", Usage: "Add and edit the project's work items", DisableSliceFlagSeparator: true, Commands: []*cli.Command{
		{Name: "show", Usage: "Show a work item's title, description, requirements and dependencies", Flags: append(projectFlags(), &cli.StringFlag{Name: "work-item", Required: true}), Action: func(ctx context.Context, c *cli.Command) error {
			return call(ctx, c, out, http.MethodGet, itemPath(c), nil)
		}},
		{Name: "add", Usage: "Write a new work item and list it in the project", DisableSliceFlagSeparator: true, Flags: append(append(requestFlags(), projectFlags()...), draftFlags...), Action: func(ctx context.Context, c *cli.Command) error {
			request := factory.CreateWorkItemRequest{RequestID: reqID(c), ProjectID: factory.ProjectID(c.String("project")), Draft: factory.WorkItemDraft{Title: c.String("title"), Description: c.String("description"), Criteria: c.StringSlice("criterion"), BlockedBy: c.StringSlice("blocked-by")}}
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
			request := factory.UpdateWorkItemRequest{RequestID: reqID(c), ProjectID: record.ProjectID, WorkItemID: record.WorkItemID, IfTicketSHA256: record.TicketSHA256, Draft: draft}
			return call(ctx, c, out, http.MethodPut, itemPath(c), request)
		}},
	}}
}
