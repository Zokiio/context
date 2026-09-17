package discovery

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/Zokiio/context/internal/recordread"
)

type Kind string

const (
	Any       Kind = ""
	Project   Kind = "project"
	Workspace Kind = "workspace"
)

// Request supplies the invocation environment explicitly. Selector is either a
// directory, relative to Cwd, or a personal alias beginning with @. An empty
// selector starts at Cwd. Kind filters candidates, with Any preferring a project
// over a workspace at the same directory. Cwd may be empty for an explicit alias
// when the caller's working directory is unavailable. Home must be absolute.
type Request struct {
	Cwd      string
	Home     string
	Kind     Kind
	Selector string
}

type Scope struct {
	Kind             Kind
	StartDirectory   string
	BindingDirectory string
	Project          *ResolvedProject
	Workspace        *WorkspaceDeclaration
	Origins          []Origin
}

type ResolvedProject struct {
	Records      string
	AllowSources []string
}

// Resolve reads configuration and selected targets without changing directories,
// printing, prompting, or writing files.
func Resolve(ctx context.Context, request Request) (Scope, error) {
	return resolve(ctx, request, nil)
}

// A replacement is the one proposed configuration document setup wants to check.
// It takes the place of that encountered filename, even before the file exists.
func resolve(ctx context.Context, request Request, replacement *Config) (scope Scope, err error) {
	entered := request.Selector
	if entered == "" {
		entered = request.Cwd
	}
	defer func() {
		if err != nil {
			err = fmt.Errorf("discover scope from %q (cwd %q): %w", entered, request.Cwd, err)
		}
	}()
	if err := ctx.Err(); err != nil {
		return Scope{}, err
	}
	if request.Kind != Any && request.Kind != Project && request.Kind != Workspace {
		return Scope{}, fmt.Errorf("unsupported scope kind %q; choose project or workspace", request.Kind)
	}
	alias := strings.HasPrefix(request.Selector, "@")
	if !filepath.IsAbs(request.Home) {
		return Scope{}, errors.New("home must be an absolute directory; supply the invocation environment")
	}
	if !filepath.IsAbs(request.Cwd) && !(alias && request.Kind != Any && request.Cwd == "") {
		return Scope{}, errors.New("cwd must be an absolute directory, or omitted for explicit alias selection; supply the invocation environment")
	}
	r := resolver{ctx: ctx, request: request, replacement: replacement,
		personalPath: appendPath(request.Home, ".context/config.md")}
	if alias {
		// No checkout or cwd availability is needed to select a saved entry.
		if request.Cwd != "" {
			r.start, _ = CanonicalPath(request.Cwd)
		}
	} else {
		start := request.Cwd
		if request.Selector != "" {
			start = commandPath(request.Cwd, request.Selector)
		}
		r.start, err = CanonicalPath(start)
		if err != nil {
			return Scope{}, fmt.Errorf("start directory %q resolves to %q: %w; choose an accessible directory", start, r.start, err)
		}
		if err := checkDirectory(start); err != nil {
			return Scope{}, fmt.Errorf("start directory %q resolves to %q: %w; choose an accessible directory", start, r.start, err)
		}
	}
	r.personal, err = r.loadConfig(r.personalPath, PersonalConfig)
	if err != nil {
		return Scope{}, err
	}
	if alias {
		return r.alias()
	}
	if err := r.checkUnresolvedBindings(); err != nil {
		return Scope{}, err
	}
	for directory := r.start; ; directory = filepath.Dir(directory) {
		if err := ctx.Err(); err != nil {
			return Scope{}, err
		}
		candidates, err := r.atDirectory(directory)
		if err != nil {
			return Scope{}, err
		}
		if len(candidates) != 0 {
			return r.selectCandidates(directory, candidates, true)
		}
		if parent := filepath.Dir(directory); parent == directory {
			return Scope{}, fmt.Errorf("no %sscope found after searching physical ancestors of %q through filesystem root %q; run ctx setup or use --bundle PATH for direct records access", kindPrefix(request.Kind), r.start, directory)
		}
	}
}

