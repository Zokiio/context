package factory

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Zokiio/context/internal/orientation"
	"github.com/Zokiio/context/internal/recordread"
)

func (s *Service) RegisterExecutor(ctx context.Context, actor Principal, request RegisterExecutorRequest) (ExecutorProfile, error) {
	if err := validateProfile(request.Key, request.DisplayName, request.Command); err != nil {
		return ExecutorProfile{}, err
	}
	s.state.mutations.Lock()
	defer s.state.mutations.Unlock()
	tx, err := s.state.db.BeginTx(ctx, nil)
	if err != nil {
		return ExecutorProfile{}, fault(CodeInternal, "start executor registration", err)
	}
	defer tx.Rollback()
	current, err := requireInstallationAdmin(ctx, tx, actor)
	if err != nil {
		return ExecutorProfile{}, err
	}
	if saved, ok, hash, err := receipt[ExecutorProfile](ctx, tx, current.UserID, "register_executor", request.RequestID, request); err != nil || ok {
		return saved, err
	} else {
		now := s.state.now().UTC()
		profile := ExecutorProfile{Key: request.Key, DisplayName: strings.TrimSpace(request.DisplayName), Command: append([]string(nil), request.Command...), Fixture: request.Fixture, CreatedAt: now}
		command, _ := marshal(profile.Command)
		if _, err := tx.ExecContext(ctx, `INSERT INTO executor_profiles(key, display_name, command_json, fixture, created_at) VALUES(?,?,?,?,?)`, profile.Key, profile.DisplayName, command, boolInt(profile.Fixture), formatTime(now)); err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "unique") {
				return ExecutorProfile{}, fault(CodeConflict, "executor profile already exists", err)
			}
			return ExecutorProfile{}, fault(CodeInternal, "register executor profile", err)
		}
		if err := saveReceipt(ctx, tx, current.UserID, "register_executor", request.RequestID, hash, profile, false, now); err != nil {
			return ExecutorProfile{}, err
		}
		if err := tx.Commit(); err != nil {
			return ExecutorProfile{}, fault(CodeInternal, "commit executor registration", err)
		}
		return profile, nil
	}
}

func (s *Service) ListExecutors(ctx context.Context, actor Principal) ([]ExecutorProfile, error) {
	if _, err := requireInstallationAdmin(ctx, s.state.db, actor); err != nil {
		return nil, err
	}
	rows, err := s.state.db.QueryContext(ctx, `SELECT key, display_name, command_json, fixture, created_at FROM executor_profiles ORDER BY key`)
	if err != nil {
		return nil, fault(CodeInternal, "list executor profiles", err)
	}
	defer rows.Close()
	profiles := []ExecutorProfile{}
	for rows.Next() {
		var profile ExecutorProfile
		var command, created string
		if err := rows.Scan(&profile.Key, &profile.DisplayName, &command, &profile.Fixture, &created); err != nil {
			return nil, fault(CodeInternal, "read executor profile", err)
		}
		_ = json.Unmarshal([]byte(command), &profile.Command)
		profile.CreatedAt, _ = parseTime(created)
		profiles = append(profiles, profile)
	}
	return profiles, rows.Err()
}

func (s *Service) RegisterCheck(ctx context.Context, actor Principal, request RegisterCheckRequest) (CheckProfile, error) {
	if err := validateProfile(request.Key, request.DisplayName, request.Command); err != nil {
		return CheckProfile{}, err
	}
	if strings.TrimSpace(request.Environment) == "" {
		return CheckProfile{}, fault(CodeInvalid, "check environment is required", nil)
	}
	s.state.mutations.Lock()
	defer s.state.mutations.Unlock()
	tx, err := s.state.db.BeginTx(ctx, nil)
	if err != nil {
		return CheckProfile{}, fault(CodeInternal, "start check registration", err)
	}
	defer tx.Rollback()
	current, err := requireInstallationAdmin(ctx, tx, actor)
	if err != nil {
		return CheckProfile{}, err
	}
	if saved, ok, hash, err := receipt[CheckProfile](ctx, tx, current.UserID, "register_check", request.RequestID, request); err != nil || ok {
		return saved, err
	} else {
		now := s.state.now().UTC()
		profile := CheckProfile{Key: request.Key, DisplayName: strings.TrimSpace(request.DisplayName), Command: append([]string(nil), request.Command...), Environment: strings.TrimSpace(request.Environment), Fixture: request.Fixture, CreatedAt: now}
		command, _ := marshal(profile.Command)
		if _, err := tx.ExecContext(ctx, `INSERT INTO check_profiles(key, display_name, command_json, environment, fixture, created_at) VALUES(?,?,?,?,?,?)`, profile.Key, profile.DisplayName, command, profile.Environment, boolInt(profile.Fixture), formatTime(now)); err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "unique") {
				return CheckProfile{}, fault(CodeConflict, "check profile already exists", err)
			}
			return CheckProfile{}, fault(CodeInternal, "register check profile", err)
		}
		if err := saveReceipt(ctx, tx, current.UserID, "register_check", request.RequestID, hash, profile, false, now); err != nil {
			return CheckProfile{}, err
		}
		if err := tx.Commit(); err != nil {
			return CheckProfile{}, fault(CodeInternal, "commit check registration", err)
		}
		return profile, nil
	}
}

