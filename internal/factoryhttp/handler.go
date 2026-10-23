// Package factoryhttp exposes the factory application and its embedded web
// workspace. All permissions are enforced again by factory.Service.
package factoryhttp

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/Zokiio/context/internal/factory"
)

//go:embed assets/*
var assets embed.FS

const maxBodyBytes = 2 << 20

type endpoint func(http.ResponseWriter, *http.Request, factory.Principal)

// NewHandler serves one application's API and browser workspace. It neither
// opens another controller nor starts a background execution loop.
func NewHandler(service *factory.Service) http.Handler {
	mux := http.NewServeMux()
	get := func(pattern string, fn endpoint) { mux.HandleFunc("GET "+pattern, authenticated(service, fn)) }
	get("/v1/me", func(w http.ResponseWriter, r *http.Request, p factory.Principal) { writeJSON(w, http.StatusOK, p) })
	get("/v1/users", func(w http.ResponseWriter, r *http.Request, p factory.Principal) {
		v, err := service.ListUsers(r.Context(), p)
		respond(w, 200, v, err)
	})
	mutation(mux, service, "POST /v1/users", 201, service.CreateUser)
	mutationPath(mux, service, "PUT /v1/users/{user}/disabled", 200, func(ctx context.Context, p factory.Principal, q factory.SetUserDisabledRequest, r *http.Request) (factory.User, error) {
		if err := bindID(r.PathValue("user"), string(q.UserID)); err != nil {
			return factory.User{}, err
		}
		q.UserID = factory.UserID(r.PathValue("user"))
		return service.SetUserDisabled(ctx, p, q)
	})
	mutationPath(mux, service, "POST /v1/users/{user}/tokens", 201, func(ctx context.Context, p factory.Principal, q factory.IssueTokenRequest, r *http.Request) (factory.IssuedToken, error) {
		if err := bindID(r.PathValue("user"), string(q.UserID)); err != nil {
			return factory.IssuedToken{}, err
		}
		q.UserID = factory.UserID(r.PathValue("user"))
		return service.IssueToken(ctx, p, q)
	})
	mutationPath(mux, service, "POST /v1/tokens/{token}/revoke", 200, func(ctx context.Context, p factory.Principal, q factory.RevokeTokenRequest, r *http.Request) (struct{}, error) {
		if err := bindID(r.PathValue("token"), string(q.TokenID)); err != nil {
			return struct{}{}, err
		}
		q.TokenID = factory.ID(r.PathValue("token"))
		return struct{}{}, service.RevokeToken(ctx, p, q)
	})
	get("/v1/projects", func(w http.ResponseWriter, r *http.Request, p factory.Principal) {
		v, err := service.ListProjects(r.Context(), p)
		respond(w, 200, v, err)
	})
	mutation(mux, service, "POST /v1/projects", 201, service.RegisterProject)
	get("/v1/projects/{project}/orientation", func(w http.ResponseWriter, r *http.Request, p factory.Principal) {
		v, err := service.Orientation(r.Context(), p, factory.ProjectID(r.PathValue("project")))
		respond(w, 200, v, err)
	})
	get("/v1/projects/{project}/metrics", func(w http.ResponseWriter, r *http.Request, p factory.Principal) {
		v, err := service.Metrics(r.Context(), p, factory.ProjectID(r.PathValue("project")))
		respond(w, 200, v, err)
	})
	get("/v1/projects/{project}/grants", func(w http.ResponseWriter, r *http.Request, p factory.Principal) {
		v, err := service.ListProjectGrants(r.Context(), p, factory.ProjectID(r.PathValue("project")))
		respond(w, 200, v, err)
	})
	get("/v1/projects/{project}/capabilities", func(w http.ResponseWriter, r *http.Request, p factory.Principal) {
		v, err := service.Capabilities(r.Context(), p, factory.ProjectID(r.PathValue("project")))
		respond(w, 200, v, err)
	})
	mutationPath(mux, service, "PUT /v1/projects/{project}/grants/{user}", 200, func(ctx context.Context, p factory.Principal, q factory.GrantProjectRequest, r *http.Request) (factory.Grant, error) {
		if err := bindID(r.PathValue("project"), string(q.ProjectID)); err != nil {
			return factory.Grant{}, err
		}
		if err := bindID(r.PathValue("user"), string(q.UserID)); err != nil {
			return factory.Grant{}, err
		}
		q.ProjectID = factory.ProjectID(r.PathValue("project"))
		q.UserID = factory.UserID(r.PathValue("user"))
		return service.GrantProject(ctx, p, q)
	})
	get("/v1/executors", func(w http.ResponseWriter, r *http.Request, p factory.Principal) {
		v, err := service.ListExecutors(r.Context(), p)
		respond(w, 200, v, err)
	})
	mutation(mux, service, "POST /v1/executors", 201, service.RegisterExecutor)
	get("/v1/checks", func(w http.ResponseWriter, r *http.Request, p factory.Principal) {
		v, err := service.ListChecks(r.Context(), p)
		respond(w, 200, v, err)
	})
	mutation(mux, service, "POST /v1/checks", 201, service.RegisterCheck)
	get("/v1/jobs", func(w http.ResponseWriter, r *http.Request, p factory.Principal) {
		q := r.URL.Query()
		limit := 100
		if raw := q.Get("limit"); raw != "" {
			n, err := strconv.Atoi(raw)
			if err != nil || n < 1 || n > 500 {
				writeError(w, invalid("limit must be between 1 and 500"))
				return
			}
			limit = n
		}
		v, err := service.Jobs(r.Context(), p, factory.JobFilter{ProjectID: factory.ProjectID(q.Get("projectId")), Status: factory.JobStatus(q.Get("status")), Limit: limit, After: q.Get("after")})
		respond(w, 200, v, err)
	})
	mutation(mux, service, "POST /v1/jobs", 202, service.Submit)
	mutationPath(mux, service, "POST /v1/jobs/{job}/merge", 200, func(ctx context.Context, p factory.Principal, q factory.MergeRequest, r *http.Request) (factory.Job, error) {
		if err := bindID(r.PathValue("job"), string(q.JobID)); err != nil {
			return factory.Job{}, err
		}
		q.JobID = factory.JobID(r.PathValue("job"))
		return service.Merge(ctx, p, q)
	})
	mutationPath(mux, service, "POST /v1/projects/{project}/expectations", 202, func(ctx context.Context, p factory.Principal, q factory.StartExpectationsRequest, r *http.Request) (factory.Expectation, error) {
		if err := bindID(r.PathValue("project"), string(q.ProjectID)); err != nil {
			return factory.Expectation{}, err
		}
		q.ProjectID = factory.ProjectID(r.PathValue("project"))
		return service.StartExpectations(ctx, p, q)
	})
	get("/v1/projects/{project}/expectations", func(w http.ResponseWriter, r *http.Request, p factory.Principal) {
		v, err := service.ListExpectations(r.Context(), p, factory.ProjectID(r.PathValue("project")), r.URL.Query().Get("workItemId"))
		respond(w, 200, v, err)
	})
	get("/v1/expectations/{expectation}", func(w http.ResponseWriter, r *http.Request, p factory.Principal) {
		v, err := service.GetExpectations(r.Context(), p, factory.ID(r.PathValue("expectation")))
		respond(w, 200, v, err)
	})
	for _, decision := range []string{"accept", "reject"} {
		decision := decision
		mutationPath(mux, service, "POST /v1/expectations/{expectation}/"+decision, 200, func(ctx context.Context, p factory.Principal, q factory.DecideExpectationsRequest, r *http.Request) (factory.Expectation, error) {
			if err := bindID(r.PathValue("expectation"), string(q.ExpectationID)); err != nil {
				return factory.Expectation{}, err
			}
			q.ExpectationID = factory.ID(r.PathValue("expectation"))
			if decision == "accept" {
				return service.AcceptExpectations(ctx, p, q)
			}
			return service.RejectExpectations(ctx, p, q)
		})
	}
	mutationPath(mux, service, "POST /v1/projects/{project}/refinements", 202, func(ctx context.Context, p factory.Principal, q factory.StartRefinementRequest, r *http.Request) (factory.Refinement, error) {
		if err := bindID(r.PathValue("project"), string(q.ProjectID)); err != nil {
			return factory.Refinement{}, err
		}
		q.ProjectID = factory.ProjectID(r.PathValue("project"))
		return service.StartRefinement(ctx, p, q)
	})
	get("/v1/projects/{project}/refinements", func(w http.ResponseWriter, r *http.Request, p factory.Principal) {
		v, err := service.Refinements(r.Context(), p, factory.ProjectID(r.PathValue("project")), r.URL.Query().Get("workItemId"))
		respond(w, 200, v, err)
	})
	get("/v1/refinements/{refinement}", func(w http.ResponseWriter, r *http.Request, p factory.Principal) {
		v, err := service.GetRefinement(r.Context(), p, factory.ID(r.PathValue("refinement")))
		respond(w, 200, v, err)
	})
	mutationPath(mux, service, "POST /v1/refinements/{refinement}/accept", 200, func(ctx context.Context, p factory.Principal, q factory.AcceptRefinementRequest, r *http.Request) (factory.Refinement, error) {
		if err := bindID(r.PathValue("refinement"), string(q.RefinementID)); err != nil {
			return factory.Refinement{}, err
		}
		q.RefinementID = factory.ID(r.PathValue("refinement"))
		return service.AcceptRefinement(ctx, p, q)
	})
	mutationPath(mux, service, "POST /v1/refinements/{refinement}/reject", 200, func(ctx context.Context, p factory.Principal, q factory.RejectRefinementRequest, r *http.Request) (factory.Refinement, error) {
		if err := bindID(r.PathValue("refinement"), string(q.RefinementID)); err != nil {
			return factory.Refinement{}, err
		}
		q.RefinementID = factory.ID(r.PathValue("refinement"))
		return service.RejectRefinement(ctx, p, q)
	})
	get("/v1/jobs/{job}", func(w http.ResponseWriter, r *http.Request, p factory.Principal) {
		v, err := service.GetJob(r.Context(), p, factory.JobID(r.PathValue("job")))
		respond(w, 200, v, err)
	})
	get("/v1/jobs/{job}/events", func(w http.ResponseWriter, r *http.Request, p factory.Principal) {
		after := int64(0)
		if raw := r.URL.Query().Get("after"); raw != "" {
			n, err := strconv.ParseInt(raw, 10, 64)
			if err != nil || n < 0 {
				writeError(w, invalid("after must be a nonnegative event sequence"))
				return
			}
			after = n
		}
		job, err := service.GetJob(r.Context(), p, factory.JobID(r.PathValue("job")))
		if err != nil {
			writeError(w, err)
			return
		}
		events := []factory.Event{}
		for _, event := range job.Events {
			if event.Sequence > after {
				events = append(events, event)
			}
		}
		writeJSON(w, 200, events)
	})
	get("/v1/jobs/{job}/artifacts/{artifact}", func(w http.ResponseWriter, r *http.Request, p factory.Principal) {
		job, err := service.GetJob(r.Context(), p, factory.JobID(r.PathValue("job")))
		if err != nil {
			writeError(w, err)
			return
		}
		for _, attempt := range job.Attempts {
			for _, artifact := range attempt.Artifacts {
				if string(artifact.ID) == r.PathValue("artifact") {
					writeJSON(w, 200, artifact)
					return
				}
			}
		}
		for _, check := range job.Checks {
			for _, artifact := range check.Artifacts {
				if string(artifact.ID) == r.PathValue("artifact") {
					writeJSON(w, 200, artifact)
					return
				}
			}
		}
		writeError(w, &factory.Error{Code: factory.CodeNotFound, Message: "artifact not found"})
	})
	mutationPath(mux, service, "POST /v1/jobs/{job}/actions", 200, func(ctx context.Context, p factory.Principal, q factory.JobActionRequest, r *http.Request) (factory.Job, error) {
		if err := bindID(r.PathValue("job"), string(q.JobID)); err != nil {
			return factory.Job{}, err
		}
		q.JobID = factory.JobID(r.PathValue("job"))
		return service.Action(ctx, p, q)
	})
	mutationPath(mux, service, "POST /v1/jobs/{job}/reviews", 200, func(ctx context.Context, p factory.Principal, q factory.ReviewRequest, r *http.Request) (factory.Job, error) {
		if err := bindID(r.PathValue("job"), string(q.JobID)); err != nil {
			return factory.Job{}, err
		}
		q.JobID = factory.JobID(r.PathValue("job"))
		return service.Review(ctx, p, q)
	})
	mutationPath(mux, service, "POST /v1/jobs/{job}/acceptance", 200, func(ctx context.Context, p factory.Principal, q factory.AcceptRequest, r *http.Request) (factory.Job, error) {
		if err := bindID(r.PathValue("job"), string(q.JobID)); err != nil {
			return factory.Job{}, err
		}
		q.JobID = factory.JobID(r.PathValue("job"))
		return service.Accept(ctx, p, q)
	})
	mutation(mux, service, "POST /v1/worker/claims", 200, service.Claim)
	mutation(mux, service, "POST /v1/worker/heartbeat", 200, service.Heartbeat)
	mutation(mux, service, "POST /v1/worker/reports", 200, service.Report)
	mux.HandleFunc("/v1/", authenticated(service, func(w http.ResponseWriter, r *http.Request, p factory.Principal) {
		allowed := []string{}
		for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete} {
			probe := r.Clone(r.Context())
			probe.Method = method
			_, pattern := mux.Handler(probe)
			if strings.HasPrefix(pattern, method+" /v1/") {
				allowed = append(allowed, method)
			}
		}
		if len(allowed) > 0 {
			w.Header().Set("Allow", strings.Join(allowed, ", "))
			writeJSON(w, http.StatusMethodNotAllowed, &factory.Error{Code: factory.CodeInvalid, Message: "method is not supported for this endpoint"})
			return
		}
		writeError(w, &factory.Error{Code: factory.CodeNotFound, Message: "endpoint not found"})
	}))
	for path, name := range map[string]string{
		"/": "index.html", "/assets/app.js": "app.js", "/assets/app.css": "app.css",
		"/assets/fonts/instrument-sans-latin.woff2":     "fonts/instrument-sans-latin.woff2",
		"/assets/fonts/instrument-sans-latin-ext.woff2": "fonts/instrument-sans-latin-ext.woff2",
		"/assets/fonts/roboto-mono-latin.woff2":         "fonts/roboto-mono-latin.woff2",
		"/assets/fonts/roboto-mono-latin-ext.woff2":     "fonts/roboto-mono-latin-ext.woff2",
		"/assets/fonts/instrument-sans-OFL.txt":         "fonts/instrument-sans-OFL.txt",
		"/assets/fonts/roboto-mono-OFL.txt":             "fonts/roboto-mono-OFL.txt",
	} {
		name := name
		pattern := "GET " + path
		if path == "/" {
			pattern = "GET /{$}"
		}
		mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
			content, err := assets.ReadFile("assets/" + name)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			typeOf := "text/html; charset=utf-8"
			if strings.HasSuffix(name, ".js") {
				typeOf = "text/javascript; charset=utf-8"
			}
			if strings.HasSuffix(name, ".css") {
				typeOf = "text/css; charset=utf-8"
			}
			if strings.HasSuffix(name, ".woff2") {
				typeOf = "font/woff2"
			}
			if strings.HasSuffix(name, ".txt") {
				typeOf = "text/plain; charset=utf-8"
			}
			w.Header().Set("Content-Type", typeOf)
			_, _ = w.Write(content)
		})
	}
	return securityHeaders(mux)
}

