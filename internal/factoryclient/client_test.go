package factoryclient

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestJSONAndRejections(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-secret" {
			t.Error("missing bearer header")
		}
		if r.URL.Query().Has("token") {
			t.Error("credential in URL")
		}
		if r.URL.Path == "/v1/denied" {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"code":"forbidden","message":"permission denied"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"job-1"}`))
	}))
	defer server.Close()
	client, err := New(server.URL, "test-secret")
	if err != nil {
		t.Fatal(err)
	}
	var job struct {
		ID string `json:"id"`
	}
	if err := client.DoJSON(context.Background(), http.MethodGet, "/v1/jobs/job-1", nil, &job); err != nil || job.ID != "job-1" {
		t.Fatalf("job = %+v, error = %v", job, err)
	}
	err = client.DoJSON(context.Background(), http.MethodGet, "/v1/denied", nil, nil)
	var rejection *APIError
	if !errors.As(err, &rejection) || rejection.Status != 403 || rejection.Code != "forbidden" {
		t.Fatalf("unexpected rejection: %v", err)
	}
	if err := client.DoJSON(context.Background(), http.MethodGet, "/v1/me?token=test-secret", nil, nil); err == nil {
		t.Fatal("credential URL was accepted")
	}
}

func TestCredentialsCannotRedirectOrUseCleartextRemote(t *testing.T) {
	called := false
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	defer target.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, http.StatusFound) }))
	defer server.Close()
	client, err := New(server.URL, "secret")
	if err != nil {
		t.Fatal(err)
	}
	if err := client.DoJSON(context.Background(), "GET", "/v1/me", nil, nil); err == nil || called {
		t.Fatalf("redirect error = %v, redirected = %v", err, called)
	}
	for _, endpoint := range []string{"http://example.com", "https://user:secret@example.com", "https://example.com?token=secret", "https://example.com#secret"} {
		if _, err := New(endpoint, "secret"); err == nil {
			t.Errorf("accepted endpoint %s", endpoint)
		}
	}
	if _, err := New(server.URL, "secret\nheader"); err == nil {
		t.Fatal("newline token accepted")
	}
	if err := client.DoJSON(context.Background(), "GET", "https://example.com/v1/me", nil, nil); err == nil || strings.Contains(err.Error(), "secret") {
		t.Fatalf("unsafe error: %v", err)
	}
}
