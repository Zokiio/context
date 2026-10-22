package factory

import (
	"context"
	"database/sql"
	"errors"
	"sort"
	"strings"
	"time"
)

type queryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func (s *Service) Bootstrap(ctx context.Context, request BootstrapRequest) (BootstrapResult, error) {
	if err := validateUsername(request.Username); err != nil {
		return BootstrapResult{}, err
	}
	s.state.mutations.Lock()
	defer s.state.mutations.Unlock()
	tx, err := s.state.db.BeginTx(ctx, nil)
	if err != nil {
		return BootstrapResult{}, fault(CodeInternal, "start bootstrap", err)
	}
	defer tx.Rollback()
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM users`).Scan(&count); err != nil {
		return BootstrapResult{}, fault(CodeInternal, "inspect installation", err)
	}
	if count != 0 {
		return BootstrapResult{}, fault(CodeConflict, "installation is already bootstrapped", nil)
	}
	userID, err := newID("usr_")
	if err != nil {
		return BootstrapResult{}, fault(CodeInternal, "create user identity", err)
	}
	tokenID, err := newID("tok_")
	if err != nil {
		return BootstrapResult{}, fault(CodeInternal, "create token identity", err)
	}
	token, err := newToken()
	if err != nil {
		return BootstrapResult{}, fault(CodeInternal, "create bootstrap token", err)
	}
	now := s.state.now().UTC()
	if _, err := tx.ExecContext(ctx, `INSERT INTO users(id, username, kind, installation_admin, disabled, created_at) VALUES(?,?,?,?,0,?)`, userID, strings.TrimSpace(request.Username), UserKindHuman, 1, formatTime(now)); err != nil {
		return BootstrapResult{}, fault(CodeInternal, "create bootstrap user", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO tokens(id, user_id, token_hash, label, created_at) VALUES(?,?,?,?,?)`, tokenID, userID, tokenHash(token), "bootstrap", formatTime(now)); err != nil {
		return BootstrapResult{}, fault(CodeInternal, "create bootstrap token", err)
	}
	if err := tx.Commit(); err != nil {
		return BootstrapResult{}, fault(CodeInternal, "commit bootstrap", err)
	}
	return BootstrapResult{User: User{ID: UserID(userID), Username: strings.TrimSpace(request.Username), Kind: UserKindHuman, InstallationAdmin: true, CreatedAt: now}, Token: token}, nil
}

func (s *Service) Authenticate(ctx context.Context, token string) (Principal, error) {
	if token == "" {
		return Principal{}, fault(CodeUnauthenticated, "bearer token is required", nil)
	}
	var principal Principal
	var disabled int
	var expiry, revoked sql.NullString
	err := s.state.db.QueryRowContext(ctx, `
SELECT u.id, u.username, u.kind, u.installation_admin, u.disabled, t.expires_at, t.revoked_at
FROM tokens t JOIN users u ON u.id=t.user_id WHERE t.token_hash=?`, tokenHash(token)).Scan(
		&principal.UserID, &principal.Username, &principal.Kind, &principal.InstallationAdmin, &disabled, &expiry, &revoked,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Principal{}, fault(CodeUnauthenticated, "invalid bearer token", nil)
	}
	if err != nil {
		return Principal{}, fault(CodeInternal, "authenticate token", err)
	}
	if disabled != 0 || revoked.Valid {
		return Principal{}, fault(CodeUnauthenticated, "credential is inactive", nil)
	}
	if expiry.Valid {
		expires, err := parseTime(expiry.String)
		if err != nil || !s.state.now().Before(expires) {
			return Principal{}, fault(CodeUnauthenticated, "credential is expired", nil)
		}
	}
	return principal, nil
}

