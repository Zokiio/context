package initialization

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Zokiio/context/internal/discovery"
)

type Request struct {
	Directory, Home, Executable, Version         string
	Title, Tracker, Skills, Docs, Records, Local string
	AllowSources                                 []string
}

type FileResult struct {
	Path, Status, Reason string
}

type Result struct {
	Files        []FileResult
	Binding      *discovery.SetupSummary
	BindingError string
	Pointers     string
}

func (result Result) Incomplete() bool {
	if result.BindingError != "" {
		return true
	}
	for _, file := range result.Files {
		if file.Status == "skipped" {
			return true
		}
	}
	return false
}

// Initialize creates bootstrap files without replacing existing content. Setup
// retains ownership of configuration writes. A partial result reports conflicts;
// an error retains the file results from writes completed before the failure.
func Initialize(ctx context.Context, request Request) (Result, error) {
	var result Result
	if err := prepareRequest(&request); err != nil {
		return result, err
	}
	documents, err := renderGuidance(request, false)
	if err != nil {
		return result, err
	}
	result.Pointers = instructionPointers(request)
	for _, doc := range documents {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		file, err := createFile(doc.path, doc.text)
		result.Files = append(result.Files, file)
		if err != nil {
			return result, err
		}
	}
	manifest := filepath.Join(request.Records, "project.md")
	if _, err := os.Lstat(manifest); err == nil {
		result.Files = append(result.Files, FileResult{manifest, "preserved", "existing manifest; --title applies only to a new manifest"})
	} else if !os.IsNotExist(err) {
		return result, err
	} else {
		var id [16]byte
		if _, err := rand.Read(id[:]); err != nil {
			return result, err
		}
		id[6], id[8] = id[6]&0x0f|0x40, id[8]&0x3f|0x80
		uuid := fmt.Sprintf("%x-%x-%x-%x-%x", id[:4], id[4:6], id[6:8], id[8:10], id[10:])
		body := fmt.Sprintf("---\ntype: Project\nid: %s\ntitle: %q\n---\n\n## Goals\n\n## Current commitments\n\nNone\n\n## Open decisions\n\nNone\n", uuid, request.Title)
		file, err := createFile(manifest, []byte(body))
		result.Files = append(result.Files, file)
		if err != nil {
			return result, err
		}
	}
	version, err := createFile(filepath.Join(request.Local, "ctx-version.txt"), []byte(request.Version))
	result.Files = append(result.Files, version)
	if err != nil {
		return result, err
	}
	local, _ := filepath.Rel(request.Directory, request.Local)
	for _, ignore := range []struct {
		path  string
		rules []string
	}{
		{filepath.Join(request.Directory, ".gitignore"), []string{ignoreDirectory(local), "/.context-cache/", "/.context/config.md.lock"}},
		{filepath.Join(request.Records, ".gitignore"), []string{"/acceptances/"}},
	} {
		file, err := appendIgnore(ignore.path, ignore.rules)
		result.Files = append(result.Files, file)
		if err != nil {
			return result, err
		}
	}
	plan, err := discovery.PrepareSetup(ctx, discovery.SetupRequest{
		Cwd: request.Directory, Home: request.Home, Records: request.Records,
		AllowSources: request.AllowSources,
	})
	if err != nil {
		result.BindingError = err.Error()
		return result, nil
	}
	summary := plan.Summary()
	// Setup retains these sidecars even when a later binding write fails.
	locks := map[string]bool{}
	for _, config := range []string{summary.Destination, filepath.Join(request.Home, ".context", "config.md")} {
		physical, err := discovery.CanonicalPath(config)
		if err != nil {
			return result, err
		}
		_, err = os.Lstat(physical + ".lock")
		locks[physical+".lock"] = err == nil
	}
	applyErr := plan.Apply(ctx)
	for _, config := range []string{summary.Destination, filepath.Join(request.Home, ".context", "config.md")} {
		physical, _ := discovery.CanonicalPath(config)
		lock := physical + ".lock"
		if _, err := os.Lstat(lock); err == nil {
			status := "created"
			if locks[lock] {
				status = "preserved"
			}
			result.Files = append(result.Files, FileResult{lock, status, "setup coordination lock"})
		}
	}
	if applyErr != nil {
		result.BindingError = applyErr.Error()
		return result, applyErr
	}
	result.Binding = &summary
	return result, nil
}

