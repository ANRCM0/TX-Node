package runtimeupdate

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const DefaultDir = "/run/txnode-update"

var requestIDPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,64}$`)

var allowedStatuses = map[string]bool{
	"accepted":    true,
	"running":     true,
	"succeeded":   true,
	"failed":      true,
	"rolled_back": true,
}

type Status struct {
	RequestID string `json:"request_id"`
	Target    string `json:"target"`
	Status    string `json:"status"`
	UpdatedAt int64  `json:"updated_at"`
	Message   string `json:"message,omitempty"`
}

type Manager struct {
	dir string
}

func New(dir string) *Manager {
	if strings.TrimSpace(dir) == "" {
		dir = DefaultDir
	}
	return &Manager{dir: dir}
}

func (m *Manager) capabilityPath() string { return filepath.Join(m.dir, "capabilities.env") }
func (m *Manager) requestPath() string    { return filepath.Join(m.dir, "request.env") }
func (m *Manager) statusPath() string     { return filepath.Join(m.dir, "status.env") }

func (m *Manager) Available() bool {
	values, err := parseEnvFile(m.capabilityPath(), 8, 512)
	if err != nil {
		return false
	}
	return values["schema"] == "1" &&
		values["updater_available"] == "true" &&
		values["target"] == "latest"
}

func ValidateRequest(requestID, target string) error {
	if !requestIDPattern.MatchString(requestID) {
		return errors.New("invalid request_id")
	}
	if target != "latest" {
		return errors.New("unsupported update target")
	}
	return nil
}

func (m *Manager) Request(requestID, target string) error {
	if err := ValidateRequest(requestID, target); err != nil {
		return err
	}
	if !m.Available() {
		return errors.New("runtime updater unavailable")
	}

	if err := os.MkdirAll(m.dir, 0o700); err != nil {
		return fmt.Errorf("create update directory: %w", err)
	}

	body := "schema=1
request_id=" + requestID + "
target=latest
"
	tmp, err := os.CreateTemp(m.dir, ".request-*")
	if err != nil {
		return fmt.Errorf("create update request: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return fmt.Errorf("chmod update request: %w", err)
	}
	if _, err := tmp.WriteString(body); err != nil {
		tmp.Close()
		return fmt.Errorf("write update request: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("sync update request: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close update request: %w", err)
	}
	if err := os.Rename(tmpName, m.requestPath()); err != nil {
		return fmt.Errorf("publish update request: %w", err)
	}
	return nil
}

func (m *Manager) LastStatus() *Status {
	values, err := parseEnvFile(m.statusPath(), 10, 2048)
	if err != nil || values["schema"] != "1" {
		return nil
	}
	requestID := values["request_id"]
	target := values["target"]
	state := values["status"]
	if !requestIDPattern.MatchString(requestID) || target != "latest" || !allowedStatuses[state] {
		return nil
	}

	updatedAt, err := strconv.ParseInt(values["updated_at"], 10, 64)
	if err != nil || updatedAt <= 0 || updatedAt > time.Now().Add(24*time.Hour).Unix() {
		return nil
	}
	message := values["message"]
	if len(message) > 160 {
		message = message[:160]
	}

	return &Status{
		RequestID: requestID,
		Target:    target,
		Status:    state,
		UpdatedAt: updatedAt,
		Message:   message,
	}
}

func parseEnvFile(path string, maxLines, maxBytes int) (map[string]string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.Size() < 0 || info.Size() > int64(maxBytes) {
		return nil, errors.New("metadata file exceeds size bound")
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	values := make(map[string]string)
	scanner := bufio.NewScanner(f)
	lines := 0
	for scanner.Scan() {
		lines++
		if lines > maxLines {
			return nil, errors.New("metadata file exceeds line bound")
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok || key == "" || strings.ContainsAny(key, " 	
") {
			return nil, errors.New("invalid metadata line")
		}
		if _, exists := values[key]; exists {
			return nil, errors.New("duplicate metadata key")
		}
		values[key] = value
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return values, nil
}
