package recordread

type Source struct {
	Path    string   `json:"path"`
	Text    string   `json:"text"`
	SHA256  string   `json:"sha256"`
	Reasons []Reason `json:"reasons"`
}

type Reason struct {
	Kind string `json:"kind"`
	From string `json:"from,omitempty"`
	Link string `json:"link,omitempty"`
}

type Diagnostic struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Path     string `json:"path"`
	From     string `json:"from,omitempty"`
	Link     string `json:"link,omitempty"`
}