func prepareRequest(request *Request) error {
	if !filepath.IsAbs(request.Directory) || !filepath.IsAbs(request.Home) || !filepath.IsAbs(request.Executable) {
		return errors.New("init requires absolute project, home, and executable paths")
	}
	for name, value := range map[string]string{"title": request.Title, "tracker": request.Tracker} {
		if strings.TrimSpace(value) == "" || strings.ContainsAny(value, "\r\n") {
			return fmt.Errorf("init --%s must be a nonempty single line", name)
		}
	}
	for _, value := range []*string{&request.Directory, &request.Skills, &request.Docs, &request.Records, &request.Local} {
		if strings.TrimSpace(*value) == "" || strings.ContainsAny(*value, "\r\n") {
			return errors.New("init directory paths must be nonempty single lines")
		}
		path := *value
		if !filepath.IsAbs(path) {
			path = request.Directory + string(filepath.Separator) + path
		}
		canonical, err := discovery.CanonicalPath(path)
		if err != nil {
			return err
		}
		*value = canonical
	}
	if !inside(request.Directory, request.Local) || discovery.SamePath(request.Directory, request.Local) {
		return errors.New("init --local-dir must be a directory inside the target project")
	}
	cache := filepath.Join(request.Directory, ".context-cache")
	for _, durable := range []string{request.Skills, request.Docs, request.Records} {
		for _, disposable := range []string{request.Local, cache, filepath.Join(request.Records, "acceptances")} {
			if inside(disposable, durable) {
				return fmt.Errorf("init durable directory %s is inside disposable directory %s", durable, disposable)
			}
		}
	}
	if inside(request.Records, request.Local) || inside(request.Records, cache) {
		return errors.New("init records must be separate from disposable local files and recovery cache")
	}
	for _, root := range request.AllowSources {
		path := root
		if !filepath.IsAbs(path) {
			path = request.Directory + string(filepath.Separator) + path
		}
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("init allowed source %q: %w", root, err)
		}
		if strings.TrimSpace(root) == "" || !info.IsDir() {
			return fmt.Errorf("init allowed source %q must be an existing directory", root)
		}
	}
	return nil
}

func inside(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func instructionPointers(request Request) string {
	link := func(path string) string {
		rel, _ := filepath.Rel(request.Directory, path)
		return filepath.ToSlash(rel)
	}
	return fmt.Sprintf("Before implementing or verifying a known task, read %s.\nBefore continuing interrupted work or publishing useful recovery notes, read %s.\nFollow %s for tracker authority and %s for completion evidence.\n",
		link(filepath.Join(request.Skills, "task-context", "SKILL.md")),
		link(filepath.Join(request.Skills, "recovery-notes", "SKILL.md")),
		link(filepath.Join(request.Docs, "tracker.md")), link(filepath.Join(request.Docs, "acceptance.md")))
}

func ignoreDirectory(path string) string {
	escape := strings.NewReplacer("\\", "\\\\", "*", "\\*", "?", "\\?", "[", "\\[", "]", "\\]", "!", "\\!", "#", "\\#", " ", "\\ ")
	return "/" + escape.Replace(filepath.ToSlash(path)) + "/"
}

func createFile(path string, text []byte) (FileResult, error) {
	result := FileResult{Path: path, Status: "skipped"}
	if info, err := os.Lstat(path); err == nil {
		if info.Mode().IsRegular() {
			current, err := os.ReadFile(path)
			if err != nil {
				return result, err
			}
			if bytes.Equal(current, text) {
				result.Status = "unchanged"
				return result, nil
			}
		}
		result.Reason = "existing file differs or is not a regular file; review it manually"
		return result, nil
	} else if !os.IsNotExist(err) {
		return result, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return result, err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if os.IsExist(err) {
		result.Reason = "file appeared during initialization; preserved for review"
		return result, nil
	}
	if err != nil {
		return result, err
	}
	_, writeErr := file.Write(text)
	err = errors.Join(writeErr, file.Close())
	if err != nil {
		result.Reason = "creation failed; inspect the partial file"
		return result, err
	}
	result.Status = "created"
	return result, nil
}

func appendIgnore(path string, rules []string) (FileResult, error) {
	result := FileResult{Path: path, Status: "skipped"}
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return createFile(path, []byte(strings.Join(rules, "\n")+"\n"))
	}
	if err != nil {
		return result, err
	}
	if !info.Mode().IsRegular() {
		result.Reason = "ignore file is not a regular file; add ignore rules manually"
		return result, nil
	}
	current, err := os.ReadFile(path)
	if err != nil {
		return result, err
	}
	var additions []string
	for _, rule := range rules {
		found := false
		for _, line := range strings.Split(string(current), "\n") {
			found = found || strings.TrimSuffix(line, "\r") == rule
		}
		if !found {
			additions = append(additions, rule)
		}
	}
	if len(additions) == 0 {
		result.Status = "unchanged"
		return result, nil
	}
	appendix := strings.Join(additions, "\n") + "\n"
	if len(current) != 0 && current[len(current)-1] != '\n' {
		appendix = "\n" + appendix
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		return result, err
	}
	_, writeErr := file.WriteString(appendix)
	if err := errors.Join(writeErr, file.Close()); err != nil {
		return result, err
	}
	result.Status = "appended"
	return result, nil
}
