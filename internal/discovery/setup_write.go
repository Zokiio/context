package discovery

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/rogpeppe/go-internal/lockedfile"
)

type setupSnapshot struct {
	data        []byte
	info        os.FileInfo
	permissions setupPermissionSnapshot
}

func observeSetupFile(path string) (string, setupSnapshot, error) {
	target, err := CanonicalPath(path)
	if err != nil {
		return target, setupSnapshot{}, fmt.Errorf("resolve setup destination %s: %w", path, err)
	}
	present, err := configPresent(path)
	if err != nil || !present {
		return target, setupSnapshot{}, err
	}
	if err := checkRegularFile(path); err != nil {
		return target, setupSnapshot{}, fmt.Errorf("read setup destination %s: %w", path, err)
	}
	file, err := os.Open(path)
	if err != nil {
		return target, setupSnapshot{}, fmt.Errorf("read setup destination %s: %w", path, err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return target, setupSnapshot{}, err
	}
	if !info.Mode().IsRegular() {
		return target, setupSnapshot{}, fmt.Errorf("setup destination %s is not a regular file", path)
	}
	permissions := captureSetupPermissions(path, info)
	data, err := io.ReadAll(file)
	if err != nil {
		return target, setupSnapshot{}, fmt.Errorf("read setup destination %s: %w", path, err)
	}
	after, err := os.Stat(path)
	if err != nil || !sameSetupFile(info, after) {
		return target, setupSnapshot{}, setupChanged(path)
	}
	if !sameSetupPermissionSnapshots(permissions, captureSetupPermissions(path, after)) {
		return target, setupSnapshot{}, setupChanged(path)
	}
	return target, setupSnapshot{data: data, info: info, permissions: permissions}, nil
}

func sameSetupFile(left, right os.FileInfo) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return os.SameFile(left, right) && left.Mode() == right.Mode() && left.Size() == right.Size() && left.ModTime().Equal(right.ModTime()) && sameSetupPermissions(left, right)
}

func setupChanged(path string) error {
	return fmt.Errorf("configuration %s changed since setup read it; rerun setup to review the current document", path)
}

func (p *SetupPlan) checkUnchanged() error {
	target, current, err := observeSetupFile(p.summary.Destination)
	if err != nil {
		return fmt.Errorf("check setup destination before writing: %w", err)
	}
	if target != p.target || !sameSetupFile(p.before.info, current.info) || !bytes.Equal(p.before.data, current.data) || !sameSetupPermissionSnapshots(p.before.permissions, current.permissions) {
		return setupChanged(p.summary.Destination)
	}
	return nil
}

type setupTempFile interface {
	io.WriteCloser
	Name() string
	Sync() error
}

// Tests use real temporary files with injected write and rename failures.
type setupWriteOps struct {
	createTemp          func(string, string) (setupTempFile, error)
	rename              func(string, string) error
	lock                func(string) (func(), error)
	preservePermissions func(string, setupPermissionSnapshot) error
}

func defaultSetupWriteOps() setupWriteOps {
	return setupWriteOps{
		createTemp:          func(directory, pattern string) (setupTempFile, error) { return os.CreateTemp(directory, pattern) },
		rename:              os.Rename,
		lock:                func(path string) (func(), error) { return lockedfile.MutexAt(path).Lock() },
		preservePermissions: preserveSetupPermissions,
	}
}

// Apply writes the prepared document only if the destination is unchanged and
// the proposed binding still resolves. Changed setup operations coordinate via
// the personal registry's sidecar lock; shared writes also lock their destination.
// Sidecars remain in place because unlinking them would split coordination.
func (p *SetupPlan) Apply(ctx context.Context) error {
	return p.apply(ctx, defaultSetupWriteOps())
}

func (p *SetupPlan) apply(ctx context.Context, ops setupWriteOps) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := p.checkUnchanged(); err != nil {
		return err
	}
	if _, err := resolve(ctx, p.request, p.proposed); err != nil {
		return fmt.Errorf("recheck proposed setup binding: %w", err)
	}
	if p.summary.Change == SetupUnchanged {
		return nil
	}
	locks, err := p.LockPaths()
	if err != nil {
		return err
	}
	if err := checkDirectory(p.request.Home); err != nil {
		return fmt.Errorf("read setup coordination home %s: %w", p.request.Home, err)
	}
	locks, err = orderSetupLocks(locks)
	if err != nil {
		return err
	}
	for _, path := range locks {
		unlock, err := ops.lock(path)
		if err != nil {
			return fmt.Errorf("lock setup configuration %s: %w", path, err)
		}
		defer unlock()
	}
	if err := p.checkUnchanged(); err != nil {
		return err
	}
	// Shared and personal files can describe the same binding. Serialize their
	// validation and commit for this home, then inspect the winner's declaration.
	if _, err := resolve(ctx, p.request, p.proposed); err != nil {
		return fmt.Errorf("recheck proposed setup binding: %w", err)
	}
	parent := filepath.Dir(p.target)
	temporary, err := ops.createTemp(parent, ".ctx-setup-*")
	if err != nil {
		return fmt.Errorf("create setup temporary file: %w", err)
	}
	defer os.Remove(temporary.Name())
	defer temporary.Close()
	if n, err := io.WriteString(temporary, p.document); err != nil || n != len(p.document) {
		if err == nil {
			err = io.ErrShortWrite
		}
		return fmt.Errorf("write setup temporary file: %w", err)
	}
	if err := ops.preservePermissions(temporary.Name(), p.before.permissions); err != nil {
		return fmt.Errorf("preserve setup file permissions: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		return fmt.Errorf("sync setup temporary file: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close setup temporary file: %w", err)
	}
	// A writer that does not participate in our lock can still change either
	// this file or another effective declaration while the temp file is built.
	if _, err := resolve(ctx, p.request, p.proposed); err != nil {
		return fmt.Errorf("recheck proposed setup binding: %w", err)
	}
	if err := p.checkUnchanged(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := ops.rename(temporary.Name(), p.target); err != nil {
		return fmt.Errorf("atomically replace setup configuration %s: %w", p.summary.Destination, err)
	}
	return nil
}