func (s *Service) ListChecks(ctx context.Context, actor Principal) ([]CheckProfile, error) {
	if _, err := requireInstallationAdmin(ctx, s.state.db, actor); err != nil {
		return nil, err
	}
	rows, err := s.state.db.QueryContext(ctx, `SELECT key, display_name, command_json, environment, fixture, created_at FROM check_profiles ORDER BY key`)
	if err != nil {
		return nil, fault(CodeInternal, "list check profiles", err)
	}
	defer rows.Close()
	profiles := []CheckProfile{}
	for rows.Next() {
		var profile CheckProfile
		var command, created string
		if err := rows.Scan(&profile.Key, &profile.DisplayName, &command, &profile.Environment, &profile.Fixture, &created); err != nil {
			return nil, fault(CodeInternal, "read check profile", err)
		}
		_ = json.Unmarshal([]byte(command), &profile.Command)
		profile.CreatedAt, _ = parseTime(created)
		profiles = append(profiles, profile)
	}
	return profiles, rows.Err()
}

func (s *Service) RegisterProject(ctx context.Context, actor Principal, request RegisterProjectRequest) (Project, error) {
	if request.RequestID == "" || request.BundlePath == "" || request.RepositoryPath == "" {
		return Project{}, fault(CodeInvalid, "requestId, bundlePath, and repositoryPath are required", nil)
	}
	request.Policy.RequiredChecks = uniqueSorted(request.Policy.RequiredChecks)
	request.Policy.AllowedExecutors = uniqueSorted(request.Policy.AllowedExecutors)
	if len(request.Policy.RequiredChecks) == 0 || len(request.Policy.AllowedExecutors) == 0 {
		return Project{}, fault(CodeInvalid, "project policy requires checks and allowed executors", nil)
	}
	if request.Policy.MaxRepairAttempts < 0 || request.Policy.MaxJobDurationSeconds <= 0 {
		return Project{}, fault(CodeInvalid, "project repair and duration bounds are invalid", nil)
	}
	if request.ReaderLimits.MaxFiles < 0 || request.ReaderLimits.MaxBytes < 0 {
		return Project{}, fault(CodeInvalid, "reader limits cannot be negative", nil)
	}
	bundle, err := canonicalDirectory(request.BundlePath)
	if err != nil {
		return Project{}, fault(CodeInvalid, "resolve bundlePath", err)
	}
	repository, err := canonicalDirectory(request.RepositoryPath)
	if err != nil {
		return Project{}, fault(CodeInvalid, "resolve repositoryPath", err)
	}
	if err := s.requireGitRepository(ctx, repository); err != nil {
		return Project{}, err
	}
	allowed := make([]string, 0, len(request.AllowedSourceRoots))
	for _, root := range request.AllowedSourceRoots {
		canonical, err := canonicalDirectory(root)
		if err != nil {
			return Project{}, fault(CodeInvalid, "resolve allowedSourceRoots", err)
		}
		allowed = append(allowed, canonical)
	}
	allowed = uniqueSorted(allowed)
	projectID, title, err := readProjectIdentity(bundle, allowed, request.ReaderLimits)
	if err != nil {
		return Project{}, err
	}
	request.BundlePath, request.RepositoryPath, request.AllowedSourceRoots = bundle, repository, allowed
	s.state.mutations.Lock()
	defer s.state.mutations.Unlock()
	tx, err := s.state.db.BeginTx(ctx, nil)
	if err != nil {
		return Project{}, fault(CodeInternal, "start project registration", err)
	}
	defer tx.Rollback()
	current, err := requireInstallationAdmin(ctx, tx, actor)
	if err != nil {
		return Project{}, err
	}
	if saved, ok, hash, err := receipt[Project](ctx, tx, current.UserID, "register_project", request.RequestID, request); err != nil || ok {
		return saved, err
	} else {
		if err := validateProjectProfiles(ctx, tx, request.Policy); err != nil {
			return Project{}, err
		}
		now := s.state.now().UTC()
		project := Project{ID: ProjectID(projectID), Title: title, BundlePath: bundle, RepositoryPath: repository, AllowedSourceRoots: allowed, ReaderLimits: request.ReaderLimits, Policy: request.Policy, CreatedAt: now}
		if existing, found, err := readProjectByIdentity(ctx, tx, project.ID, bundle); err != nil {
			return Project{}, err
		} else if found {
			if !sameProjectRegistration(existing, project) {
				return Project{}, fault(CodeConflict, "project identity or bundle is already registered differently", nil)
			}
			project = existing
		} else {
			sources, _ := marshal(allowed)
			policy, _ := marshal(request.Policy)
			if _, err := tx.ExecContext(ctx, `INSERT INTO projects(id, title, bundle_path, repository_path, allowed_sources_json, reader_max_files, reader_max_bytes, policy_json, created_at) VALUES(?,?,?,?,?,?,?,?,?)`, project.ID, project.Title, project.BundlePath, project.RepositoryPath, sources, project.ReaderLimits.MaxFiles, project.ReaderLimits.MaxBytes, policy, formatTime(now)); err != nil {
				return Project{}, fault(CodeConflict, "project is already registered", err)
			}
		}
		if err := saveReceipt(ctx, tx, current.UserID, "register_project", request.RequestID, hash, project, false, now); err != nil {
			return Project{}, err
		}
		if err := tx.Commit(); err != nil {
			return Project{}, fault(CodeInternal, "commit project registration", err)
		}
		return project, nil
	}
}