func (s *Service) CreateUser(ctx context.Context, actor Principal, request CreateUserRequest) (User, error) {
	if err := validateUsername(request.Username); err != nil {
		return User{}, err
	}
	if request.Kind != UserKindHuman && request.Kind != UserKindWorker {
		return User{}, fault(CodeInvalid, "kind must be human or worker", nil)
	}
	s.state.mutations.Lock()
	defer s.state.mutations.Unlock()
	tx, err := s.state.db.BeginTx(ctx, nil)
	if err != nil {
		return User{}, fault(CodeInternal, "start user creation", err)
	}
	defer tx.Rollback()
	current, err := requireInstallationAdmin(ctx, tx, actor)
	if err != nil {
		return User{}, err
	}
	if saved, ok, hash, err := receipt[User](ctx, tx, current.UserID, "create_user", request.RequestID, request); err != nil || ok {
		return saved, err
	} else {
		id, err := newID("usr_")
		if err != nil {
			return User{}, fault(CodeInternal, "create user identity", err)
		}
		now := s.state.now().UTC()
		user := User{ID: UserID(id), Username: strings.TrimSpace(request.Username), Kind: request.Kind, InstallationAdmin: request.InstallationAdmin, CreatedAt: now}
		if _, err := tx.ExecContext(ctx, `INSERT INTO users(id, username, kind, installation_admin, disabled, created_at) VALUES(?,?,?,?,0,?)`, id, user.Username, user.Kind, boolInt(user.InstallationAdmin), formatTime(now)); err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "unique") {
				return User{}, fault(CodeConflict, "username already exists", err)
			}
			return User{}, fault(CodeInternal, "create user", err)
		}
		if err := saveReceipt(ctx, tx, current.UserID, "create_user", request.RequestID, hash, user, false, now); err != nil {
			return User{}, err
		}
		if err := tx.Commit(); err != nil {
			return User{}, fault(CodeInternal, "commit user creation", err)
		}
		return user, nil
	}
}

func (s *Service) ListUsers(ctx context.Context, actor Principal) ([]User, error) {
	if _, err := requireInstallationAdmin(ctx, s.state.db, actor); err != nil {
		return nil, err
	}
	rows, err := s.state.db.QueryContext(ctx, `SELECT id, username, kind, installation_admin, disabled, created_at FROM users ORDER BY username COLLATE NOCASE`)
	if err != nil {
		return nil, fault(CodeInternal, "list users", err)
	}
	defer rows.Close()
	users := []User{}
	for rows.Next() {
		var user User
		var created string
		if err := rows.Scan(&user.ID, &user.Username, &user.Kind, &user.InstallationAdmin, &user.Disabled, &created); err != nil {
			return nil, fault(CodeInternal, "read user", err)
		}
		user.CreatedAt, _ = parseTime(created)
		users = append(users, user)
	}
	return users, rows.Err()
}

func (s *Service) IssueToken(ctx context.Context, actor Principal, request IssueTokenRequest) (IssuedToken, error) {
	if request.UserID == "" || strings.TrimSpace(request.Label) == "" {
		return IssuedToken{}, fault(CodeInvalid, "userId and label are required", nil)
	}
	if request.ExpiresAt != nil && !s.state.now().Before(*request.ExpiresAt) {
		return IssuedToken{}, fault(CodeInvalid, "expiresAt must be in the future", nil)
	}
	s.state.mutations.Lock()
	defer s.state.mutations.Unlock()
	tx, err := s.state.db.BeginTx(ctx, nil)
	if err != nil {
		return IssuedToken{}, fault(CodeInternal, "start token issue", err)
	}
	defer tx.Rollback()
	current, err := requireInstallationAdmin(ctx, tx, actor)
	if err != nil {
		return IssuedToken{}, err
	}
	if _, ok, hash, err := receipt[IssuedToken](ctx, tx, current.UserID, "issue_token", request.RequestID, request); err != nil || ok {
		return IssuedToken{}, err
	} else {
		var disabled int
		if err := tx.QueryRowContext(ctx, `SELECT disabled FROM users WHERE id=?`, request.UserID).Scan(&disabled); errors.Is(err, sql.ErrNoRows) {
			return IssuedToken{}, fault(CodeNotFound, "user not found", nil)
		} else if err != nil {
			return IssuedToken{}, fault(CodeInternal, "read token subject", err)
		} else if disabled != 0 {
			return IssuedToken{}, fault(CodeConflict, "cannot issue a token to a disabled user", nil)
		}
		id, err := newID("tok_")
		if err != nil {
			return IssuedToken{}, fault(CodeInternal, "create token identity", err)
		}
		plaintext, err := newToken()
		if err != nil {
			return IssuedToken{}, fault(CodeInternal, "create token", err)
		}
		now := s.state.now().UTC()
		var expiry any
		if request.ExpiresAt != nil {
			expiry = formatTime(*request.ExpiresAt)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO tokens(id, user_id, token_hash, label, expires_at, created_at) VALUES(?,?,?,?,?,?)`, id, request.UserID, tokenHash(plaintext), strings.TrimSpace(request.Label), expiry, formatTime(now)); err != nil {
			return IssuedToken{}, fault(CodeInternal, "issue token", err)
		}
		issued := IssuedToken{ID: ID(id), UserID: request.UserID, Label: strings.TrimSpace(request.Label), Token: plaintext, ExpiresAt: request.ExpiresAt, CreatedAt: now}
		if err := saveReceipt(ctx, tx, current.UserID, "issue_token", request.RequestID, hash, nil, true, now); err != nil {
			return IssuedToken{}, err
		}
		if err := tx.Commit(); err != nil {
			return IssuedToken{}, fault(CodeInternal, "commit token issue", err)
		}
		return issued, nil
	}
}

func (s *Service) SetUserDisabled(ctx context.Context, actor Principal, request SetUserDisabledRequest) (User, error) {
	if request.UserID == "" {
		return User{}, fault(CodeInvalid, "userId is required", nil)
	}
	s.state.mutations.Lock()
	defer s.state.mutations.Unlock()
	tx, err := s.state.db.BeginTx(ctx, nil)
	if err != nil {
		return User{}, fault(CodeInternal, "start user update", err)
	}
	defer tx.Rollback()
	current, err := requireInstallationAdmin(ctx, tx, actor)
	if err != nil {
		return User{}, err
	}
	if saved, ok, hash, err := receipt[User](ctx, tx, current.UserID, "set_user_disabled", request.RequestID, request); err != nil || ok {
		return saved, err
	} else {
		user, err := readUser(ctx, tx, request.UserID)
		if err != nil {
			return User{}, err
		}
		if request.Disabled && user.InstallationAdmin && !user.Disabled {
			var otherCredentials int
			now := formatTime(s.state.now())
			if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM tokens t JOIN users u ON u.id=t.user_id WHERE u.installation_admin=1 AND u.disabled=0 AND u.id<>? AND t.revoked_at IS NULL AND (t.expires_at IS NULL OR t.expires_at>?)`, user.ID, now).Scan(&otherCredentials); err != nil {
				return User{}, fault(CodeInternal, "count other administrator credentials", err)
			}
			if otherCredentials == 0 {
				return User{}, fault(CodeConflict, "cannot disable the last enabled installation administrator", nil)
			}
		}
		if _, err := tx.ExecContext(ctx, `UPDATE users SET disabled=? WHERE id=?`, boolInt(request.Disabled), request.UserID); err != nil {
			return User{}, fault(CodeInternal, "update user", err)
		}
		user.Disabled = request.Disabled
		if err := saveReceipt(ctx, tx, current.UserID, "set_user_disabled", request.RequestID, hash, user, false, s.state.now()); err != nil {
			return User{}, err
		}
		if err := tx.Commit(); err != nil {
			return User{}, fault(CodeInternal, "commit user update", err)
		}
		return user, nil
	}
}

