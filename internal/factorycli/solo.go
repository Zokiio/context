package factorycli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Zokiio/context/internal/factory"
	"github.com/Zokiio/context/internal/factoryclient"
	"github.com/Zokiio/context/internal/factoryhttp"
	"github.com/Zokiio/context/internal/factoryworker"
	cli "github.com/urfave/cli/v3"
)

func decisionFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{Name: "decision-model", Usage: "Optional Ollama decision model that flags vague acceptance criteria, for example kev:0.8b-mxfp8"},
		&cli.StringFlag{Name: "decision-url", Value: "http://127.0.0.1:11434", Usage: "Ollama base URL for the decision model"},
		&cli.Float64Flag{Name: "decision-threshold", Value: factory.DefaultDecisionThreshold, Usage: "Flag criteria whose checkable probability is below this uncalibrated value"},
	}
}

func serviceOptions(c *cli.Command) factory.Options {
	return factory.Options{DecisionModel: c.String("decision-model"), DecisionURL: c.String("decision-url"), DecisionThreshold: c.Float64("decision-threshold")}
}

func agentCommand(stdout io.Writer) *cli.Command {
	return &cli.Command{Name: "agent", Usage: "Run a native coding agent for the implementation assignment on stdin and commit its candidate", DisableSliceFlagSeparator: true,
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "harness", Required: true, Usage: "claude or codex"},
			&cli.StringFlag{Name: "binary", Usage: "Agent executable; defaults to claude or codex on PATH"},
			&cli.StringFlag{Name: "model", Usage: "Model passed to the agent"},
			&cli.Float64Flag{Name: "max-budget-usd", Usage: "Claude Code spending limit for one attempt"},
			&cli.StringSliceFlag{Name: "allow-tool", Usage: "Claude Code tool permission, for example Bash(go test:*)"},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			return factoryworker.Agent(ctx, os.Stdin, stdout, factoryworker.AgentOptions{Harness: c.String("harness"), Binary: c.String("binary"), Model: c.String("model"), MaxBudgetUSD: c.Float64("max-budget-usd"), AllowedTools: c.StringSlice("allow-tool")})
		}}
}

