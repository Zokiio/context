package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"

	"github.com/Zokiio/context/internal/discovery"
	"github.com/Zokiio/context/internal/workspace"
)

type workspaceReport struct {
	SchemaVersion int                     `json:"schemaVersion"`
	Kind          string                  `json:"kind"`
	Complete      bool                    `json:"complete"`
	Workspace     workspace.Identity      `json:"workspace"`
	WorkStatus    string                  `json:"workStatus"`
	Members       []workspaceMemberReport `json:"members"`
	Diagnostics   []workspace.Diagnostic  `json:"diagnostics"`
}

type workspaceMemberReport struct {
	Key           string                 `json:"key"`
	Title         string                 `json:"title,omitempty"`
	Directory     string                 `json:"directory,omitempty"`
	Records       string                 `json:"records"`
	Availability  workspace.Availability `json:"availability"`
	SelectionArgs []string               `json:"selectionArgs"`
}

func runWorkspaceOrientation(ctx context.Context, output io.Writer, declaration discovery.WorkspaceDeclaration, asJSON bool) error {
	result, err := workspace.Navigate(ctx, declaration)
	if err != nil {
		return err
	}
	report := workspaceNavigationReport(result)
	if asJSON {
		if err := json.NewEncoder(output).Encode(report); err != nil {
			return fmt.Errorf("write workspace navigation JSON: %w", err)
		}
		return nil
	}
	if err := renderWorkspace(output, report); err != nil {
		return fmt.Errorf("write workspace navigation text: %w", err)
	}
	return nil
}

func workspaceNavigationReport(result workspace.Result) workspaceReport {
	report := workspaceReport{SchemaVersion: 1, Kind: "workspace-navigation", Complete: result.Complete,
		Workspace: result.Workspace, WorkStatus: result.WorkStatus,
		Members: []workspaceMemberReport{}, Diagnostics: result.Diagnostics}
	for _, member := range result.Members {
		args := []string{"ctx", "orient", "--bundle", member.Records}
		for _, root := range member.AllowSources {
			args = append(args, "--allow-source", root)
		}
		report.Members = append(report.Members, workspaceMemberReport{
			Key: member.Key, Title: member.Title, Directory: member.Directory, Records: member.Records,
			Availability: member.Availability, SelectionArgs: args,
		})
	}
	return report
}

func renderWorkspace(output io.Writer, report workspaceReport) error {
	var text strings.Builder
	fmt.Fprintf(&text, "Workspace: %s [%s]\n", workspaceDisplay(report.Workspace.Title), workspaceDisplay(report.Workspace.ID))
	fmt.Fprintln(&text, "Work status was not evaluated.")
	if len(report.Members) == 0 {
		fmt.Fprintln(&text, "Members: none")
	} else {
		fmt.Fprintf(&text, "Members: %d\n", len(report.Members))
	}
	for _, member := range report.Members {
		fmt.Fprintf(&text, "  %s", workspaceDisplay(member.Key))
		if member.Title != "" {
			fmt.Fprintf(&text, ": %s", workspaceDisplay(member.Title))
		}
		fmt.Fprintf(&text, " [%s]\n    records: %s\n", member.Availability, workspaceDisplay(member.Records))
		if member.Directory != "" {
			fmt.Fprintf(&text, "    checkout: %s\n", workspaceDisplay(member.Directory))
		}
		fmt.Fprintf(&text, "    select: %s\n", posixCommand(member.SelectionArgs))
	}
	for _, diagnostic := range report.Diagnostics {
		fmt.Fprintf(&text, "  %s %s [%s]: %s\n", diagnostic.Severity, diagnostic.Code, workspaceDisplay(diagnostic.MemberKey), workspaceDisplay(diagnostic.Message))
		if diagnostic.From != "" {
			fmt.Fprintf(&text, "    source: %s field %s\n", workspaceDisplay(diagnostic.From), workspaceDisplay(diagnostic.Field))
		}
	}
	_, err := io.WriteString(output, text.String())
	return err
}

func workspaceDisplay(value string) string {
	if strings.ContainsFunc(value, unicode.IsControl) {
		return strconv.Quote(value)
	}
	return value
}

// POSIX single quotes preserve shell metacharacters and literal newlines. The
// quote sequence closes the literal, emits one apostrophe, and opens it again.
func posixCommand(args []string) string {
	quoted := make([]string, len(args))
	for i, arg := range args {
		safe := arg != "" && !strings.ContainsFunc(arg, func(char rune) bool {
			return !(char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' ||
				char >= '0' && char <= '9' || strings.ContainsRune("_./:-", char))
		})
		if safe {
			quoted[i] = arg
		} else {
			quoted[i] = "'" + strings.ReplaceAll(arg, "'", "'\"'\"'") + "'"
		}
	}
	return strings.Join(quoted, " ")
}