func (s *Service) RevokeToken(ctx context.Context, actor Principal, request RevokeTokenRequest) error {
	if request.TokenID == "" {
		return fault(CodeInvalid, "tokenId is required", nil)
	}
	s.state.mutations.Lock()
	defer s.state.mutations.Unlock()
	tx, err := s.state.db.BeginTx(ctx, nil)
	if err != nil {
		return fault(CodeInternal, "start token revocation", err)
	}
	defer tx.Rollback()
	current, err := requireInstallationAdmin(ctx, tx, actor)
	if err != nil {
		return err
	}
	if _, ok, hash, err := receipt[struct{}](ctx, tx, current.UserID, "revoke_token", request.RequestID, request); err != nil {
		return err
	} else if ok {
		return nil
	} else {
		var userID UserID
		var revoked sql.NullString
		var isAdmin, disabled int
		if err := tx.QueryRowContext(ctx, `SELECT t.user_id, t.revoked_at, u.installation_admin, u.disabled FROM tokens t JOIN users u ON u.id=t.user_id WHERE t.id=?`, request.TokenID).Scan(&userID, &revoked, &isAdmin, &disabled); errors.Is(err, sql.ErrNoRows) {
			return fault(CodeNotFound, "token not found", nil)
		} else if err != nil {
			return fault(CodeInternal, "read token", err)
		}
		if !revoked.Valid && isAdmin != 0 && disabled == 0 {
			var usable int
			now := formatTime(s.state.now())
			if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM tokens WHERE user_id=? AND revoked_at IS NULL AND (expires_at IS NULL OR expires_at>?)`, userID, now).Scan(&usable); err != nil {
				return fault(CodeInternal, "count administrator credentials", err)
			}
			var otherCredentials int
			if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM tokens t JOIN users u ON u.id=t.user_id WHERE u.installation_admin=1 AND u.disabled=0 AND u.id<>? AND t.revoked_at IS NULL AND (t.expires_at IS NULL OR t.expires_at>?)`, userID, now).Scan(&otherCredentials); err != nil {
				return fault(CodeInternal, "count other administrator credentials", err)
			}
			if usable <= 1 && otherCredentials == 0 {
				return fault(CodeConflict, "cannot revoke the last credential of the last enabled installation administrator", nil)
			}
		}
		now := s.state.now().UTC()
		if _, err := tx.ExecContext(ctx, `UPDATE tokens SET revoked_at=COALESCE(revoked_at, ?) WHERE id=?`, formatTime(now), request.TokenID); err != nil {
			return fault(CodeInternal, "revoke token", err)
		}
		if err := saveReceipt(ctx, tx, current.UserID, "revoke_token", request.RequestID, hash, struct{}{}, false, now); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return fault(CodeInternal, "commit token revocation", err)
		}
		return nil
	}
}

