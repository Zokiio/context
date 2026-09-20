package taskcontext

import "github.com/Zokiio/context/internal/recordread"

const (
	DefaultMaxFiles       = recordread.DefaultMaxFiles
	DefaultMaxBytes int64 = recordread.DefaultMaxBytes
)

type Request struct {
	ProjectDir        string
	TicketPath        string
	AllowedSourceDirs []string
	// Zero uses the default. Negative limits are invalid.
	MaxFiles int
	MaxBytes int64
}