type resolver struct {
	ctx          context.Context
	request      Request
	replacement  *Config
	personalPath string
	personal     *Config
	start        string
}

type candidate struct {
	origin    Origin
	directory *PathValue
	project   *ProjectBinding
	workspace *WorkspaceDeclaration
	marker    bool
	err       error
}

func kindPrefix(kind Kind) string {
	if kind == Any {
		return ""
	}
	return string(kind) + " "
}

func appendPath(directory, suffix string) string {
	return directory + string(filepath.Separator) + filepath.FromSlash(suffix)
}

func commandPath(cwd, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return appendPath(cwd, path)
}

func (r *resolver) loadConfig(path string, profile ConfigProfile) (*Config, error) {
	if r.replacement != nil && sameConfigLocation(r.replacement.Path, path) {
		if r.replacement.Profile != profile {
			return nil, fmt.Errorf("configuration %s requires the %s profile; repair the proposed configuration", path, profile)
		}
		return r.replacement, nil
	}
	present, err := configPresent(path)
	if err != nil {
		return nil, fmt.Errorf("inspect configuration %s: %w; repair its file or directory", path, err)
	}
	if !present {
		return nil, nil
	}
	config, err := ReadConfig(path, profile)
	if err != nil {
		return nil, fmt.Errorf("%w; repair the configuration before discovery", err)
	}
	return config, nil
}

func sameConfigLocation(left, right string) bool {
	leftParent, leftName := filepath.Split(left)
	rightParent, rightName := filepath.Split(right)
	leftCanonical, leftErr := CanonicalPath(leftParent)
	rightCanonical, rightErr := CanonicalPath(rightParent)
	return leftErr == nil && rightErr == nil && SamePath(leftCanonical, rightCanonical) &&
		(leftName == rightName || SamePath(left, right))
}

