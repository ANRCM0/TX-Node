package service

import (
	"os"
	"strings"
	"testing"

	"github.com/PaiMonCai/TX-Node/internal/config"
)

func TestOpsNetworkTargetValidation(t *testing.T) {
	if got, err := opsNetworkTarget(map[string]interface{}{"target": "node.example.com"}); err != nil || got != "node.example.com" {
		t.Fatalf("unexpected valid target result: %q, %v", got, err)
	}
	for _, target := range []string{"", "https://example.com", "example.com/path", "bad host"} {
		if _, err := opsNetworkTarget(map[string]interface{}{"target": target}); err == nil {
			t.Fatalf("expected target %q to be rejected", target)
		}
	}
}

func TestOpsPortValidation(t *testing.T) {
	for _, value := range []interface{}{1, 443, 65535} {
		if _, err := opsPort(map[string]interface{}{"port": value}); err != nil {
			t.Fatalf("expected port %v to be accepted: %v", value, err)
		}
	}
	for _, value := range []interface{}{0, 65536, "abc"} {
		if _, err := opsPort(map[string]interface{}{"port": value}); err == nil {
			t.Fatalf("expected port %v to be rejected", value)
		}
	}
}

func TestTailApplicationLogIsBoundedAndRedacted(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "tx-node-*.log")
	if err != nil {
		t.Fatal(err)
	}
	path := file.Name()

	for i := 0; i < 10; i++ {
		if _, err := file.WriteString("12:00:00 INFO [core] harmless line\n"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := file.WriteString("12:00:01 INFO [core] token=super-secret password=hunter2 uuid=00000000-0000-0000-0000-000000000001\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString("12:00:02 INFO [core] Authorization: Bearer abc.def.ghi\n"); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	s := &Service{cfg: &config.Config{Log: config.LogConfig{Output: path}}}
	result, err := s.tailApplicationLog(map[string]interface{}{
		"source":    "application",
		"lines":     3,
		"max_bytes": 4096,
	})
	if err != nil {
		t.Fatalf("tailApplicationLog: %v", err)
	}

	content, _ := result["content"].(string)
	if strings.Contains(content, "super-secret") ||
		strings.Contains(content, "hunter2") ||
		strings.Contains(content, "abc.def.ghi") ||
		strings.Contains(content, "00000000-0000-0000-0000-000000000001") {
		t.Fatalf("sensitive data was not redacted: %s", content)
	}
	if !strings.Contains(content, "[REDACTED]") {
		t.Fatalf("expected redaction marker, got: %s", content)
	}
	if got := len(strings.Split(content, "\n")); got > 3 {
		t.Fatalf("expected at most 3 lines, got %d", got)
	}
}

func TestTailApplicationLogRejectsStdoutAndOversizedBounds(t *testing.T) {
	s := &Service{cfg: &config.Config{Log: config.LogConfig{Output: "stdout"}}}
	if _, err := s.tailApplicationLog(map[string]interface{}{"source": "application", "lines": 10}); err == nil {
		t.Fatal("expected stdout log source to be unavailable")
	}

	if _, err := opsBoundedInt(map[string]interface{}{"lines": 201}, "lines", 100, 1, 200); err == nil {
		t.Fatal("expected oversized line count to be rejected")
	}
}