// soloCommand runs one person's whole factory in one process: the service on a
// loopback listener plus an implementation worker and a verification worker. It
// creates the operator and worker identities on first start, keeps their tokens in
// the data directory, and grants the workers access to every registered project.
func soloCommand(stdout, stderr io.Writer) *cli.Command {
	return &cli.Command{Name: "solo", Usage: "Run the service and both workers for one person on this machine",
		Flags: append([]cli.Flag{
			&cli.StringFlag{Name: "data-dir", Required: true},
			&cli.StringFlag{Name: "listen", Value: "127.0.0.1:8741"},
			&cli.DurationFlag{Name: "poll", Value: 2 * time.Second},
			&cli.StringFlag{Name: "refiner-model", Value: "sonnet", Usage: "Model for the read-only Claude Code refiner"},
			&cli.Float64Flag{Name: "refiner-budget-usd", Value: 0.5, Usage: "Spending cap for one refinement"},
			&cli.BoolFlag{Name: "no-refiner", Usage: "Disable refinement"},
			&cli.StringFlag{Name: "expectation-model", Value: "sonnet", Usage: "Model for the Claude Code expectation author"},
			&cli.Float64Flag{Name: "expectation-budget-usd", Value: 1, Usage: "Spending cap for one expectation suite"},
			&cli.BoolFlag{Name: "no-expectations", Usage: "Disable the expectation author"},
			&cli.BoolFlag{Name: "no-pilot", Usage: "Do not run the autopilot, even for projects whose ask policy enables it"},
			&cli.DurationFlag{Name: "pilot-interval", Value: 5 * time.Second},
		}, append(append(decisionFlags(), refinerFlags()...), expectationFlags()...)...),
		Action: func(ctx context.Context, c *cli.Command) error {
			address := c.String("listen")
			if host, _, err := net.SplitHostPort(address); err != nil || !isLoopback(host) {
				return errors.New("solo mode listens on loopback only; use serve with TLS for shared access")
			}
			options := serviceOptions(c)
			options.Pilot, options.PilotInterval = !c.Bool("no-pilot"), c.Duration("pilot-interval")
			refiner, err := soloRefiner(c)
			if err != nil {
				return err
			}
			options.RefinerCommand = refiner
			if options.ExpectationCommand, err = soloExpectationAuthor(c); err != nil {
				return err
			}
			service, err := factory.Open(c.String("data-dir"), options)
			if err != nil {
				return err
			}
			defer service.Close()
			soloDir := filepath.Join(c.String("data-dir"), "solo")
			if err := os.MkdirAll(soloDir, 0o700); err != nil {
				return err
			}
			operator, operatorToken, err := soloOperator(ctx, service, soloDir)
			if err != nil {
				return err
			}
			workers := map[factory.Stage]string{}
			for _, stage := range []factory.Stage{factory.StageImplementation, factory.StageVerification} {
				token, err := soloWorker(ctx, service, operator, soloDir, stage)
				if err != nil {
					return err
				}
				workers[stage] = token
			}
			registerAgentExecutors(ctx, service, operator, stderr)

			listener, err := net.Listen("tcp", address)
			if err != nil {
				return err
			}
			server := &http.Server{Handler: factoryhttp.NewHandler(service), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
			serveErr := make(chan error, 1)
			go func() { serveErr <- server.Serve(listener) }()
			url := "http://" + listener.Addr().String()

			runCtx, stop := context.WithCancel(ctx)
			defer stop()
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				syncSoloGrants(runCtx, service, operator, c.Duration("poll"), stderr)
			}()
			for stage, token := range workers {
				client, err := factoryclient.New(url, token)
				if err != nil {
					return err
				}
				worker := &factoryworker.Worker{Client: client, Process: &factoryworker.Process{ReceiptDir: filepath.Join(soloDir, "receipts", string(stage))}, Stage: stage, HeartbeatInterval: 2 * time.Second, PollInterval: c.Duration("poll"), PermanentError: func(error) bool { return false }}
				wg.Add(1)
				go func(stage factory.Stage) {
					defer wg.Done()
					_ = worker.Loop(runCtx, func(r factoryworker.Receipt, e error) {
						if r.AttemptID != "" {
							fmt.Fprintf(stderr, "%s worker: attempt %s %s %s\n", stage, r.AttemptID, r.State, r.Candidate)
						}
						var api *factoryclient.APIError
						if e != nil && !(errors.As(e, &api) && api.Code == string(factory.CodeNoWork)) {
							fmt.Fprintf(stderr, "%s worker: %v\n", stage, e)
						}
					})
				}(stage)
			}
			fmt.Fprintf(stdout, "Waymark solo factory at %s\nOperator: %s (token file %s)\nOpen the URL and sign in with that token, or run:\n  ctx factory --url %s --token-file %s me\n", url, operator.Username, operatorToken, url, operatorToken)

			select {
			case err = <-serveErr:
			case <-ctx.Done():
				shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				err = server.Shutdown(shutdown)
			}
			stop()
			wg.Wait()
			if errors.Is(err, http.ErrServerClosed) {
				return nil
			}
			return err
		}}
}

// soloRefiner uses an explicit refiner command, or read-only Claude Code when it is
// installed.
func soloRefiner(c *cli.Command) ([]string, error) {
	if c.Bool("no-refiner") {
		return nil, nil
	}
	if argv, err := refinerCommandFile(c); err != nil || argv != nil {
		return argv, err
	}
	if _, err := exec.LookPath("claude"); err != nil {
		return nil, nil
	}
	self, err := os.Executable()
	if err != nil {
		return nil, err
	}
	return []string{self, "factory", "refiner", "--model", c.String("refiner-model"), "--max-budget-usd", fmt.Sprint(c.Float64("refiner-budget-usd"))}, nil
}

// soloExpectationAuthor uses an explicit command, or Claude Code when it is installed.
// The author may run tests and vet but nothing else in the shell.
func soloExpectationAuthor(c *cli.Command) ([]string, error) {
	if c.Bool("no-expectations") {
		return nil, nil
	}
	if argv, err := commandFile(c.String("expectation-command-file")); err != nil || argv != nil {
		return argv, err
	}
	if _, err := exec.LookPath("claude"); err != nil {
		return nil, nil
	}
	self, err := os.Executable()
	if err != nil {
		return nil, err
	}
	return []string{self, "factory", "expectation-author", "--model", c.String("expectation-model"), "--max-budget-usd", fmt.Sprint(c.Float64("expectation-budget-usd")), "--allow-tool", "Bash(go test:*)", "--allow-tool", "Bash(go vet:*)"}, nil
}

