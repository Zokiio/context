# Read discovery configuration

The configuration reader owns ContextConfig validation and saved filesystem paths in `internal/discovery`. It shares path identity rules with the following resolver and setup slices. Existing application readers continue to receive explicit scope. This slice implements [ticket 01](../records/project-discovery/issues/01-read-configuration.md).

## Caller usage

```go
config, err := discovery.ReadConfig(configFile, discovery.SharedConfig)
if err != nil {
	return err
}
records := config.Project.Records
if records.Err != nil {
	return records.Err
}
// Selection must still check the original path's availability and directory type.
considerRecords(records.Path, records.Canonical)
```

Personal callers read `config.Projects` and `config.Workspaces` in authored order. Future setup can validate proposed bytes through `ParseConfig` and retain `config.Raw` for concurrent-change detection. `Metadata` and `Body` preserve unknown authored fields and Markdown.

## Module boundary

`config.go` owns profile parsing and validation. `path.go` owns saved path provenance, canonical identity, and physical-location comparison. `recordread.ParseDocument` owns Markdown framing and YAML decoding. No YAML AST crosses the discovery interface.

`Config` contains either shared project/workspace declarations or personal registrations. Project bindings contain records and allowed sources. Personal project registrations add a key, optional alias, and directory. Workspace declarations contain identity, title, and ordered members. Personal workspace registrations add a key, optional alias, and optional directory. Members retain their own records and allowed roots.

A `ConfigError` identifies the declaring file, field, and authored value. `PathValue` retains its origin, authored spelling, filesystem spelling, canonical identity, and any identity-resolution failure. An unavailable registration does not make the whole document invalid.

## Path semantics

Relative values use the encountered configuration filename's directory, even when that file is a symlink. Path processing preserves `..` until preceding symlinks have been followed. Cleaning the spelling first can redirect a valid declaration to a different record store.

Canonical identity can identify a missing target through its existing ancestors. It does not establish availability. The resolver must check the original selected path before opening a directory, so a spelling such as `missing/../records` cannot become a usable directory merely through normalization. Non-directory ancestors and symlink loops retain errors. Case-insensitive aliases use physical identity when the filesystem provides it.

Configuration reads check for a regular file before opening it, including when the configuration filename is a symlink. Tilde, environment-looking strings, commas, and percent escapes remain literal filesystem values.

## Design choice

A standalone functional reader and a document-owned reader with opaque path handles were compared. Both met the initial interface, preservation, path, testability, and scope criteria. The functional boundary remains the chosen shape. The completed implementation already supplies that boundary alongside the resolver and setup path rules, so these slices use its shared module.

This keeps one configuration parser and one path identity implementation. It avoids a second conversion layer between independent configuration and resolver models. `ReadConfig` handles an existing file; `ParseConfig` validates proposed bytes for setup without writing them. Neither selects a scope.

## Verification

External-package tests cover both profiles, field diagnostics, YAML and declaration duplicates, unknown metadata and Markdown preservation, independent record stores, symlinked configuration files, literal values, unavailable targets, symlink-before-parent traversal, non-directory ancestors, and physical identity.

The review correction adds regressions for nested mapping keys that alias an earlier scalar anchor. Duplicate keys are rejected after alias resolution; distinct alias keys and YAML merge overrides retain their meaning. Criterion evidence and the current Acceptance record retain the verified source revision.
