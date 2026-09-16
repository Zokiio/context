package discoveryconfig

import "path/filepath"

func (d *decoder) path(fields map[string]any, key, prefix string) TargetPath {
	return d.target(d.stringValue(fields, key, prefix, true), fieldPath(prefix, key))
}

func (d *decoder) optionalPath(fields map[string]any, key, prefix string) *TargetPath {
	if _, present := fields[key]; !present {
		return nil
	}
	path := d.path(fields, key, prefix)
	return &path
}

func (d *decoder) target(authored, field string) TargetPath {
	if d.err != nil {
		return TargetPath{}
	}
	resolved := authored
	if !filepath.IsAbs(resolved) {
		// Use the encountered config directory, not its symlink target. These
		// are filesystem values, not bundle-relative Markdown links.
		resolved = filepath.Join(filepath.Dir(d.file), resolved)
	}
	resolved = filepath.Clean(resolved)
	path := TargetPath{File: d.file, Field: field, Authored: authored, Resolved: resolved}
	canonical, err := filepath.EvalSymlinks(resolved)
	if err != nil {
		path.Failure = &FieldError{File: d.file, Field: field, Value: authored, Resolved: resolved, Message: "cannot resolve target", Err: err}
	} else {
		path.Canonical = canonical
	}
	return path
}
