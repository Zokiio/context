package discovery

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type setupProcessEvent struct {
	Phase string
	Path  string
	Error string
}

// Separate processes use real filesystem locks. Events let the parent hold the
// first rename until the second writer either waits for a shared lock or also
// reaches rename, reproducing the unsafe interleaving without timing sleeps.
func TestSetupConcurrentProcessWriter(t *testing.T) {
	role := os.Getenv("CTX_SETUP_LOCK_ROLE")
	if role == "" {
		t.Skip("subprocess helper")
	}
	root := os.Getenv("CTX_SETUP_LOCK_ROOT")
	request := SetupRequest{Cwd: filepath.Join(root, "checkout"), Home: filepath.Join(root, "home"), Records: "../records"}
	if role == "personal" {
		request.Personal = true
		request.AllowSources = []string{"."}
	} else if role == "shared" {
		request.Home = filepath.Join(root, "home-link")
	} else if role == "right" {
		request.Cwd, request.Home = request.Home, request.Cwd
	}
	plan := setupWritePlan(t, request)
	output := json.NewEncoder(os.Stdout)
	emit := func(event setupProcessEvent) {
		if err := output.Encode(event); err != nil {
			t.Fatal(err)
		}
	}
	ops := defaultSetupWriteOps()
	lock := ops.lock
	firstLock := true
	ops.lock = func(path string) (func(), error) {
		emit(setupProcessEvent{Phase: "lock", Path: path})
		unlock, err := lock(path)
		if err == nil && firstLock && (role == "left" || role == "right") {
			firstLock = false
			emit(setupProcessEvent{Phase: "held", Path: path})
			if _, err := bufio.NewReader(os.Stdin).ReadString('\n'); err != nil {
				unlock()
				return nil, err
			}
		}
		return unlock, err
	}
	ops.rename = func(from, to string) error {
		emit(setupProcessEvent{Phase: "rename"})
		if _, err := bufio.NewReader(os.Stdin).ReadString('\n'); err != nil {
			return err
		}
		return os.Rename(from, to)
	}
	err := plan.apply(context.Background(), ops)
	result := setupProcessEvent{Phase: "result"}
	if err != nil {
		result.Error = err.Error()
	}
	emit(result)
}

type setupWriterProcess struct {
	input  io.WriteCloser
	events chan setupProcessEvent
	done   chan error
	closed chan struct{}
	output bytes.Buffer
}

func startSetupWriter(t *testing.T, root, role string) *setupWriterProcess {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestSetupConcurrentProcessWriter$")
	command.Env = append(os.Environ(), "CTX_SETUP_LOCK_ROLE="+role, "CTX_SETUP_LOCK_ROOT="+root)
	writer := &setupWriterProcess{events: make(chan setupProcessEvent, 8), done: make(chan error, 1), closed: make(chan struct{})}
	var err error
	writer.input, err = command.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	output, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	command.Stderr = &writer.output
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cancel()
		writer.input.Close()
		<-writer.closed
	})
	go func() {
		scanner := bufio.NewScanner(output)
		for scanner.Scan() {
			var event setupProcessEvent
			if json.Unmarshal(scanner.Bytes(), &event) == nil && event.Phase != "" {
				writer.events <- event
			}
		}
		err := command.Wait()
		if scanner.Err() != nil {
			err = scanner.Err()
		}
		writer.done <- err
		close(writer.events)
		close(writer.closed)
	}()
	return writer
}

func TestSetupOrdersLocksAcrossDifferentHomes(t *testing.T) {
	request, _ := setupWriteFixture(t, false)
	root := filepath.Dir(request.Home)
	first := startSetupWriter(t, root, "left")
	firstLock := first.next(t)
	if event := first.next(t); event.Phase != "held" {
		t.Fatalf("first writer did not acquire its lock: %#v", event)
	}
	second := startSetupWriter(t, root, "right")
	secondLock := second.next(t)
	if !SamePath(firstLock.Path, secondLock.Path) {
		if event := second.next(t); event.Phase != "held" {
			t.Fatalf("second writer did not acquire its lock: %#v", event)
		}
		first.release(t)
		second.release(t)
		firstWait, secondWait := first.next(t), second.next(t)
		if firstWait.Phase == "lock" && secondWait.Phase == "lock" && SamePath(firstWait.Path, secondLock.Path) && SamePath(secondWait.Path, firstLock.Path) {
			t.Fatal("writers hold one another's next lock and cannot make progress")
		}
		t.Fatalf("writers took inconsistent lock order: %#v, %#v", firstWait, secondWait)
	}
	first.release(t)
	for event := first.next(t); event.Phase != "rename"; event = first.next(t) {
		if event.Phase != "lock" {
			t.Fatalf("first writer failed: %#v", event)
		}
	}
	first.release(t)
	if result := first.next(t); result.Phase != "result" || result.Error != "" {
		t.Fatalf("first writer failed: %#v", result)
	}
	if event := second.next(t); event.Phase != "held" {
		t.Fatalf("second writer did not acquire the released lock: %#v", event)
	}
	second.release(t)
	for event := second.next(t); ; event = second.next(t) {
		if event.Phase == "rename" {
			second.release(t)
		}
		if event.Phase == "result" {
			// The first shared config is the second writer's personal registry.
			// That profile conflict must fail after the locks become available.
			if event.Error == "" {
				t.Fatal("second writer ignored the newly conflicting profile")
			}
			break
		}
	}
	for _, process := range []*setupWriterProcess{first, second} {
		if err := <-process.done; err != nil {
			t.Fatalf("writer process failed: %v\n%s", err, process.output.String())
		}
	}
}

