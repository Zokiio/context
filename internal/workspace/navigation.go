// Package workspace enumerates selected workspace membership without evaluating
// project records or work readiness.
package workspace

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/Zokiio/context/internal/discovery"
)

type Availability string

const (
	Available   Availability = "available"
	Unavailable Availability = "unavailable"
	Unknown     Availability = "unknown"
)

type Identity struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// Result describes membership only. Complete says that every authored member was
// enumerated; unavailable records do not make enumeration incomplete.
type Result struct {
	Complete    bool
	Workspace   Identity
	WorkStatus  string
	Members     []Member
	Diagnostics []Diagnostic
}

type Member struct {
	Key          string
	Title        string
	Directory    string
	Records      string
	AllowSources []string
	Availability Availability
}

type Diagnostic struct {
	Code      string `json:"code"`
	Severity  string `json:"severity"`
	MemberKey string `json:"memberKey"`
	Path      string `json:"path"`
	From      string `json:"from,omitempty"`
	Field     string `json:"field,omitempty"`
	Message   string `json:"message"`
}

// Navigate receives an already selected declaration. It checks access to each
// records directory without opening manifests, enumerating work, or checking
// checkout and allowed-source availability.
func Navigate(ctx context.Context, declaration discovery.WorkspaceDeclaration) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if err := validateDeclaration(declaration); err != nil {
		return Result{}, err
	}
	result := Result{Complete: true, Workspace: Identity{ID: declaration.ID, Title: declaration.Title},
		WorkStatus: "unevaluated", Members: []Member{}, Diagnostics: []Diagnostic{}}
	for _, entry := range declaration.Members {
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}
		member := Member{Key: entry.Key, Title: entry.Title, Records: entry.Records.Path,
			AllowSources: []string{}, Availability: Available}
		if entry.Directory != nil {
			member.Directory = entry.Directory.Path
		}
		for _, root := range entry.AllowSources {
			member.AllowSources = append(member.AllowSources, root.Path)
		}
		if err := directoryAccess(entry.Records.Path); err != nil {
			member.Availability = classifyAvailability(err)
			result.Diagnostics = append(result.Diagnostics, Diagnostic{
				Code: "member_records_" + string(member.Availability), Severity: "warning",
				MemberKey: entry.Key, Path: entry.Records.Path, From: entry.Records.Origin.Path, Field: entry.Records.Origin.Field,
				Message: fmt.Sprintf("records directory access is %s: %v; check this member's records declaration", member.Availability, err),
			})
		}
		result.Members = append(result.Members, member)
	}
	return result, nil
}

func validateDeclaration(declaration discovery.WorkspaceDeclaration) error {
	if strings.TrimSpace(declaration.ID) == "" || strings.TrimSpace(declaration.Title) == "" {
		return errors.New("workspace navigation requires a selected workspace with nonempty id and title")
	}
	keys := map[string]bool{}
	for _, member := range declaration.Members {
		if strings.TrimSpace(member.Key) == "" || keys[member.Key] {
			return fmt.Errorf("workspace %q member key %q must be nonempty and unique", declaration.ID, member.Key)
		}
		keys[member.Key] = true
		paths := append([]discovery.PathValue{member.Records}, member.AllowSources...)
		if member.Directory != nil {
			paths = append(paths, *member.Directory)
		}
		for _, path := range paths {
			if !filepath.IsAbs(path.Path) {
				return fmt.Errorf("workspace %q member %q field %s requires an absolute resolved path", declaration.ID, member.Key, path.Origin.Field)
			}
		}
	}
	return nil
}

func directoryAccess(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return &os.PathError{Op: "open directory", Path: path, Err: syscall.ENOTDIR}
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		return err
	}
	defer root.Close()
	directory, err := root.Open(".")
	if err != nil {
		return err
	}
	defer directory.Close()
	// Check directory read access without scanning an inventory or reading any
	// named member file. EOF is a successful check for an empty directory.
	if _, err := directory.ReadDir(1); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}

func classifyAvailability(err error) Availability {
	if errors.Is(err, os.ErrNotExist) || errors.Is(err, os.ErrPermission) ||
		errors.Is(err, syscall.ENOTDIR) || errors.Is(err, syscall.ELOOP) {
		return Unavailable
	}
	return Unknown
}