func (s *Service) ListProjects(ctx context.Context, actor Principal) ([]Project, error) {
	current, err := requirePrincipal(ctx, s.state.db, actor)
	if err != nil {
		return nil, err
	}
	query := `SELECT p.id, p.title, p.bundle_path, p.repository_path, p.allowed_sources_json, p.reader_max_files, p.reader_max_bytes, p.policy_json, p.created_at FROM projects p`
	args := []any{}
	if !current.InstallationAdmin {
		query += ` WHERE EXISTS(SELECT 1 FROM project_grants g WHERE g.project_id=p.id AND g.user_id=? AND g.capability=?)`
		args = append(args, current.UserID, CapabilityRead)
	}
	query += ` ORDER BY p.title, p.id`
	rows, err := s.state.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fault(CodeInternal, "list projects", err)
	}
	defer rows.Close()
	projects := []Project{}
	for rows.Next() {
		project, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		projects = append(projects, project)
	}
	return projects, rows.Err()
}

func (s *Service) Orientation(ctx context.Context, actor Principal, projectID ProjectID) (ProjectOrientation, error) {
	if _, err := authorizeProject(ctx, s.state.db, actor, projectID, CapabilityRead); err != nil {
		return ProjectOrientation{}, err
	}
	project, err := readProject(ctx, s.state.db, projectID)
	if err != nil {
		return ProjectOrientation{}, err
	}
	result, err := orientation.Orient(ctx, orientation.Request{ProjectDir: project.BundlePath, AllowedSourceDirs: project.AllowedSourceRoots, MaxFiles: project.ReaderLimits.MaxFiles, MaxBytes: project.ReaderLimits.MaxBytes})
	if err != nil {
		return ProjectOrientation{}, fault(CodeInternal, "read project orientation", err)
	}
	return ProjectOrientation{Project: project, Result: result}, nil
}

func validateProfile(key, displayName string, command []string) error {
	if strings.TrimSpace(key) == "" || strings.TrimSpace(displayName) == "" || len(command) == 0 || strings.TrimSpace(command[0]) == "" {
		return fault(CodeInvalid, "profile key, displayName, and command are required", nil)
	}
	if len(key) > 100 || strings.ContainsAny(key, " \t\r\n/") {
		return fault(CodeInvalid, "profile key must be a short path-safe name", nil)
	}
	for _, arg := range command {
		if strings.ContainsRune(arg, 0) {
			return fault(CodeInvalid, "command arguments cannot contain NUL", nil)
		}
	}
	return nil
}

func canonicalDirectory(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	canonical, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(canonical)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", errors.New("path is not a directory")
	}
	return filepath.Clean(canonical), nil
}

