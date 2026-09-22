package service

import "testing"

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
