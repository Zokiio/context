package cli

import (
	"bytes"
	"context"
	"errors"
	"runtime/debug"
	"strings"
	"testing"
)

func TestVersionBuildProvenance(t *testing.T) {
	for _, tc := range []struct {
		name string
		info *debug.BuildInfo
		want string
	}{
		{"unavailable", nil, "ctx unknown\nrevision: unknown\nmodified: unknown\n"},
		{"unstamped", &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}}, "ctx (devel)\nrevision: unknown\nmodified: unknown\n"},
		{"clean", &debug.BuildInfo{Main: debug.Module{Version: "v1.2.3"}, Settings: []debug.BuildSetting{{Key: "vcs.revision", Value: "abc123"}, {Key: "vcs.modified", Value: "false"}}}, "ctx v1.2.3\nrevision: abc123\nmodified: false\n"},
		{"dirty", &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}, Settings: []debug.BuildSetting{{Key: "vcs.revision", Value: "abc123"}, {Key: "vcs.modified", Value: "true"}}}, "ctx (devel)\nrevision: abc123\nmodified: true\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			if err := renderVersion(&out, tc.info); err != nil {
				t.Fatal(err)
			}
			if out.String() != tc.want {
				t.Fatalf("got %q, want %q", out.String(), tc.want)
			}
		})
	}
}

func TestVersionDoesNotDiscoverProject(t *testing.T) {
	var out, stderr bytes.Buffer
	noDiscovery := func() (string, error) { t.Fatal("version tried to discover a project"); return "", nil }
	env := Environment{WorkingDirectory: noDiscovery, HomeDirectory: noDiscovery}
	code := RunWithEnvironment(context.Background(), []string{"ctx", "version"}, &out, &stderr, Operations{}, env)
	if code != 0 || stderr.Len() != 0 || !strings.Contains(out.String(), "revision: ") || !strings.Contains(out.String(), "modified: ") {
		t.Fatalf("exit=%d stdout=%q stderr=%q", code, out.String(), stderr.String())
	}
	out.Reset()
	stderr.Reset()
	code = RunWithEnvironment(context.Background(), []string{"ctx", "version", "extra"}, &out, &stderr, Operations{}, env)
	if code != 2 || out.Len() != 0 {
		t.Fatalf("extra argument: exit=%d stdout=%q", code, out.String())
	}
	code = RunWithEnvironment(context.Background(), []string{"ctx", "version"}, versionFailWriter{}, &stderr, Operations{}, env)
	if code != 2 {
		t.Fatalf("write failure exit=%d", code)
	}
}

type versionFailWriter struct{}

func (versionFailWriter) Write([]byte) (int, error) { return 0, errors.New("output unavailable") }