func configPresent(path string) (bool, error) {
	_, err := os.Lstat(path)
	if err == nil {
		return true, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	// A missing file is normal. A present dangling .context symlink is a broken
	// configuration location, rather than permission to fall through.
	parent := filepath.Dir(path)
	info, parentErr := os.Lstat(parent)
	if parentErr == nil && info.Mode()&os.ModeSymlink != 0 {
		if _, err := os.Stat(parent); err != nil {
			return false, err
		}
	}
	if parentErr != nil && !errors.Is(parentErr, os.ErrNotExist) {
		return false, parentErr
	}
	return false, nil
}

func (r *resolver) atDirectory(directory string) ([]candidate, error) {
	var projects, workspaces []candidate
	if r.personal != nil {
		if r.request.Kind != Workspace {
			for i := range r.personal.Projects {
				entry := &r.personal.Projects[i]
				if entry.Directory.Err == nil && SamePath(entry.Directory.Canonical, directory) {
					projects = append(projects, candidate{origin: entry.Origin, directory: &entry.Directory, project: &entry.ProjectBinding})
				}
			}
		}
		if r.request.Kind != Project {
			for i := range r.personal.Workspaces {
				entry := &r.personal.Workspaces[i]
				if entry.Directory != nil && entry.Directory.Err == nil && SamePath(entry.Directory.Canonical, directory) {
					workspaces = append(workspaces, candidate{origin: entry.Origin, directory: entry.Directory, workspace: &entry.WorkspaceDeclaration})
				}
			}
		}
	}
	localPath := filepath.Join(directory, ".context", "config.md")
	if !sameConfigLocation(localPath, r.personalPath) {
		config, err := r.loadConfig(localPath, SharedConfig)
		if err != nil {
			return nil, err
		}
		if config != nil {
			if config.Project != nil && r.request.Kind != Workspace {
				projects = append(projects, candidate{origin: config.Project.Origin, project: config.Project})
			}
			if config.Workspace != nil && r.request.Kind != Project {
				workspaces = append(workspaces, candidate{origin: config.Workspace.Origin, workspace: config.Workspace})
			}
		}
	}
	if r.request.Kind != Workspace {
		markerPath := filepath.Join(directory, "project.md")
		_, err := os.Lstat(markerPath)
		if err == nil || !errors.Is(err, os.ErrNotExist) {
			marker := candidate{origin: Origin{Path: markerPath, Field: "project"}, marker: true}
			records := PathValue{Origin: marker.origin, Authored: directory, Path: directory, Canonical: directory}
			marker.project = &ProjectBinding{Origin: marker.origin, Records: records}
			if err != nil {
				marker.err = fmt.Errorf("inspect project marker %s: %w; repair the reserved marker", markerPath, err)
			} else {
				marker.err = validateManifest(directory)
			}
			projects = append(projects, marker)
		}
	}
	if len(projects) != 0 {
		return projects, nil
	}
	return workspaces, nil
}

func (r *resolver) alias() (Scope, error) {
	name := strings.TrimPrefix(r.request.Selector, "@")
	if name == "" || r.request.Kind == Any {
		return Scope{}, errors.New("a nonempty alias requires --project @NAME or --workspace @NAME")
	}
	if r.personal != nil {
		if r.request.Kind == Project {
			for i := range r.personal.Projects {
				entry := &r.personal.Projects[i]
				if entry.Alias == name {
					return r.selectCandidates(entry.Directory.Canonical, []candidate{{origin: entry.Origin, directory: &entry.Directory, project: &entry.ProjectBinding}}, false)
				}
			}
		} else {
			for i := range r.personal.Workspaces {
				entry := &r.personal.Workspaces[i]
				if entry.Alias == name {
					directory := ""
					if entry.Directory != nil {
						directory = entry.Directory.Canonical
					}
					return r.selectCandidates(directory, []candidate{{origin: entry.Origin, directory: entry.Directory, workspace: &entry.WorkspaceDeclaration}}, false)
				}
			}
		}
	}
	return Scope{}, fmt.Errorf("unknown %s alias %q in %s; add a personal registration or choose a directory", r.request.Kind, name, r.personalPath)
}

func (r *resolver) checkUnresolvedBindings() error {
	if r.personal == nil {
		return nil
	}
	var bindings []PathValue
	if r.request.Kind != Workspace {
		for _, entry := range r.personal.Projects {
			bindings = append(bindings, entry.Directory)
		}
	}
	if r.request.Kind != Project {
		for _, entry := range r.personal.Workspaces {
			if entry.Directory != nil {
				bindings = append(bindings, *entry.Directory)
			}
		}
	}
	for _, binding := range bindings {
		// A non-directory component proves this path cannot contain the start.
		if errors.Is(binding.Err, syscall.ENOTDIR) {
			continue
		}
		// An unreadable component can hide a symlink into the start directory,
		// even when its known prefix is elsewhere. Ordinary missing targets have
		// a canonical inferred identity and do not enter this case.
		if binding.Err != nil {
			return pathFailure(binding, "cannot establish whether this binding contains the start directory", binding.Err)
		}
	}
	return nil
}

func (r *resolver) selectCandidates(directory string, candidates []candidate, validateBinding bool) (Scope, error) {
	scope := Scope{StartDirectory: r.start, BindingDirectory: directory, Origins: []Origin{}}
	for _, item := range candidates {
		if item.err != nil {
			return Scope{}, item.err
		}
		scope.Origins = append(scope.Origins, item.origin)
	}
	if candidates[0].project != nil {
		scope.Kind = Project
		selected := candidates[0]
		var authored *candidate
		for i := range candidates {
			item := &candidates[i]
			if !samePathValue(selected.project.Records, item.project.Records) {
				return Scope{}, conflict(Project, directory, selected, *item)
			}
			if !item.marker {
				if authored != nil && !samePathSet(authored.project.AllowSources, item.project.AllowSources) {
					return Scope{}, conflict(Project, directory, *authored, *item)
				}
				authored = item
			}
		}
		if authored != nil {
			selected = *authored
		}
		for _, item := range candidates {
			if err := r.validateCandidate(item, validateBinding); err != nil {
				return Scope{}, err
			}
		}
		scope.Project = &ResolvedProject{Records: selected.project.Records.Canonical, AllowSources: uniquePaths(selected.project.AllowSources)}
		return scope, nil
	}
	scope.Kind = Workspace
	for _, item := range candidates[1:] {
		if !sameWorkspace(*candidates[0].workspace, *item.workspace) {
			return Scope{}, conflict(Workspace, directory, candidates[0], item)
		}
	}
	for _, item := range candidates {
		if err := r.validateCandidate(item, validateBinding); err != nil {
			return Scope{}, err
		}
	}
	scope.Workspace = candidates[0].workspace
	return scope, nil
}

func (r *resolver) validateCandidate(item candidate, validateBinding bool) error {
	if err := r.ctx.Err(); err != nil {
		return err
	}
	if validateBinding && item.directory != nil {
		if err := r.validatePath(*item.directory); err != nil {
			return err
		}
	}
	if item.project == nil {
		return nil
	}
	if err := r.validatePath(item.project.Records); err != nil {
		return err
	}
	if err := validateManifest(item.project.Records.Canonical); err != nil {
		return pathFailure(item.project.Records, "selected records require a valid Project manifest", err)
	}
	for _, root := range item.project.AllowSources {
		if err := r.validatePath(root); err != nil {
			return err
		}
	}
	return nil
}

func (r *resolver) validatePath(path PathValue) error {
	if path.Err != nil {
		return pathFailure(path, "cannot resolve selected directory", path.Err)
	}
	spelling := path.Path
	if r.replacement != nil && sameConfigLocation(path.Origin.Path, r.replacement.Path) && !filepath.IsAbs(path.Authored) {
		// Setup can validate a document before creating its .context directory.
		// Interpret leading parent traversal out of that future directory, but
		// retain all remaining components for normal filesystem validation.
		parent, _ := filepath.Split(r.replacement.Path)
		if _, err := os.Lstat(strings.TrimSuffix(parent, string(filepath.Separator))); errors.Is(err, os.ErrNotExist) {
			base, err := CanonicalPath(parent)
			if err != nil {
				return pathFailure(path, "cannot resolve proposed configuration directory", err)
			}
			parts := strings.Split(path.Authored, string(filepath.Separator))
			for len(parts) > 0 && (parts[0] == "." || parts[0] == "..") {
				if parts[0] == ".." {
					base = filepath.Dir(base)
				}
				parts = parts[1:]
				if _, err := os.Stat(base); err == nil {
					break
				}
			}
			spelling = appendPath(base, strings.Join(parts, string(filepath.Separator)))
		}
	}
	if err := checkDirectory(spelling); err != nil {
		return pathFailure(path, "selected directory is unavailable", err)
	}
	return nil
}

func pathFailure(path PathValue, problem string, err error) error {
	return fmt.Errorf("%s field %s value %q resolves to %q: %s: %w; repair the directory or its declaration", path.Origin.Path, path.Origin.Field, path.Authored, path.Canonical, problem, err)
}

func checkDirectory(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", path)
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		return err
	}
	defer root.Close()
	file, err := root.Open(".")
	if err != nil {
		return err
	}
	defer file.Close()
	return nil
}

func validateManifest(directory string) error {
	reader, err := recordread.NewReader(directory, nil)
	if err != nil {
		return err
	}
	defer reader.Close()
	path := filepath.Join(directory, "project.md")
	if err := checkRegularFile(path); err != nil {
		return fmt.Errorf("project marker %s: %w; repair the reserved marker", path, err)
	}
	source, diagnostic := reader.Read(path, recordread.RecordSource)
	if diagnostic != nil {
		return fmt.Errorf("project marker %s: %s; repair the reserved marker", path, diagnostic.Message)
	}
	document, err := recordread.ParseDocument([]byte(source.Text))
	if err != nil {
		return fmt.Errorf("project marker %s: %w; repair the reserved marker", path, err)
	}
	for _, field := range []string{"type", "id", "title"} {
		value := document.Metadata[field]
		text, ok := value.(string)
		if !ok || strings.TrimSpace(text) == "" || field == "type" && text != "Project" {
			return fmt.Errorf("project marker %s field %s (%v): requires Project type, nonempty id, and nonempty title; repair the reserved marker", path, field, value)
		}
	}
	return nil
}