func mutation[Q, V any](mux *http.ServeMux, service *factory.Service, pattern string, status int, operation func(context.Context, factory.Principal, Q) (V, error)) {
	mutationPath(mux, service, pattern, status, func(ctx context.Context, p factory.Principal, q Q, r *http.Request) (V, error) {
		return operation(ctx, p, q)
	})
}

// mutationPath validates the URL subject before passing a core request through.
func mutationPath[Q, V any](mux *http.ServeMux, service *factory.Service, pattern string, status int, operation func(context.Context, factory.Principal, Q, *http.Request) (V, error)) {
	mux.HandleFunc(pattern, authenticated(service, func(w http.ResponseWriter, r *http.Request, p factory.Principal) {
		var input Q
		if !decode(w, r, &input) {
			return
		}
		output, err := operation(r.Context(), p, input, r)
		respond(w, status, output, err)
	}))
}

func authenticated(service *factory.Service, next endpoint) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if len(r.RequestURI) > 8192 {
			writeJSON(w, 414, &factory.Error{Code: factory.CodeInvalid, Message: "request URL is too long"})
			return
		}
		for key := range r.URL.Query() {
			if strings.Contains(strings.ToLower(key), "token") || strings.EqualFold(key, "authorization") {
				writeError(w, invalid("credentials are accepted only in the Authorization header"))
				return
			}
		}
		if !sameOrigin(r) {
			writeError(w, &factory.Error{Code: factory.CodeForbidden, Message: "cross-origin requests are not permitted"})
			return
		}
		parts := strings.Fields(r.Header.Get("Authorization"))
		if len(r.Header.Values("Authorization")) != 1 || len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			writeError(w, &factory.Error{Code: factory.CodeUnauthenticated, Message: "authentication required"})
			return
		}
		p, err := service.Authenticate(r.Context(), parts[1])
		if err != nil {
			writeError(w, err)
			return
		}
		next(w, r, p)
	}
}

