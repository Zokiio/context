package taskcontext

import "github.com/Zokiio/context/internal/recordread"

type sourceRole = recordread.Role

const (
	ticketSource   = recordread.RecordSource
	documentSource = recordread.DocumentSource
)

type sourceReader struct {
	*recordread.Reader
	project string
}

func newSourceReader(request Request) (*sourceReader, error) {
	reader, err := recordread.NewReader(request.ProjectDir, request.AllowedSourceDirs)
	if err != nil {
		return nil, err
	}
	return &sourceReader{Reader: reader, project: reader.Project()}, nil
}
func (r *sourceReader) close() { r.Close() }
func (r *sourceReader) read(path string, role sourceRole) (Source, *Diagnostic) {
	return r.Read(path, role)
}
func (r *sourceReader) linkPath(from string, link relationship) (string, *Diagnostic) {
	return r.LinkPath(from, recordread.Relationship{Kind: link.kind, Link: link.link, Destination: link.destination})
}
