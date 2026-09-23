package runtimeupdate

import (
	"os"
	"strconv"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestManagerAvailabilityAndRequest(t *testing.T) {
	dir := t.TempDir()
	m := New(dir)

	if m.Available() {
		t.Fatal("updater must be unavailable without Installer capability")
	}

	if err := os.WriteFile(filepath.Join(dir, "capabilities.env"), []byte(
		"schema=1
updater_available=true
target=latest
",
	), 0o600); err != nil {
		t.Fatal(err)
	}
	if !m.Available() {
		t.Fatal("expected valid capability marker")
	}

	if err := m.Request("mup_test-01", "latest"); err != nil {
		t.Fatalf("Request: %v", err)
	}
	body, err := os.ReadFile(filepath.Join(dir, "request.env"))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "schema=1
request_id=mup_test-01
target=latest
" {
		t.Fatalf("unexpected request body: %q", body)
	}

	info, err := os.Stat(filepath.Join(dir, "request.env"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("request file permissions too broad: %o", info.Mode().Perm())
	}
}

func TestValidateRequestRejectsUnboundedInputs(t *testing.T) {
	cases := []struct {
		id     string
		target string
	}{
		{"", "latest"},
		{"../../escape", "latest"},
		{strings.Repeat("a", 65), "latest"},
		{"mup_ok", "v2.3.0"},
		{"mup_ok", "ghcr.io/example/other:latest"},
	}
	for _, tc := range cases {
		if err := ValidateRequest(tc.id, tc.target); err == nil {
			t.Fatalf("ValidateRequest(%q,%q) unexpectedly succeeded", tc.id, tc.target)
		}
	}
}

func TestLastStatusIsBoundedAndValidated(t *testing.T) {
	dir := t.TempDir()
	m := New(dir)
	statusPath := filepath.Join(dir, "status.env")
	now := time.Now().Unix()

	body := "schema=1
request_id=mup_test-02
target=latest
status=rolled_back
updated_at=" +
		strconv.FormatInt(now, 10) + "
message=previous image restored
"
	if err := os.WriteFile(statusPath, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	status := m.LastStatus()
	if status == nil {
		t.Fatal("expected valid status")
	}
	if status.Status != "rolled_back" || status.RequestID != "mup_test-02" {
		t.Fatalf("unexpected status: %+v", status)
	}

	if err := os.WriteFile(statusPath, []byte(
		"schema=1
request_id=mup_test-03
target=latest
status=exec
updated_at="+strconv.FormatInt(now, 10)+"
",
	), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := m.LastStatus(); got != nil {
		t.Fatalf("unknown state unexpectedly accepted: %+v", got)
	}
}

func TestCapabilityRejectsUnknownOrOversizedData(t *testing.T) {
	dir := t.TempDir()
	m := New(dir)
	path := filepath.Join(dir, "capabilities.env")

	if err := os.WriteFile(path, []byte(
		"schema=1
updater_available=true
target=v2.3.0
",
	), 0o600); err != nil {
		t.Fatal(err)
	}
	if m.Available() {
		t.Fatal("non-latest target must not advertise updater")
	}

	if err := os.WriteFile(path, []byte(strings.Repeat("x", 513)), 0o600); err != nil {
		t.Fatal(err)
	}
	if m.Available() {
		t.Fatal("oversized capability file must be rejected")
	}
}

