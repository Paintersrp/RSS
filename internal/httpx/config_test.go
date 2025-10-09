package httpx

import (
	"net/url"
	"testing"
)

func TestRuntimeConfigSnapshotSanitizesDSN(t *testing.T) {
	cfg := RuntimeConfig{
		Service: "test",
		Database: DatabaseConfig{
			Driver: "pgx",
			DSN:    "postgres://user:secret@localhost:5432/db?sslmode=disable&password=secret&pass=foo&pwd=bar&password_file=/tmp/file&keep=this",
		},
	}

	snapshot := cfg.Snapshot()
	sanitized := snapshot.Database.DSN

	parsed, err := url.Parse(sanitized)
	if err != nil {
		t.Fatalf("failed to parse sanitized DSN: %v", err)
	}

	if parsed.User == nil {
		t.Fatalf("expected user information to be present")
	}

	if _, hasPassword := parsed.User.Password(); hasPassword {
		t.Fatalf("expected user password to be removed from DSN, got %q", sanitized)
	}

	query := parsed.Query()
	sensitiveKeys := []string{"password", "pass", "pwd", "password_file"}
	for _, key := range sensitiveKeys {
		if _, exists := query[key]; exists {
			t.Fatalf("expected sensitive query parameter %q to be removed", key)
		}
	}

	if got := query.Get("keep"); got != "this" {
		t.Fatalf("expected non-sensitive query parameter to remain, got %q", got)
	}
}
