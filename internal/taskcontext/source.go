package taskcontext

import "github.com/Zokiio/context/internal/recordread"

type sourceRole = recordread.Role

const (
	ticketSource   = recordread.RecordSource
	documentSource = recordread.DocumentSource
)

type sourceReader struct {
	*recordread.Capture
	project string
}

func newSourceReader(request Request) (*sourceReader, error) {
	reader, err := recordread.NewCapture(request.ProjectDir, request.AllowedSourceDirs, recordread.Limits{MaxFiles: request.MaxFiles, MaxBytes: request.MaxBytes})
	if err != nil {
		return nil, err
	}
	return capturedSourceReader(reader), nil
}
func capturedSourceReader(capture *recordread.Capture) *sourceReader {
	return &sourceReader{Capture: capture, project: capture.Project()}
}
func (r *sourceReader) read(path string, role sourceRole) (Source, *Diagnostic) {
	return r.Read(path, role)
}
func (r *sourceReader) linkPath(from string, link relationship) (string, *Diagnostic) {
	return r.LinkPath(from, recordread.Relationship{Kind: link.kind, Link: link.link, Destination: link.destination})
}
