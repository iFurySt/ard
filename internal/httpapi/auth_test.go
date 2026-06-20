package httpapi

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadAdminTokensFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tokens.json")
	if err := os.WriteFile(path, []byte(`{
  "version": "1",
  "tokens": [
    {"name": "reader", "token": "reader-token", "role": "reader"},
    {"name": "reviewer", "token": "reviewer-token", "role": "reviewer"}
  ]
}`), 0o600); err != nil {
		t.Fatalf("write token file: %v", err)
	}

	tokens, err := LoadAdminTokensFile(path)
	if err != nil {
		t.Fatalf("load token file: %v", err)
	}
	if len(tokens) != 2 {
		t.Fatalf("expected 2 tokens, got %d", len(tokens))
	}
	if tokens[0].Role != adminRoleReader || tokens[1].Role != adminRoleReviewer {
		t.Fatalf("unexpected roles: %#v", tokens)
	}
}

func TestNormalizeAdminTokensRejectsInvalidRole(t *testing.T) {
	_, err := NormalizeAdminTokens([]AdminToken{{Name: "bad", Token: "bad-token", Role: "owner"}})
	if err == nil {
		t.Fatal("expected invalid role error")
	}
}

func TestAdminAuthorizerReloadsTokenFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tokens.json")
	if err := os.WriteFile(path, []byte(`{
  "version": "1",
  "tokens": [
    {"name": "reader", "token": "reader-token", "role": "reader"}
  ]
}`), 0o600); err != nil {
		t.Fatalf("write token file: %v", err)
	}
	authorizer := newAdminAuthorizer(nil, path)

	if principal, ok := authorizer.authenticate("Bearer reader-token"); !ok || principal.Role != adminRoleReader {
		t.Fatalf("expected initial reader token to authenticate, got %#v %v", principal, ok)
	}

	time.Sleep(10 * time.Millisecond)
	if err := os.WriteFile(path, []byte(`{
  "version": "1",
  "tokens": [
    {"name": "publisher", "token": "publisher-token-rotated", "role": "publisher"}
  ]
}`), 0o600); err != nil {
		t.Fatalf("rotate token file: %v", err)
	}

	if _, ok := authorizer.authenticate("Bearer reader-token"); ok {
		t.Fatal("expected removed reader token to stop authenticating after reload")
	}
	if principal, ok := authorizer.authenticate("Bearer publisher-token-rotated"); !ok || principal.Role != adminRolePublisher {
		t.Fatalf("expected rotated publisher token to authenticate, got %#v %v", principal, ok)
	}
}

func TestAdminAuthorizerKeepsLastGoodTokenFileOnReloadError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tokens.json")
	if err := os.WriteFile(path, []byte(`{
  "version": "1",
  "tokens": [
    {"name": "reader", "token": "reader-token", "role": "reader"}
  ]
}`), 0o600); err != nil {
		t.Fatalf("write token file: %v", err)
	}
	authorizer := newAdminAuthorizer(nil, path)

	time.Sleep(10 * time.Millisecond)
	if err := os.WriteFile(path, []byte(`{"version": "1", "tokens": [`), 0o600); err != nil {
		t.Fatalf("write invalid token file: %v", err)
	}
	if principal, ok := authorizer.authenticate("Bearer reader-token"); !ok || principal.Role != adminRoleReader {
		t.Fatalf("expected last good reader token to remain active, got %#v %v", principal, ok)
	}
}

func TestAdminRolePermissions(t *testing.T) {
	cases := []struct {
		role       string
		permission adminPermission
		want       bool
	}{
		{role: adminRoleReader, permission: adminPermissionRead, want: true},
		{role: adminRoleReader, permission: adminPermissionPublish, want: false},
		{role: adminRolePublisher, permission: adminPermissionPublish, want: true},
		{role: adminRolePublisher, permission: adminPermissionOperate, want: false},
		{role: adminRoleReviewer, permission: adminPermissionReview, want: true},
		{role: adminRoleReviewer, permission: adminPermissionOperate, want: false},
		{role: adminRoleOperator, permission: adminPermissionOperate, want: true},
		{role: adminRoleOperator, permission: adminPermissionReview, want: false},
		{role: adminRoleAdmin, permission: adminPermissionOperate, want: true},
	}
	for _, tc := range cases {
		if got := roleAllows(tc.role, tc.permission); got != tc.want {
			t.Fatalf("roleAllows(%s, %s) = %v, want %v", tc.role, tc.permission, got, tc.want)
		}
	}
}
