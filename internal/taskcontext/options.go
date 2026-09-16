package taskcontext

const (
	DefaultMaxFiles       = 100
	DefaultMaxBytes int64 = 1_048_576
)

type Request struct {
	ProjectDir        string
	TicketPath        string
	AllowedSourceDirs []string
	// Zero uses the default. Negative limits are invalid.
	MaxFiles int
	MaxBytes int64
}