func (s *Service) GrantProject(ctx context.Context, actor Principal, request GrantProjectRequest) (Grant, error) {
	if request.ProjectID == "" || request.UserID == "" {
		return Grant{}, fault(CodeInvalid, "projectId and userId are required", nil)
	}
	capabilities, err := normalizeCapabilities(request.Capabilities)
	if err != nil {
		return Grant{}, err
	}
	request.Capabilities = capabilities
	s.state.mutations.Lock()
	defer s.state.mutations.Unlock()
	tx, err := s.state.db.BeginTx(ctx, nil)
	if err != nil {
		return Grant{}, fault(CodeInternal, "start grant update", err)
	}
	defer tx.Rollback()
	current, err := authorizeProject(ctx, tx, actor, request.ProjectID, CapabilityProjectAdmin)
	if err != nil {
		return Grant{}, err
	}
	if saved, ok, hash, err := receipt[Grant](ctx, tx, current.UserID, "grant_project", request.RequestID, request); err != nil || ok {
		return saved, err
	} else {
		if _, err := readUser(ctx, tx, request.UserID); err != nil {
			return Grant{}, err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM project_grants WHERE project_id=? AND user_id=?`, request.ProjectID, request.UserID); err != nil {
			return Grant{}, fault(CodeInternal, "replace project grant", err)
		}
		for _, capability := range capabilities {
			if _, err := tx.ExecContext(ctx, `INSERT INTO project_grants(project_id, user_id, capability) VALUES(?,?,?)`, request.ProjectID, request.UserID, capability); err != nil {
				return Grant{}, fault(CodeInternal, "save project capability", err)
			}
		}
		grant := Grant{ProjectID: request.ProjectID, UserID: request.UserID, Capabilities: capabilities}
		now := s.state.now().UTC()
		if err := saveReceipt(ctx, tx, current.UserID, "grant_project", request.RequestID, hash, grant, false, now); err != nil {
			return Grant{}, err
		}
		if err := tx.Commit(); err != nil {
			return Grant{}, fault(CodeInternal, "commit project grant", err)
		}
		return grant, nil
	}
}

func (s *Service) ListProjectGrants(ctx context.Context, actor Principal, projectID ProjectID) ([]Grant, error) {
	if _, err := authorizeProject(ctx, s.state.db, actor, projectID, CapabilityProjectAdmin); err != nil {
		return nil, err
	}
	rows, err := s.state.db.QueryContext(ctx, `SELECT user_id, capability FROM project_grants WHERE project_id=? ORDER BY user_id, capability`, projectID)
	if err != nil {
		return nil, fault(CodeInternal, "list project grants", err)
	}
	defer rows.Close()
	byUser := map[UserID][]Capability{}
	order := []UserID{}
	for rows.Next() {
		var userID UserID
		var capability Capability
		if err := rows.Scan(&userID, &capability); err != nil {
			return nil, fault(CodeInternal, "read project grant", err)
		}
		if _, exists := byUser[userID]; !exists {
			order = append(order, userID)
		}
		byUser[userID] = append(byUser[userID], capability)
	}
	grants := make([]Grant, 0, len(order))
	for _, userID := range order {
		grants = append(grants, Grant{ProjectID: projectID, UserID: userID, Capabilities: byUser[userID]})
	}
	return grants, rows.Err()
}

func (s *Service) Capabilities(ctx context.Context, actor Principal, projectID ProjectID) ([]Capability, error) {
	current, err := requirePrincipal(ctx, s.state.db, actor)
	if err != nil {
		return nil, err
	}
	if current.InstallationAdmin {
		return allCapabilities(), nil
	}
	capabilities, err := readCapabilities(ctx, s.state.db, current.UserID, projectID)
	if err != nil {
		return nil, err
	}
	if !hasCapability(capabilities, CapabilityRead) {
		return nil, fault(CodeNotFound, "project not found", nil)
	}
	return capabilities, nil
}

func validateUsername(username string) error {
	username = strings.TrimSpace(username)
	if username == "" || len(username) > 100 {
		return fault(CodeInvalid, "username must contain 1 to 100 characters", nil)
	}
	return nil
}

func readUser(ctx context.Context, q queryer, userID UserID) (User, error) {
	var user User
	var created string
	err := q.QueryRowContext(ctx, `SELECT id, username, kind, installation_admin, disabled, created_at FROM users WHERE id=?`, userID).Scan(&user.ID, &user.Username, &user.Kind, &user.InstallationAdmin, &user.Disabled, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, fault(CodeNotFound, "user not found", nil)
	}
	if err != nil {
		return User{}, fault(CodeInternal, "read user", err)
	}
	user.CreatedAt, _ = parseTime(created)
	return user, nil
}

func requirePrincipal(ctx context.Context, q queryer, actor Principal) (Principal, error) {
	if actor.UserID == "" {
		return Principal{}, fault(CodeUnauthenticated, "authenticated principal is required", nil)
	}
	var current Principal
	var disabled int
	err := q.QueryRowContext(ctx, `SELECT id, username, kind, installation_admin, disabled FROM users WHERE id=?`, actor.UserID).Scan(&current.UserID, &current.Username, &current.Kind, &current.InstallationAdmin, &disabled)
	if errors.Is(err, sql.ErrNoRows) || disabled != 0 {
		return Principal{}, fault(CodeUnauthenticated, "principal is inactive", nil)
	}
	if err != nil {
		return Principal{}, fault(CodeInternal, "read principal", err)
	}
	return current, nil
}

func requireInstallationAdmin(ctx context.Context, q queryer, actor Principal) (Principal, error) {
	current, err := requirePrincipal(ctx, q, actor)
	if err != nil {
		return Principal{}, err
	}
	if !current.InstallationAdmin {
		return Principal{}, fault(CodeForbidden, "installation administration is required", nil)
	}
	return current, nil
}

func authorizeProject(ctx context.Context, q queryer, actor Principal, projectID ProjectID, needed Capability) (Principal, error) {
	current, err := requirePrincipal(ctx, q, actor)
	if err != nil {
		return Principal{}, err
	}
	var exists int
	if err := q.QueryRowContext(ctx, `SELECT count(*) FROM projects WHERE id=?`, projectID).Scan(&exists); err != nil {
		return Principal{}, fault(CodeInternal, "read project", err)
	}
	if exists == 0 {
		return Principal{}, fault(CodeNotFound, "project not found", nil)
	}
	if current.InstallationAdmin {
		return current, nil
	}
	capabilities, err := readCapabilities(ctx, q, current.UserID, projectID)
	if err != nil {
		return Principal{}, err
	}
	if !hasCapability(capabilities, CapabilityRead) {
		return Principal{}, fault(CodeNotFound, "project not found", nil)
	}
	if !hasCapability(capabilities, needed) {
		return Principal{}, fault(CodeForbidden, "project capability is required", nil)
	}
	return current, nil
}

func readCapabilities(ctx context.Context, q queryer, userID UserID, projectID ProjectID) ([]Capability, error) {
	rows, err := q.QueryContext(ctx, `SELECT capability FROM project_grants WHERE user_id=? AND project_id=? ORDER BY capability`, userID, projectID)
	if err != nil {
		return nil, fault(CodeInternal, "read project capabilities", err)
	}
	defer rows.Close()
	result := []Capability{}
	for rows.Next() {
		var capability Capability
		if err := rows.Scan(&capability); err != nil {
			return nil, fault(CodeInternal, "read project capability", err)
		}
		result = append(result, capability)
	}
	return result, rows.Err()
}

func normalizeCapabilities(input []Capability) ([]Capability, error) {
	allowed := map[Capability]bool{}
	for _, capability := range allCapabilities() {
		allowed[capability] = true
	}
	seen := map[Capability]bool{}
	result := []Capability{}
	for _, capability := range input {
		if !allowed[capability] {
			return nil, fault(CodeInvalid, "unknown project capability", nil)
		}
		if !seen[capability] {
			seen[capability] = true
			result = append(result, capability)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result, nil
}

func allCapabilities() []Capability {
	return []Capability{CapabilityRead, CapabilityCoordinate, CapabilityVerify, CapabilityReview, CapabilityAccept, CapabilityProjectAdmin, CapabilityImplementWorker, CapabilityVerifyWorker}
}

func hasCapability(capabilities []Capability, wanted Capability) bool {
	for _, capability := range capabilities {
		if capability == wanted {
			return true
		}
	}
	return false
}

func future(value *time.Time, now time.Time) bool {
	return value == nil || now.Before(*value)
}
