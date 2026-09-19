package recordread

import (
	"errors"
	"fmt"
)

const (
	DefaultMaxFiles       = 100
	DefaultMaxBytes int64 = 1_048_576
)

type Limits struct {
	// Zero uses the default. Negative limits are invalid.
	MaxFiles int
	MaxBytes int64
}

type Usage struct {
	Files int
	Bytes int64
}

// Limit describes why a complete source could not be admitted to a capture.
type Limit struct {
	description string
}

func (l *Limit) Error() string { return l.description }

// Capture holds immutable source bytes and one collection budget for a caller
// operation. Read always reapplies role authorization and, after a limit
// breach, returns only sources already captured. Admit counts a canonical
// physical source at most once.
type Capture struct {
	reader    *Reader
	limits    Limits
	usage     Usage
	admitted  map[string]bool
	exhausted *Limit
}

func NewCapture(project string, allowed []string, limits Limits) (*Capture, error) {
	if limits.MaxFiles < 0 || limits.MaxBytes < 0 {
		return nil, errors.New("source limits must be positive")
	}
	if limits.MaxFiles == 0 {
		limits.MaxFiles = DefaultMaxFiles
	}
	if limits.MaxBytes == 0 {
		limits.MaxBytes = DefaultMaxBytes
	}
	reader, err := NewReader(project, allowed)
	if err != nil {
		return nil, err
	}
	return &Capture{reader: reader, limits: limits, admitted: map[string]bool{}}, nil
}

func (c *Capture) Close() { c.reader.Close() }

func (c *Capture) Project() string { return c.reader.Project() }

func (c *Capture) Read(path string, role Role) (Source, *Diagnostic) {
	resolved, diagnostic := c.reader.resolve(path, role)
	if diagnostic != nil {
		return Source{}, diagnostic
	}
	if failure, exists := c.reader.failures[resolved]; exists {
		return c.reader.snapshots[resolved], &failure
	}
	if source, exists := c.reader.snapshots[resolved]; exists {
		return source, nil
	}
	if c.exhausted != nil {
		return Source{}, &Diagnostic{
			Code: "source_omitted", Severity: "error",
			Message: "source was not read after the shared collection limit was exceeded", Path: resolved,
		}
	}
	return c.reader.Read(path, role)
}

func (c *Capture) LinkPath(from string, relationship Relationship) (string, *Diagnostic) {
	return c.reader.LinkPath(from, relationship)
}

// Admit reserves budget for source unless its physical path was admitted by an
// earlier phase. The source is whole or rejected; its bytes are never truncated.
func (c *Capture) Admit(source Source) *Limit {
	if c.admitted[source.Path] {
		return nil
	}
	if c.exhausted != nil {
		return c.exhausted
	}
	if c.usage.Files >= c.limits.MaxFiles {
		c.exhausted = &Limit{description: fmt.Sprintf("file limit of %d", c.limits.MaxFiles)}
		return c.exhausted
	}
	if int64(len(source.Text)) > c.limits.MaxBytes-c.usage.Bytes {
		c.exhausted = &Limit{description: fmt.Sprintf("source byte limit of %d", c.limits.MaxBytes)}
		return c.exhausted
	}
	c.admitted[source.Path] = true
	c.usage.Files++
	c.usage.Bytes += int64(len(source.Text))
	return nil
}

// Admitted reports whether path resolves to an already admitted source for the
// requested role. It performs authorization without reading uncaptured bytes.
func (c *Capture) Admitted(path string, role Role) bool {
	resolved, diagnostic := c.reader.resolve(path, role)
	return diagnostic == nil && c.admitted[resolved]
}

func (c *Capture) Usage() Usage { return c.usage }

func (c *Capture) Exhausted() bool { return c.exhausted != nil }