func (p *setupWriterProcess) next(t *testing.T) setupProcessEvent {
	t.Helper()
	event, ok := <-p.events
	if !ok {
		t.Fatalf("writer stopped before expected event: %v\n%s", <-p.done, p.output.String())
	}
	return event
}

func (p *setupWriterProcess) release(t *testing.T) {
	t.Helper()
	if _, err := fmt.Fprintln(p.input); err != nil {
		t.Fatal(err)
	}
}

func TestSetupSerializesSharedAndPersonalProcesses(t *testing.T) {
	for _, firstRole := range []string{"shared", "personal"} {
		t.Run(firstRole+" first", func(t *testing.T) {
			request, _ := setupWriteFixture(t, false)
			root := filepath.Dir(request.Home)
			if err := os.Symlink(request.Home, filepath.Join(root, "home-link")); err != nil {
				t.Fatal(err)
			}
			first := startSetupWriter(t, root, firstRole)
			var held []string
			for event := first.next(t); event.Phase != "rename"; event = first.next(t) {
				if event.Phase != "lock" {
					t.Fatalf("first writer failed before rename: %#v", event)
				}
				held = append(held, event.Path)
			}
			secondRole := "shared"
			if firstRole == secondRole {
				secondRole = "personal"
			}
			second := startSetupWriter(t, root, secondRole)
			released := false
			var result setupProcessEvent
			for result = second.next(t); result.Phase != "result"; result = second.next(t) {
				if result.Phase == "lock" && !released {
					for _, path := range held {
						if SamePath(path, result.Path) {
							first.release(t)
							released = true
							break
						}
					}
				}
				if result.Phase == "rename" {
					if !released {
						first.release(t)
						released = true
					}
					second.release(t)
				}
			}
			if !released {
				first.release(t)
			}
			firstResult := first.next(t)
			for _, process := range []*setupWriterProcess{first, second} {
				if err := <-process.done; err != nil {
					t.Fatalf("writer process failed: %v\n%s", err, process.output.String())
				}
			}
			if firstResult.Phase != "result" || firstResult.Error != "" {
				t.Fatalf("first writer failed: %#v", firstResult)
			}
			scope, err := Resolve(context.Background(), Request{Cwd: request.Cwd, Home: request.Home})
			if result.Error == "" || !strings.Contains(result.Error, "conflicting") || err != nil {
				t.Fatalf("want first writer to succeed and second to reject conflict; second=%q discovery=%v", result.Error, err)
			}
			wantRoots := []string{}
			if firstRole == "personal" {
				wantRoots = []string{request.Cwd}
			}
			if scope.Project.Records != request.Records || !sameSetupPaths(scope.Project.AllowSources, wantRoots) {
				t.Fatalf("winning binding changed: %#v", scope.Project)
			}
		})
	}
}

func TestSetupSharedWriteRetainsRegistryAndDestinationLocks(t *testing.T) {
	request, path := setupWriteFixture(t, false)
	plan := setupWritePlan(t, request)
	if err := plan.Apply(context.Background()); err != nil {
		t.Fatal(err)
	}
	registry := filepath.Join(request.Home, ".context", "config.md")
	for _, lock := range []string{registry + ".lock", path + ".lock"} {
		if info, err := os.Stat(lock); err != nil || !info.Mode().IsRegular() {
			t.Fatalf("coordination sidecar %s: %v, %v", lock, info, err)
		}
	}
	if _, err := os.Stat(registry); !os.IsNotExist(err) {
		t.Fatalf("shared setup created a personal configuration: %v", err)
	}
}

func TestSetupPersonalWriteDoesNotNeedWritableCheckout(t *testing.T) {
	request, path := setupWriteFixture(t, false)
	request.Personal = true
	if err := os.Chmod(request.Cwd, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(request.Cwd, 0o755) })
	plan := setupWritePlan(t, request)
	if err := plan.Apply(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Dir(path)); !os.IsNotExist(err) {
		t.Fatalf("personal setup wrote to the checkout: %v", err)
	}
}

func TestSetupRegistryLockFailureLeavesDestinationUntouched(t *testing.T) {
	request, path := setupWriteFixture(t, false)
	plan := setupWritePlan(t, request)
	lock := filepath.Join(request.Home, ".context", "config.md.lock")
	if err := os.MkdirAll(lock, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := plan.Apply(context.Background()); err == nil || !strings.Contains(err.Error(), "lock setup configuration") {
		t.Fatalf("unwritable coordination sidecar: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("failed registry lock wrote the destination: %v", err)
	}
}
