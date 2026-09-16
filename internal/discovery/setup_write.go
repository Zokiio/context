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
	data []byte
	info os.FileInfo
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
	data, err := io.ReadAll(file)
	if err != nil {
		return target, setupSnapshot{}, fmt.Errorf("read setup destination %s: %w", path, err)
	}
	after, err := os.Stat(path)
	if err != nil || !sameSetupFile(info, after) {
		return target, setupSnapshot{}, setupChanged(path)
	}
	return target, setupSnapshot{data: data, info: info}, nil
}

func sameSetupFile(left, right os.FileInfo) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return os.SameFile(left, right) && left.Mode() == right.Mode() && left.Size() == right.Size() && left.ModTime().Equal(right.ModTime())
}

func setupChanged(path string) error {
	return fmt.Errorf("configuration %s changed since setup read it; rerun setup to review the current document", path)
}

func (p *SetupPlan) checkUnchanged() error {
	target, current, err := observeSetupFile(p.summary.Destination)
	if err != nil {
		return fmt.Errorf("check setup destination before writing: %w", err)
	}
	if target != p.target || !sameSetupFile(p.before.info, current.info) || !bytes.Equal(p.before.data, current.data) {
		return setupChanged(p.summary.Destination)
	}
	return nil
}

type setupTempFile interface {
	io.WriteCloser
	Name() string
	Chmod(os.FileMode) error
	Sync() error
}

// Tests use real temporary files with injected write and rename failures.
type setupWriteOps struct {
	createTemp func(string, string) (setupTempFile, error)
	rename     func(string, string) error
}

func defaultSetupWriteOps() setupWriteOps {
	return setupWriteOps{
		createTemp: func(directory, pattern string) (setupTempFile, error) { return os.CreateTemp(directory, pattern) },
		rename:     os.Rename,
	}
}

// Apply writes the prepared document only if the destination is unchanged and
// the proposed binding still resolves. Cooperating writers retain the sidecar
// <physical destination>.lock; unlinking it would split the coordination lock.
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
	parent := filepath.Dir(p.target)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return fmt.Errorf("create setup configuration directory %s: %w", parent, err)
	}
	unlock, err := lockedfile.MutexAt(p.target + ".lock").Lock()
	if err != nil {
		return fmt.Errorf("lock setup destination %s: %w", p.summary.Destination, err)
	}
	defer unlock()
	if err := p.checkUnchanged(); err != nil {
		return err
	}
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
	mode := os.FileMode(0o644)
	if p.before.info != nil {
		mode = p.before.info.Mode()
	}
	if err := temporary.Chmod(mode); err != nil {
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
