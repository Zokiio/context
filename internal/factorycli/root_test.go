package factorycli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	ctxcli "github.com/Zokiio/context/internal/cli"
	"github.com/Zokiio/context/internal/factory"
)

func TestFactoryDeniedReturnsNonzeroThroughCtxRoot(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer root-cli-token" {
			t.Error("missing authentication")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		io.WriteString(w, `{"code":"forbidden","message":"operation denied"}`)
	}))
	defer server.Close()
	tokenPath := filepath.Join(t.TempDir(), "token")
	if e := os.WriteFile(tokenPath, []byte("root-cli-token"), 0600); e != nil {
		t.Fatal(e)
	}
	var out, stderr bytes.Buffer
	status := ctxcli.Run(context.Background(), []string{"ctx", "factory", "--url", server.URL, "--token-file", tokenPath, "projects", "list"}, &out, &stderr, ctxcli.Operations{})
	if status == 0 || out.Len() != 0 || !strings.Contains(stderr.String(), "operation denied") {
		t.Fatalf("status=%d out=%q stderr=%q", status, out.String(), stderr.String())
	}
	if strings.Contains(stderr.String(), "root-cli-token") {
		t.Fatal("secret in diagnostics")
	}
}
func TestFactoryInitBootstrapsOnceIntoPrivateFile(t *testing.T) {
	dir := t.TempDir()
	state := filepath.Join(dir, "state")
	token := filepath.Join(dir, "admin-token")
	var out, stderr bytes.Buffer
	args := []string{"ctx", "factory", "init", "--data-dir", state, "--username", "operator", "--token-file", token}
	if status := ctxcli.Run(context.Background(), args, &out, &stderr, ctxcli.Operations{}); status != 0 {
		t.Fatalf("init status=%d error=%s", status, stderr.String())
	}
	secret, e := os.ReadFile(token)
	if e != nil || len(secret) == 0 {
		t.Fatalf("token %q %v", secret, e)
	}
	info, e := os.Stat(token)
	if e != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("private token metadata %+v %v", info, e)
	}
	if strings.Contains(out.String(), strings.TrimSpace(string(secret))) || strings.Contains(stderr.String(), strings.TrimSpace(string(secret))) {
		t.Fatal("bootstrap secret leaked to output")
	}
	var result struct {
		User      factory.User `json:"user"`
		TokenFile string       `json:"tokenFile"`
	}
	if e = json.Unmarshal(out.Bytes(), &result); e != nil || result.User.Username != "operator" || result.TokenFile != token {
		t.Fatalf("result %+v %v", result, e)
	}
	service, e := factory.Open(state, factory.Options{})
	if e != nil {
		t.Fatal(e)
	}
	principal, e := service.Authenticate(context.Background(), strings.TrimSpace(string(secret)))
	if e != nil || !principal.InstallationAdmin {
		t.Fatalf("bootstrap credential %+v %v", principal, e)
	}
	if e = service.Close(); e != nil {
		t.Fatal(e)
	}
	second := filepath.Join(dir, "second-token")
	out.Reset()
	stderr.Reset()
	args = []string{"ctx", "factory", "--token-file", second, "init", "--data-dir", state}
	if status := ctxcli.Run(context.Background(), args, &out, &stderr, ctxcli.Operations{}); status == 0 {
		t.Fatal("second initial administrator created")
	}
	if _, e = os.Stat(second); !os.IsNotExist(e) {
		t.Fatal("failed bootstrap left token output")
	}
}
func TestFactoryInitRefusesExistingTokenBeforeOpeningState(t *testing.T) {
	dir := t.TempDir()
	token := filepath.Join(dir, "token")
	if e := os.WriteFile(token, []byte("keep-existing-token"), 0600); e != nil {
		t.Fatal(e)
	}
	state := filepath.Join(dir, "unopened-state")
	var out, stderr bytes.Buffer
	if status := ctxcli.Run(context.Background(), []string{"ctx", "factory", "init", "--data-dir", state, "--token-file", token}, &out, &stderr, ctxcli.Operations{}); status == 0 {
		t.Fatal("overwrote token")
	}
	if b, e := os.ReadFile(token); e != nil || string(b) != "keep-existing-token" {
		t.Fatalf("existing token %q %v", b, e)
	}
	if _, e := os.Stat(state); !os.IsNotExist(e) {
		t.Fatal("refused invocation opened service state")
	}
}
