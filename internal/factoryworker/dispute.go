package factoryworker

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/Zokiio/context/internal/factory"
)

const maxDisputeBytes = 64 << 10

// readDispute reads only the named regular file within its already-open parent.
// It never interprets free-form agent output as a dispute.
func readDispute(path string) (*factory.DisputeInput, error) {
	if path == "" {
		return nil, nil
	}
	root, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	defer root.Close()
	file, err := openDisputeFile(root, filepath.Base(path))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open dispute report: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > maxDisputeBytes {
		return nil, errors.New("dispute report must be a regular file of at most 64 KiB")
	}
	raw, err := io.ReadAll(io.LimitReader(file, maxDisputeBytes+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > maxDisputeBytes {
		return nil, errors.New("dispute report exceeds 64 KiB")
	}
	var dispute factory.DisputeInput
	if err := strictJSON(raw, &dispute); err != nil {
		return nil, fmt.Errorf("decode dispute report: %w", err)
	}
	if err := factory.ValidateDisputeInput(dispute); err != nil {
		return nil, err
	}
	return &dispute, nil
}

func (p *Process) disputePath(id string) string {
	return filepath.Join(filepath.Dir(p.receiptPath(id)), "dispute.json")
}

func publishAgentDispute(path string, dispute *factory.DisputeInput) error {
	data, err := json.Marshal(dispute)
	if err != nil {
		return err
	}
	if err := writeExclusive(path, data, 0o600); err != nil {
		return fmt.Errorf("retain agent dispute: %w", err)
	}
	return syncDir(filepath.Dir(path))
}
