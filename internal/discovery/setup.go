package discovery

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/goccy/go-yaml"
)

// SetupRequest connects existing records. Filesystem inputs are relative to Cwd;
// an omitted Directory means Cwd. AliasSet distinguishes omission from an empty
// value so replacement can retain an existing alias.
type SetupRequest struct {
	Cwd, Home, Records, Directory string
	Personal                      bool
	Alias                         string
	AliasSet                      bool
	AllowSources                  []string
	Replace                       bool
}

type SetupChange string

const (
	SetupCreate    SetupChange = "create"
	SetupUpdate    SetupChange = "update"
	SetupUnchanged SetupChange = "unchanged"
)

type SetupSummary struct {
	Change                                 SetupChange
	Directory, Records, Destination, Alias string
	AllowSources, RemovedSources           []string
	PreviousRecords, PreviousAlias         string
}

// SetupPlan retains the exact document and observed file behind its summary.
// PrepareSetup never writes; callers can display or inspect the plan before Apply.
type SetupPlan struct {
	summary  SetupSummary
	document string
	target   string
	before   setupSnapshot
	request  Request
	proposed *Config
}

func (p *SetupPlan) Summary() SetupSummary {
	summary := p.summary
	summary.AllowSources = slices.Clone(summary.AllowSources)
	summary.RemovedSources = slices.Clone(summary.RemovedSources)
	return summary
}

func (p *SetupPlan) Document() string { return p.document }

func PrepareSetup(ctx context.Context, request SetupRequest) (*SetupPlan, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !filepath.IsAbs(request.Cwd) || !filepath.IsAbs(request.Home) {
		return nil, errors.New("setup requires absolute cwd and home directories")
	}
	if request.Alias != "" {
		request.AliasSet = true
	}
	if request.AliasSet && (!request.Personal || strings.TrimSpace(request.Alias) == "") {
		return nil, errors.New("--alias requires --personal and a nonempty name")
	}
	if request.Directory == "" {
		request.Directory = request.Cwd
	}
	directory, err := setupDirectory(request.Cwd, request.Directory, "binding directory")
	if err != nil {
		return nil, err
	}
	records, err := setupDirectory(request.Cwd, request.Records, "records")
	if err != nil {
		return nil, err
	}
	// Validate the entered path before using its canonical spelling for output.
	if err := validateManifest(commandPath(request.Cwd, request.Records)); err != nil {
		return nil, fmt.Errorf("setup records %q: %w", request.Records, err)
	}
	roots := make([]string, 0, len(request.AllowSources))
	for _, root := range request.AllowSources {
		canonical, err := setupDirectory(request.Cwd, root, "allowed source")
		if err != nil {
			return nil, err
		}
		if !containsSetupPath(roots, canonical) {
			roots = append(roots, canonical)
		}
	}
	profile, destination := SharedConfig, filepath.Join(directory, ".context", "config.md")
	personalPath := appendPath(request.Home, ".context/config.md")
	if request.Personal {
		if err := checkDirectory(request.Home); err != nil {
			return nil, fmt.Errorf("setup personal home %q: %w", request.Home, err)
		}
		profile, destination = PersonalConfig, personalPath
	} else if sameConfigLocation(destination, personalPath) {
		return nil, fmt.Errorf("shared setup destination %s is the reserved personal registry; use --personal to register this directory", destination)
	}
	target, before, err := observeSetupFile(destination)
	if err != nil {
		return nil, err
	}
	config := &Config{Path: destination, Profile: profile, Metadata: map[string]any{"type": "ContextConfig", "version": 1}}
	if before.info != nil {
		config, err = ParseConfig(destination, before.data, profile)
		if err != nil {
			return nil, fmt.Errorf("setup cannot replace malformed configuration: %w", err)
		}
	}
	entry, previous, previousAlias, err := setupEntry(config, directory, request)
	if err != nil {
		return nil, err
	}
	entry["records"], err = setupSavedPath(destination, records, request.Personal)
	if err != nil {
		return nil, err
	}
	savedRoots := make([]string, 0, len(roots))
	for _, root := range roots {
		saved, err := setupSavedPath(destination, root, request.Personal)
		if err != nil {
			return nil, err
		}
		savedRoots = append(savedRoots, saved)
	}
	entry["allowSources"] = savedRoots
	metadata, err := yaml.Marshal(config.Metadata)
	if err != nil {
		return nil, fmt.Errorf("encode setup configuration %s: %w", destination, err)
	}
	document := "---\n" + string(metadata) + "---\n" + string(config.Body)
	proposed, err := ParseConfig(destination, []byte(document), profile)
	if err != nil {
		return nil, fmt.Errorf("validate proposed setup configuration: %w", err)
	}
	plan := &SetupPlan{document: document, target: target, before: before, proposed: proposed,
		request: Request{Cwd: directory, Home: request.Home, Kind: Project},
		summary: SetupSummary{Change: SetupCreate, Directory: directory, Records: records, Destination: destination, AllowSources: roots}}
	if request.Personal {
		plan.summary.Alias, _ = entry["alias"].(string)
	}
	if previous != nil {
		plan.summary.Change = SetupUpdate
		same := previous.Records.Err == nil && SamePath(previous.Records.Canonical, records) && sameSetupRoots(previous.AllowSources, roots) && previousAlias == plan.summary.Alias
		if same {
			plan.summary.Change = SetupUnchanged
			plan.document = string(before.data)
			plan.proposed, err = ParseConfig(destination, before.data, profile)
			if err != nil {
				return nil, err
			}
		} else if !request.Replace {
			return nil, fmt.Errorf("project binding %s field %s already differs from the requested records, roots, or alias; inspect it and use --replace to update this entry", destination, previous.Origin.Field)
		}
		if !SamePath(previous.Records.Canonical, records) {
			plan.summary.PreviousRecords = previous.Records.Canonical
		}
		if previousAlias != plan.summary.Alias {
			plan.summary.PreviousAlias = previousAlias
		}
		for _, root := range previous.AllowSources {
			if !containsSetupPath(roots, root.Canonical) && !containsSetupPath(plan.summary.RemovedSources, root.Canonical) {
				plan.summary.RemovedSources = append(plan.summary.RemovedSources, root.Canonical)
			}
		}
	}
	if _, err := resolve(ctx, plan.request, plan.proposed); err != nil {
		return nil, fmt.Errorf("proposed setup binding conflicts with or cannot use effective declarations: %w", err)
	}
	return plan, nil
}