func sameOrigin(r *http.Request) bool {
	if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
		return false
	}
	if origin := r.Header.Get("Origin"); origin != "" {
		u, err := url.Parse(origin)
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		return err == nil && u.Scheme == scheme && u.Host == r.Host && u.Path == "" && u.RawQuery == "" && u.Fragment == "" && u.User == nil
	}
	return true
}

func decode(w http.ResponseWriter, r *http.Request, destination any) bool {
	typeOf, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || typeOf != "application/json" {
		writeJSON(w, 415, &factory.Error{Code: factory.CodeInvalid, Message: "Content-Type must be application/json"})
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	decoder := json.NewDecoder(r.Body)
	var raw json.RawMessage
	if err := decoder.Decode(&raw); err != nil {
		return decodeError(w, err)
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		return decodeError(w, err)
	}
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || raw[0] != '{' {
		return decodeError(w, nil)
	}
	typed := json.NewDecoder(bytes.NewReader(raw))
	typed.DisallowUnknownFields()
	if err := typed.Decode(destination); err != nil {
		return decodeError(w, err)
	}
	return true
}

func decodeError(w http.ResponseWriter, err error) bool {
	var oversized *http.MaxBytesError
	if errors.As(err, &oversized) {
		writeJSON(w, 413, &factory.Error{Code: factory.CodeInvalid, Message: "JSON body exceeds 2 MiB"})
		return false
	}
	writeError(w, invalid("body must contain one valid JSON object with known fields"))
	return false
}

func bindID(pathID, bodyID string) error {
	if bodyID != "" && bodyID != pathID {
		return invalid("URL subject differs from request subject")
	}
	return nil
}
func invalid(message string) error {
	return &factory.Error{Code: factory.CodeInvalid, Message: message}
}
func respond(w http.ResponseWriter, status int, value any, err error) {
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, status, value)
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, err error) {
	code := factory.ErrorCodeOf(err)
	status := http.StatusInternalServerError
	message := "service operation failed"
	var details map[string]any
	var failure *factory.Error
	if errors.As(err, &failure) && code != factory.CodeInternal {
		message = failure.Message
		details = failure.Details
	}
	switch code {
	case factory.CodeInvalid:
		status = 400
	case factory.CodeUnauthenticated:
		status = 401
		message = "authentication required"
		details = nil
		w.Header().Set("WWW-Authenticate", "Bearer")
	case factory.CodeForbidden:
		status = 403
		message = "permission denied"
		details = nil
	case factory.CodeNotFound:
		status = 404
		message = "resource not found"
		details = nil
	case factory.CodeNoWork:
		status = 404
	case factory.CodeConflict, factory.CodeStale, factory.CodeUncertain:
		status = 409
	case factory.CodeNotReady, factory.CodeDeadline, factory.CodeExhausted:
		status = 422
	case factory.CodeLocked:
		status = 503
	}
	writeJSON(w, status, &factory.Error{Code: code, Message: message, Details: details})
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'self'; font-src 'self'; connect-src 'self'; img-src 'self' data:; base-uri 'none'; form-action 'self'; frame-ancestors 'none'; object-src 'none'")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
		w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		next.ServeHTTP(w, r)
	})
}
