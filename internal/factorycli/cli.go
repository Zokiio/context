// Package factorycli exposes the shared factory service through ctx factory.
package factorycli

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Zokiio/context/internal/factory"
	"github.com/Zokiio/context/internal/factoryclient"
	"github.com/Zokiio/context/internal/factoryworker"
	cli "github.com/urfave/cli/v3"
)

// Command adds an opt-in factory surface. Reader commands never open a service.
func Command(stdout, stderr io.Writer) *cli.Command {
	root := &cli.Command{Name: "factory", Usage: "Operate the Waymark factory service", Writer: stderr, ErrWriter: stderr,
		Flags: []cli.Flag{&cli.StringFlag{Name: "url", Value: "http://127.0.0.1:8741", Usage: "Factory service URL"}, &cli.StringFlag{Name: "token-file", Usage: "Private file containing the bearer token"}, &cli.BoolFlag{Name: "json", Usage: "Write JSON results (the default)"}},
		Action: func(context.Context, *cli.Command) error {
			return errors.New("expected a factory subcommand; see ctx factory --help")
		}}
	root.Commands = []*cli.Command{
		initCommand(stdout), serveCommand(stderr), soloCommand(stdout, stderr), agentCommand(stdout), refineCommand(stdout), refinerCommand(stdout, stderr), expectationsCommand(stdout), expectationAuthorCommand(stdout, stderr),
		{Name: "me", Usage: "Inspect the authenticated identity", Action: request(stdout, http.MethodGet, "/v1/me", nil)},
		usersCommand(stdout), projectsCommand(stdout), profilesCommand(stdout, false), profilesCommand(stdout, true), jobsCommand(stdout), workerCommand(stdout, stderr),
		{Name: "fixture", Usage: "Run the offline fixture implementation from assignment stdin", Hidden: true, Flags: []cli.Flag{&cli.DurationFlag{Name: "delay"}, &cli.BoolFlag{Name: "fail"}}, Action: func(ctx context.Context, c *cli.Command) error {
			return factoryworker.Fixture(ctx, os.Stdin, stdout, c.Duration("delay"), c.Bool("fail"))
		}},
		{Name: "fixture-check", Usage: "Verify a real offline fixture candidate from assignment stdin", Hidden: true, Action: func(_ context.Context, _ *cli.Command) error { return factoryworker.FixtureCheck(os.Stdin, stdout) }},
	}
	return root
}
func usersCommand(out io.Writer) *cli.Command {
	return &cli.Command{Name: "users", Usage: "Manage installation identities and credentials", Commands: []*cli.Command{
		{Name: "list", Usage: "List users", Action: request(out, http.MethodGet, "/v1/users", nil)},
		{Name: "create", Usage: "Create a distinct human or worker user", Flags: append(requestFlags(), &cli.StringFlag{Name: "username", Required: true}, &cli.StringFlag{Name: "kind", Value: "human"}, &cli.BoolFlag{Name: "admin"}), Action: request(out, http.MethodPost, "/v1/users", func(c *cli.Command) (any, error) {
			return factory.CreateUserRequest{RequestID: reqID(c), Username: c.String("username"), Kind: factory.UserKind(c.String("kind")), InstallationAdmin: c.Bool("admin")}, nil
		})},
		{Name: "token", Usage: "Issue a token into a new private file", Flags: append(requestFlags(), &cli.StringFlag{Name: "user", Required: true}, &cli.StringFlag{Name: "out", Required: true}, &cli.StringFlag{Name: "label", Value: "ctx"}, &cli.StringFlag{Name: "expires"}), Action: func(ctx context.Context, c *cli.Command) error {
			f, e := reserveSecret(c.String("out"))
			if e != nil {
				return e
			}
			defer f.Close()
			ok := false
			defer func() {
				if !ok {
					os.Remove(c.String("out"))
				}
			}()
			client, e := client(c)
			if e != nil {
				return e
			}
			var expiry *time.Time
			if c.String("expires") != "" {
				v, e := parseTime(c.String("expires"))
				if e != nil {
					return e
				}
				expiry = &v
			}
			input := factory.IssueTokenRequest{RequestID: reqID(c), UserID: factory.UserID(c.String("user")), Label: c.String("label"), ExpiresAt: expiry}
			var result factory.IssuedToken
			if e = client.DoJSON(ctx, http.MethodPost, "/v1/users/"+esc(c.String("user"))+"/tokens", input, &result); e != nil {
				return e
			}
			if e = storeSecret(f, result.Token); e != nil {
				return e
			}
			ok = true
			result.Token = ""
			return encode(out, map[string]any{"credential": result, "tokenFile": c.String("out")})
		}},
		{Name: "disable", Usage: "Disable or enable a user", Flags: append(requestFlags(), &cli.StringFlag{Name: "user", Required: true}, &cli.BoolFlag{Name: "enabled"}), Action: func(ctx context.Context, c *cli.Command) error {
			return call(ctx, c, out, http.MethodPut, "/v1/users/"+esc(c.String("user"))+"/disabled", factory.SetUserDisabledRequest{RequestID: reqID(c), UserID: factory.UserID(c.String("user")), Disabled: !c.Bool("enabled")})
		}},
		{Name: "revoke-token", Usage: "Revoke a token by its nonsecret credential ID", Flags: append(requestFlags(), &cli.StringFlag{Name: "token-id", Required: true}), Action: func(ctx context.Context, c *cli.Command) error {
			return call(ctx, c, out, http.MethodPost, "/v1/tokens/"+esc(c.String("token-id"))+"/revoke", factory.RevokeTokenRequest{RequestID: reqID(c), TokenID: factory.ID(c.String("token-id"))})
		}},
	}}
}
func projectsCommand(out io.Writer) *cli.Command {
	return &cli.Command{Name: "projects", Usage: "Register existing projects and administer access", Commands: []*cli.Command{
		{Name: "list", Usage: "List visible projects", Action: request(out, http.MethodGet, "/v1/projects", nil)},
		{Name: "register", Usage: "Register an existing OKF bundle, repository, and admin-owned execution policy", DisableSliceFlagSeparator: true, Flags: append(requestFlags(), &cli.StringFlag{Name: "bundle", Required: true}, &cli.StringFlag{Name: "repo", Required: true}, &cli.StringSliceFlag{Name: "allow-source"}, &cli.StringSliceFlag{Name: "executor"}, &cli.StringSliceFlag{Name: "required-check"}, &cli.IntFlag{Name: "max-repairs", Value: 1}, &cli.DurationFlag{Name: "max-duration", Value: time.Hour}, &cli.IntFlag{Name: "max-files"}, &cli.Int64Flag{Name: "max-bytes"}, &cli.BoolFlag{Name: "fixture"}, &cli.StringSliceFlag{Name: "protect", Usage: "Repository path pattern an implementation may not change, such as tests/ or **/*_test.go"}, &cli.BoolFlag{Name: "solo-review", Usage: "Let the person who submits a job also review it"}, &cli.BoolFlag{Name: "require-expectations", Usage: "Admit jobs only from accepted holdout expectations"}), Action: request(out, http.MethodPost, "/v1/projects", func(c *cli.Command) (any, error) {
			return factory.RegisterProjectRequest{RequestID: reqID(c), BundlePath: c.String("bundle"), RepositoryPath: c.String("repo"), AllowedSourceRoots: c.StringSlice("allow-source"), ReaderLimits: factory.ReaderLimits{MaxFiles: c.Int("max-files"), MaxBytes: c.Int64("max-bytes")}, Policy: factory.ProjectPolicy{RequiredChecks: c.StringSlice("required-check"), AllowedExecutors: c.StringSlice("executor"), MaxRepairAttempts: c.Int("max-repairs"), MaxJobDurationSeconds: int64(c.Duration("max-duration") / time.Second), Fixture: c.Bool("fixture"), ProtectedPaths: c.StringSlice("protect"), AllowCoordinatorReview: c.Bool("solo-review"), RequireExpectations: c.Bool("require-expectations")}}, nil
		})},
		{Name: "inspect", Usage: "Inspect project records, readiness, and policy", Flags: projectFlags(), Action: func(ctx context.Context, c *cli.Command) error {
			return inspectProject(ctx, c, out)
		}},
		{Name: "work-items", Usage: "Inspect the authoritative work-item inventory and readiness", Flags: projectFlags(), Action: func(ctx context.Context, c *cli.Command) error {
			return inspectProject(ctx, c, out)
		}},
		{Name: "metrics", Usage: "Measured first-pass rate, lead times, human decisions and agent cost", Flags: projectFlags(), Action: func(ctx context.Context, c *cli.Command) error {
			return call(ctx, c, out, http.MethodGet, "/v1/projects/"+esc(c.String("project"))+"/metrics", nil)
		}},
		{Name: "grants", Usage: "List project access", Flags: projectFlags(), Action: func(ctx context.Context, c *cli.Command) error {
			return call(ctx, c, out, http.MethodGet, "/v1/projects/"+esc(c.String("project"))+"/grants", nil)
		}},
		{Name: "grant", Usage: "Replace a user's project capabilities", DisableSliceFlagSeparator: true, Flags: append(requestFlags(), &cli.StringFlag{Name: "project", Required: true}, &cli.StringFlag{Name: "user", Required: true}, &cli.StringSliceFlag{Name: "capability"}), Action: func(ctx context.Context, c *cli.Command) error {
			caps := []factory.Capability{}
			for _, v := range c.StringSlice("capability") {
				caps = append(caps, factory.Capability(v))
			}
			r := factory.GrantProjectRequest{RequestID: reqID(c), ProjectID: factory.ProjectID(c.String("project")), UserID: factory.UserID(c.String("user")), Capabilities: caps}
			return call(ctx, c, out, http.MethodPut, "/v1/projects/"+esc(c.String("project"))+"/grants/"+esc(c.String("user")), r)
		}},
	}}
}
func profilesCommand(out io.Writer, check bool) *cli.Command {
	name := "executors"
	if check {
		name = "checks"
	}
	endpoint := "/v1/" + name
	flags := append(requestFlags(), &cli.StringFlag{Name: "key", Required: true}, &cli.StringFlag{Name: "name"}, &cli.StringFlag{Name: "program"}, &cli.StringSliceFlag{Name: "arg"}, &cli.StringFlag{Name: "command-file", Usage: "JSON argv array, mutually exclusive with program and arg"}, &cli.BoolFlag{Name: "fixture", Usage: "Restrict this profile to fixture projects"})
	if check {
		flags = append(flags, &cli.StringFlag{Name: "environment", Value: "local"})
	}
	return &cli.Command{Name: name, Usage: "Manage administrator-owned command profiles", Commands: []*cli.Command{
		{Name: "list", Action: request(out, http.MethodGet, endpoint, nil)},
		{Name: "register", Usage: "Register a literal argument array", DisableSliceFlagSeparator: true, Flags: flags, Action: request(out, http.MethodPost, endpoint, func(c *cli.Command) (any, error) {
			argv, e := commandArgv(c, check)
			if e != nil {
				return nil, e
			}
			if check {
				return factory.RegisterCheckRequest{RequestID: reqID(c), Key: c.String("key"), DisplayName: profileName(c), Command: argv, Environment: c.String("environment"), Fixture: c.Bool("fixture")}, nil
			}
			return factory.RegisterExecutorRequest{RequestID: reqID(c), Key: c.String("key"), DisplayName: profileName(c), Command: argv, Fixture: c.Bool("fixture")}, nil
		})},
	}}
}
func commandArgv(c *cli.Command, check bool) ([]string, error) {
	if path := c.String("command-file"); path != "" {
		if c.String("program") != "" || len(c.StringSlice("arg")) > 0 {
			return nil, errors.New("command-file cannot be combined with program or arg")
		}
		b, e := os.ReadFile(path)
		if e != nil {
			return nil, e
		}
		var argv []string
		if e = json.Unmarshal(b, &argv); e != nil {
			return nil, e
		}
		if len(argv) == 0 || argv[0] == "" {
			return nil, errors.New("command-file must contain a nonempty argv")
		}
		return argv, nil
	}
	if program := c.String("program"); program != "" {
		return append([]string{program}, c.StringSlice("arg")...), nil
	}
	if c.Bool("fixture") {
		exe, e := os.Executable()
		if e != nil {
			return nil, e
		}
		sub := "fixture"
		if check {
			sub = "fixture-check"
		}
		return []string{exe, "factory", sub}, nil
	}
	return nil, errors.New("program or command-file is required")
}
func jobsCommand(out io.Writer) *cli.Command {
	commands := []*cli.Command{
		{Name: "submit", Usage: "Admit one ready work item under project policy", Flags: append(requestFlags(), &cli.StringFlag{Name: "project", Required: true}, &cli.StringFlag{Name: "work-item", Required: true}, &cli.StringFlag{Name: "base", Usage: "Base revision; the expectations commit when --expectations is given"}, &cli.StringFlag{Name: "expectations", Usage: "Accepted holdout expectations to start from"}, &cli.StringFlag{Name: "executor", Required: true}, &cli.IntFlag{Name: "repair-limit"}, &cli.StringFlag{Name: "deadline", Required: true}), Action: request(out, http.MethodPost, "/v1/jobs", func(c *cli.Command) (any, error) {
			deadline, e := parseTime(c.String("deadline"))
			if e != nil {
				return nil, e
			}
			return factory.SubmitRequest{RequestID: reqID(c), ProjectID: factory.ProjectID(c.String("project")), WorkItemID: c.String("work-item"), BaseRevision: c.String("base"), ExecutorKey: c.String("executor"), RepairLimit: c.Int("repair-limit"), Deadline: deadline, ExpectationsID: factory.ID(c.String("expectations"))}, nil
		})},
		{Name: "list", Usage: "Inspect the visible job queue", Flags: []cli.Flag{&cli.StringFlag{Name: "project"}, &cli.StringFlag{Name: "status"}}, Action: func(ctx context.Context, c *cli.Command) error {
			q := url.Values{}
			if c.String("project") != "" {
				q.Set("projectId", c.String("project"))
			}
			if c.String("status") != "" {
				q.Set("status", c.String("status"))
			}
			path := "/v1/jobs"
			if len(q) > 0 {
				path += "?" + q.Encode()
			}
			return call(ctx, c, out, http.MethodGet, path, nil)
		}},
		{Name: "show", Usage: "Inspect requirements, attempts, events, evidence, and remaining actions", Flags: jobFlags(), Action: func(ctx context.Context, c *cli.Command) error {
			id, e := jobID(c)
			if e != nil {
				return e
			}
			return call(ctx, c, out, http.MethodGet, "/v1/jobs/"+esc(id), nil)
		}},
		{Name: "watch", Usage: "Print each changed job version until accepted or a blocking outcome", Flags: append(jobFlags(), &cli.DurationFlag{Name: "interval", Value: time.Second}, &cli.BoolFlag{Name: "until-idle"}), Action: func(ctx context.Context, c *cli.Command) error { return watch(ctx, c, out) }},
	}
	for _, kind := range []factory.ActionKind{factory.ActionCancel, factory.ActionRepair, factory.ActionReconcile, factory.ActionResolve} {
		kind := kind
		name := string(kind)
		if kind == factory.ActionResolve {
			name = "resolve"
		}
		flags := append(jobMutationFlags(), &cli.StringFlag{Name: "attempt"}, &cli.StringFlag{Name: "resolution"}, &cli.StringFlag{Name: "evidence"})
		commands = append(commands, &cli.Command{Name: name, Usage: "Request " + string(kind) + " through service policy", Flags: flags, Action: func(ctx context.Context, c *cli.Command) error {
			id, e := jobID(c)
			if e != nil {
				return e
			}
			var attempt *factory.AttemptID
			if c.String("attempt") != "" {
				a := factory.AttemptID(c.String("attempt"))
				attempt = &a
			}
			r := factory.JobActionRequest{RequestID: reqID(c), JobID: factory.JobID(id), IfVersion: c.Int64("if-version"), Action: kind, AttemptID: attempt, Resolution: factory.Resolution(c.String("resolution")), Evidence: c.String("evidence")}
			return call(ctx, c, out, http.MethodPost, "/v1/jobs/"+esc(id)+"/actions", r)
		}})
	}
	for _, review := range []bool{true, false} {
		review := review
		name := "accept"
		if review {
			name = "review"
		}
		flags := append(jobMutationFlags(), &cli.StringFlag{Name: "candidate", Required: true}, &cli.StringFlag{Name: "requirements", Required: true}, &cli.StringFlag{Name: "summary"})
		if review {
			flags = append(flags, &cli.StringFlag{Name: "decision", Value: "approved"})
		}
		commands = append(commands, &cli.Command{Name: name, Usage: "Record an authorized " + name + " decision for exact inputs", Flags: flags, Action: func(ctx context.Context, c *cli.Command) error {
			id, e := jobID(c)
			if e != nil {
				return e
			}
			if review {
				r := factory.ReviewRequest{RequestID: reqID(c), JobID: factory.JobID(id), IfVersion: c.Int64("if-version"), CandidateRevision: c.String("candidate"), RequirementFingerprint: c.String("requirements"), Decision: factory.ReviewDecision(c.String("decision")), Summary: c.String("summary")}
				return call(ctx, c, out, http.MethodPost, "/v1/jobs/"+esc(id)+"/reviews", r)
			}
			r := factory.AcceptRequest{RequestID: reqID(c), JobID: factory.JobID(id), IfVersion: c.Int64("if-version"), CandidateRevision: c.String("candidate"), RequirementFingerprint: c.String("requirements"), Summary: c.String("summary")}
			return call(ctx, c, out, http.MethodPost, "/v1/jobs/"+esc(id)+"/acceptance", r)
		}})
	}
	commands = append(commands, &cli.Command{Name: "merge", Usage: "Merge an accepted candidate into a branch after human approval", Flags: append(jobMutationFlags(), &cli.StringFlag{Name: "candidate", Required: true}, &cli.StringFlag{Name: "target-branch", Usage: "Defaults to the branch checked out in the project repository"}), Action: func(ctx context.Context, c *cli.Command) error {
		id, e := jobID(c)
		if e != nil {
			return e
		}
		r := factory.MergeRequest{RequestID: reqID(c), JobID: factory.JobID(id), IfVersion: c.Int64("if-version"), CandidateRevision: c.String("candidate"), TargetBranch: c.String("target-branch")}
		return call(ctx, c, out, http.MethodPost, "/v1/jobs/"+esc(id)+"/merge", r)
	}})
	return &cli.Command{Name: "jobs", Usage: "Submit, inspect, and coordinate factory jobs", Commands: commands}
}
func workerCommand(out, stderr io.Writer) *cli.Command {
	return &cli.Command{Name: "worker", Usage: "Run an enrolled external-process worker", Flags: []cli.Flag{&cli.StringFlag{Name: "role", Required: true, Usage: "implementation or verification"}, &cli.StringFlag{Name: "receipts", Required: true, Usage: "Durable private local receipt directory"}, &cli.StringFlag{Name: "project"}, &cli.BoolFlag{Name: "once"}, &cli.DurationFlag{Name: "poll", Value: time.Second}, &cli.DurationFlag{Name: "heartbeat", Value: 2 * time.Second}}, Action: func(ctx context.Context, c *cli.Command) error {
		client, e := client(c)
		if e != nil {
			return e
		}
		stage := factory.Stage(c.String("role"))
		if stage != factory.StageImplementation && stage != factory.StageVerification {
			return errors.New("role must be implementation or verification")
		}
		var project *factory.ProjectID
		if c.String("project") != "" {
			v := factory.ProjectID(c.String("project"))
			project = &v
		}
		w := &factoryworker.Worker{Client: client, Process: &factoryworker.Process{ReceiptDir: c.String("receipts")}, Stage: stage, ProjectID: project, HeartbeatInterval: c.Duration("heartbeat"), PollInterval: c.Duration("poll"), PermanentError: func(err error) bool {
			var api *factoryclient.APIError
			return errors.As(err, &api) && api.Status < 500 && api.Code != string(factory.CodeNoWork)
		}}
		if c.Bool("once") {
			r, e := w.Once(ctx)
			if r.AttemptID != "" {
				if enc := encode(out, r); enc != nil {
					return enc
				}
			}
			if e != nil {
				return e
			}
			if r.State != "succeeded" {
				return fmt.Errorf("worker outcome %s", r.State)
			}
			return nil
		}
		return w.Loop(ctx, func(r factoryworker.Receipt, e error) {
			if r.AttemptID != "" {
				_ = encode(out, r)
			}
			if e != nil {
				var api *factoryclient.APIError
				if errors.As(e, &api) && api.Code == string(factory.CodeNoWork) {
					return
				}
				fmt.Fprintln(stderr, e)
			}
		})
	}}
}
func watch(ctx context.Context, c *cli.Command, out io.Writer) error {
	id, e := jobID(c)
	if e != nil {
		return e
	}
	cl, e := client(c)
	if e != nil {
		return e
	}
	interval := c.Duration("interval")
	if interval <= 0 {
		return errors.New("watch interval must be positive")
	}
	var version int64 = -1
	for {
		var job factory.Job
		if e = cl.DoJSON(ctx, http.MethodGet, "/v1/jobs/"+esc(id), nil, &job); e != nil {
			return e
		}
		if job.Version != version {
			if e = encode(out, job); e != nil {
				return e
			}
			version = job.Version
		}
		switch job.Status {
		case factory.JobAccepted:
			return nil
		case factory.JobFailed, factory.JobCancelled, factory.JobCancellationUnknown, factory.JobImplementationUnknown, factory.JobStale, factory.JobVerificationFailed:
			return fmt.Errorf("job requires action: %s", job.Status)
		case factory.JobReadyForReview, factory.JobReadyForAcceptance:
			if c.Bool("until-idle") {
				return nil
			}
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}
func request(out io.Writer, method, path string, input func(*cli.Command) (any, error)) cli.ActionFunc {
	return func(ctx context.Context, c *cli.Command) error {
		if c.NArg() != 0 {
			return errors.New("command does not accept positional arguments")
		}
		var value any
		var e error
		if input != nil {
			value, e = input(c)
			if e != nil {
				return e
			}
		}
		return call(ctx, c, out, method, path, value)
	}
}
func call(ctx context.Context, c *cli.Command, out io.Writer, method, path string, input any) error {
	cl, e := client(c)
	if e != nil {
		return e
	}
	var result json.RawMessage
	if e = cl.DoJSON(ctx, method, path, input, &result); e != nil {
		return e
	}
	if len(result) == 0 {
		return nil
	}
	return encode(out, result)
}
func client(c *cli.Command) (*factoryclient.Client, error) {
	token, e := readSecret(c.String("token-file"))
	if e != nil {
		return nil, e
	}
	return factoryclient.New(c.String("url"), token)
}
func encode(out io.Writer, v any) error { return json.NewEncoder(out).Encode(v) }
func requestFlags() []cli.Flag {
	return []cli.Flag{&cli.StringFlag{Name: "request-id", Usage: "Stable idempotency ID; generated when omitted"}}
}
func projectFlags() []cli.Flag { return []cli.Flag{&cli.StringFlag{Name: "project", Required: true}} }
func jobFlags() []cli.Flag {
	return []cli.Flag{&cli.StringFlag{Name: "job", Usage: "Job ID, alternatively the sole positional argument"}}
}
func jobMutationFlags() []cli.Flag {
	return append(append(requestFlags(), jobFlags()...), &cli.Int64Flag{Name: "if-version", Required: true})
}
func jobID(c *cli.Command) (string, error) {
	if c.NArg() > 1 || c.NArg() == 1 && c.String("job") != "" {
		return "", errors.New("supply one job ID")
	}
	id := c.String("job")
	if c.NArg() == 1 {
		id = c.Args().First()
	}
	if id == "" {
		return "", errors.New("job ID is required")
	}
	return id, nil
}
func reqID(c *cli.Command) string {
	if s := c.String("request-id"); s != "" {
		return s
	}
	var b [16]byte
	if _, e := rand.Read(b[:]); e != nil {
		panic(e)
	}
	return hex.EncodeToString(b[:])
}
func esc(s string) string { return url.PathEscape(s) }
func parseTime(s string) (time.Time, error) {
	v, e := time.Parse(time.RFC3339, s)
	if e != nil {
		return v, errors.New("timestamp must use RFC3339, for example 2026-10-23T17:00:00Z")
	}
	return v, nil
}
func readSecret(path string) (string, error) {
	if path == "" {
		return "", errors.New("--token-file is required")
	}
	info, e := os.Lstat(path)
	if e != nil {
		return "", fmt.Errorf("read token file: %w", e)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		return "", errors.New("token file must be a regular private file with mode 0600 or 0400")
	}
	if info.Size() > 8192 {
		return "", errors.New("token file is too large")
	}
	b, e := os.ReadFile(path)
	if e != nil {
		return "", e
	}
	s := strings.TrimSpace(string(b))
	if s == "" {
		return "", errors.New("token file is empty")
	}
	return s, nil
}
func reserveSecret(path string) (*os.File, error) {
	if path == "" {
		return nil, errors.New("token output path is required")
	}
	return os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
}
func storeSecret(f *os.File, token string) error {
	if _, e := io.WriteString(f, token+"\n"); e != nil {
		return e
	}
	return f.Sync()
}

func inspectProject(ctx context.Context, c *cli.Command, out io.Writer) error {
	cl, e := client(c)
	if e != nil {
		return e
	}
	var result factory.ProjectOrientation
	if e = cl.DoJSON(ctx, http.MethodGet, "/v1/projects/"+esc(c.String("project"))+"/orientation", nil, &result); e != nil {
		return e
	}
	if e = encode(out, result); e != nil {
		return e
	}
	if !result.Result.Complete || !result.Result.InventoryComplete {
		return errors.New("project inspection is incomplete; see result diagnostics")
	}
	return nil
}

func profileName(c *cli.Command) string {
	if name := c.String("name"); name != "" {
		return name
	}
	return c.String("key")
}