func isLoopback(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func soloOperator(ctx context.Context, service *factory.Service, dir string) (factory.Principal, string, error) {
	path := filepath.Join(dir, "operator.token")
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		result, err := service.Bootstrap(ctx, factory.BootstrapRequest{Username: "operator"})
		if err != nil {
			return factory.Principal{}, "", fmt.Errorf("bootstrap solo operator: %w (an installation set up with init must keep using serve)", err)
		}
		if err := writeSoloSecret(path, result.Token); err != nil {
			return factory.Principal{}, "", err
		}
	}
	token, err := readSecret(path)
	if err != nil {
		return factory.Principal{}, "", err
	}
	principal, err := service.Authenticate(ctx, token)
	return principal, path, err
}

func soloWorker(ctx context.Context, service *factory.Service, operator factory.Principal, dir string, stage factory.Stage) (string, error) {
	name := "solo-" + string(stage)
	path := filepath.Join(dir, name+".token")
	if token, err := readSecret(path); err == nil {
		return token, nil
	}
	user, err := service.CreateUser(ctx, operator, factory.CreateUserRequest{RequestID: "solo-user-" + name, Username: name, Kind: factory.UserKindWorker})
	if err != nil {
		return "", err
	}
	issued, err := service.IssueToken(ctx, operator, factory.IssueTokenRequest{RequestID: fmt.Sprintf("solo-token-%s-%d", name, time.Now().UnixNano()), UserID: user.ID, Label: "solo " + string(stage) + " worker"})
	if err != nil {
		return "", err
	}
	return issued.Token, writeSoloSecret(path, issued.Token)
}

func writeSoloSecret(path, token string) error {
	f, err := reserveSecret(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return storeSecret(f, token)
}

// syncSoloGrants keeps both solo workers enrolled in every registered project, so a
// project registered while solo mode runs is picked up without a restart.
func syncSoloGrants(ctx context.Context, service *factory.Service, operator factory.Principal, interval time.Duration, stderr io.Writer) {
	capabilities := map[string][]factory.Capability{
		"solo-implementation": {factory.CapabilityRead, factory.CapabilityImplementWorker},
		"solo-verification":   {factory.CapabilityRead, factory.CapabilityVerifyWorker, factory.CapabilityVerify},
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		users, err := service.ListUsers(ctx, operator)
		projects, perr := service.ListProjects(ctx, operator)
		if err == nil && perr == nil {
			for _, project := range projects {
				for _, user := range users {
					caps, ok := capabilities[user.Username]
					if !ok {
						continue
					}
					request := factory.GrantProjectRequest{RequestID: "solo-grant-" + string(project.ID) + "-" + user.Username, ProjectID: project.ID, UserID: user.ID, Capabilities: caps}
					if _, err := service.GrantProject(ctx, operator, request); err != nil && ctx.Err() == nil {
						fmt.Fprintf(stderr, "solo: grant %s on %s: %v\n", user.Username, project.ID, err)
					}
				}
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// registerAgentExecutors offers Claude Code and Codex executor profiles when their
// CLIs are installed. A project still has to allow a profile before it can run.
func registerAgentExecutors(ctx context.Context, service *factory.Service, operator factory.Principal, stderr io.Writer) {
	self, err := os.Executable()
	if err != nil {
		return
	}
	for _, harness := range []string{"claude", "codex"} {
		if _, err := exec.LookPath(harness); err != nil {
			continue
		}
		key := harness + "-agent"
		command := []string{self, "factory", "agent", "--harness", harness}
		if harness == "claude" {
			command = append(command, "--max-budget-usd", "2")
		}
		request := factory.RegisterExecutorRequest{RequestID: "solo-executor-" + key, Key: key, DisplayName: strings.ToUpper(harness[:1]) + harness[1:] + " agent", Command: command}
		if _, err := service.RegisterExecutor(ctx, operator, request); err != nil {
			var fault *factory.Error
			if !errors.As(err, &fault) || fault.Code != factory.CodeConflict {
				fmt.Fprintf(stderr, "solo: register %s: %v\n", key, err)
			}
		}
	}
}