func readProjectIdentity(bundle string, allowed []string, limits ReaderLimits) (string, string, error) {
	capture, err := recordread.NewCapture(bundle, allowed, recordread.Limits{MaxFiles: limits.MaxFiles, MaxBytes: limits.MaxBytes})
	if err != nil {
		return "", "", fault(CodeInvalid, "open project records", err)
	}
	defer capture.Close()
	source, diagnostic := capture.Read(filepath.Join(bundle, "project.md"), recordread.RecordSource)
	if diagnostic != nil {
		return "", "", fault(CodeInvalid, "read project manifest", errors.New(diagnostic.Message))
	}
	if limit := capture.Admit(source); limit != nil {
		return "", "", fault(CodeInvalid, "project manifest exceeds reader limits", limit)
	}
	document, err := recordread.ParseDocument([]byte(source.Text))
	if err != nil {
		return "", "", fault(CodeInvalid, "parse project manifest", err)
	}
	kind, _ := document.Metadata["type"].(string)
	id, _ := document.Metadata["id"].(string)
	title, _ := document.Metadata["title"].(string)
	if kind != "Project" || strings.TrimSpace(id) == "" || strings.TrimSpace(title) == "" {
		return "", "", fault(CodeInvalid, "project manifest requires type Project, id, and title", nil)
	}
	return strings.TrimSpace(id), strings.TrimSpace(title), nil
}

func (s *Service) requireGitRepository(ctx context.Context, repository string) error {
	command := exec.CommandContext(ctx, s.state.gitBinary, "-C", repository, "rev-parse", "--show-toplevel")
	output, err := command.Output()
	if err != nil {
		return fault(CodeInvalid, "repositoryPath is not a Git repository", err)
	}
	root, err := canonicalDirectory(strings.TrimSpace(string(output)))
	if err != nil || root != repository {
		return fault(CodeInvalid, "repositoryPath must be the Git worktree root", err)
	}
	return nil
}

func validateProjectProfiles(ctx context.Context, tx *sql.Tx, policy ProjectPolicy) error {
	for _, key := range policy.AllowedExecutors {
		var fixture int
		if err := tx.QueryRowContext(ctx, `SELECT fixture FROM executor_profiles WHERE key=?`, key).Scan(&fixture); errors.Is(err, sql.ErrNoRows) {
			return fault(CodeInvalid, "project references an unknown executor profile", nil)
		} else if err != nil {
			return fault(CodeInternal, "read executor profile", err)
		} else if fixture != 0 && !policy.Fixture {
			return fault(CodeInvalid, "fixture executor requires an explicitly fixture project", nil)
		}
	}
	for _, key := range policy.RequiredChecks {
		var fixture int
		if err := tx.QueryRowContext(ctx, `SELECT fixture FROM check_profiles WHERE key=?`, key).Scan(&fixture); errors.Is(err, sql.ErrNoRows) {
			return fault(CodeInvalid, "project references an unknown check profile", nil)
		} else if err != nil {
			return fault(CodeInternal, "read check profile", err)
		} else if fixture != 0 && !policy.Fixture {
			return fault(CodeInvalid, "fixture check requires an explicitly fixture project", nil)
		}
	}
	return nil
}

func readProject(ctx context.Context, q queryer, projectID ProjectID) (Project, error) {
	row := q.QueryRowContext(ctx, `SELECT id, title, bundle_path, repository_path, allowed_sources_json, reader_max_files, reader_max_bytes, policy_json, created_at FROM projects WHERE id=?`, projectID)
	project, err := scanProject(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Project{}, fault(CodeNotFound, "project not found", nil)
	}
	return project, err
}

type rowScanner interface{ Scan(...any) error }

func scanProject(scanner rowScanner) (Project, error) {
	var project Project
	var sources, policy, created string
	if err := scanner.Scan(&project.ID, &project.Title, &project.BundlePath, &project.RepositoryPath, &sources, &project.ReaderLimits.MaxFiles, &project.ReaderLimits.MaxBytes, &policy, &created); err != nil {
		return Project{}, err
	}
	if err := json.Unmarshal([]byte(sources), &project.AllowedSourceRoots); err != nil {
		return Project{}, fault(CodeInternal, "decode project source roots", err)
	}
	if err := json.Unmarshal([]byte(policy), &project.Policy); err != nil {
		return Project{}, fault(CodeInternal, "decode project policy", err)
	}
	project.CreatedAt, _ = parseTime(created)
	return project, nil
}

func readProjectByIdentity(ctx context.Context, q queryer, projectID ProjectID, bundle string) (Project, bool, error) {
	row := q.QueryRowContext(ctx, `SELECT id, title, bundle_path, repository_path, allowed_sources_json, reader_max_files, reader_max_bytes, policy_json, created_at FROM projects WHERE id=? OR bundle_path=?`, projectID, bundle)
	project, err := scanProject(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Project{}, false, nil
	}
	if err != nil {
		return Project{}, false, fault(CodeInternal, "read existing project", err)
	}
	return project, true, nil
}

func sameProjectRegistration(left, right Project) bool {
	left.CreatedAt = right.CreatedAt
	l, _ := json.Marshal(left)
	r, _ := json.Marshal(right)
	return string(l) == string(r)
}

func uniqueSorted(values []string) []string {
	seen := map[string]bool{}
	result := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}