func setupDirectory(cwd, value, field string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("setup %s must be a nonempty directory path", field)
	}
	entered := commandPath(cwd, value)
	canonical, err := CanonicalPath(entered)
	if err == nil {
		err = checkDirectory(entered)
	}
	if err != nil {
		return "", fmt.Errorf("setup %s %q resolves to %q: %w; choose an accessible directory", field, value, canonical, err)
	}
	return canonical, nil
}

func setupSavedPath(config, target string, personal bool) (string, error) {
	if personal {
		return target, nil
	}
	parent, err := CanonicalPath(filepath.Dir(config))
	if err != nil {
		return "", err
	}
	return filepath.Rel(parent, target)
}

func setupEntry(config *Config, directory string, request SetupRequest) (map[string]any, *ProjectBinding, string, error) {
	if !request.Personal {
		entry, ok := config.Metadata["project"].(map[string]any)
		if !ok {
			entry = map[string]any{}
			config.Metadata["project"] = entry
		}
		return entry, config.Project, "", nil
	}
	index := -1
	for i, entry := range config.Projects {
		if entry.Directory.Err != nil {
			return nil, nil, "", pathFailure(entry.Directory, "cannot match existing registration", entry.Directory.Err)
		}
		if SamePath(entry.Directory.Canonical, directory) {
			if index != -1 {
				return nil, nil, "", fmt.Errorf("personal configuration %s has ambiguous project entries %s and %s for %s; repair the duplicate directory bindings", config.Path, config.Projects[index].Key, entry.Key, directory)
			}
			index = i
		}
	}
	if request.AliasSet {
		for i, entry := range config.Projects {
			if i != index && entry.Alias == request.Alias {
				return nil, nil, "", fmt.Errorf("personal alias %q already belongs to %s field %s; choose another alias without retargeting that entry", request.Alias, config.Path, entry.Origin.Field)
			}
		}
	}
	entries, _ := config.Metadata["projects"].([]any)
	var entry map[string]any
	var previous *ProjectBinding
	alias := ""
	if index >= 0 {
		entry = entries[index].(map[string]any)
		previous = &config.Projects[index].ProjectBinding
		alias = config.Projects[index].Alias
	} else {
		key, err := setupUUID()
		if err != nil {
			return nil, nil, "", err
		}
		entry = map[string]any{"key": key}
		config.Metadata["projects"] = append(entries, entry)
	}
	entry["directory"] = directory
	if request.AliasSet {
		entry["alias"] = request.Alias
	}
	return entry, previous, alias, nil
}

func setupUUID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", fmt.Errorf("generate project registration key: %w", err)
	}
	value[6] = value[6]&0x0f | 0x40
	value[8] = value[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", value[:4], value[4:6], value[6:8], value[8:10], value[10:]), nil
}

func containsSetupPath(paths []string, target string) bool {
	for _, path := range paths {
		if SamePath(path, target) {
			return true
		}
	}
	return false
}

func sameSetupRoots(before []PathValue, after []string) bool {
	canonical := uniquePaths(before)
	if len(canonical) != len(after) {
		return false
	}
	for _, root := range before {
		if root.Err != nil || !containsSetupPath(after, root.Canonical) {
			return false
		}
	}
	return true
}
