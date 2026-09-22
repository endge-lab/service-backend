// Package buildinfo exposes metadata embedded into the service binary.
package buildinfo

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Version and WorkspaceSchemaVersion are set from the repository VERSION file through Go linker flags.
var (
	Version                string
	WorkspaceSchemaVersion string
)

// Metadata contains immutable versions embedded into one backend build.
type Metadata struct {
	AppVersion             string
	WorkspaceSchemaVersion int
}

// Resolve returns embedded build metadata. Reading VERSION is a local go-run
// fallback; deployed binaries always receive both values during the image build.
func Resolve(fallback string) (Metadata, error) {
	appVersion := strings.TrimSpace(Version)
	workspaceSchemaVersion := strings.TrimSpace(WorkspaceSchemaVersion)
	if appVersion == "" || workspaceSchemaVersion == "" {
		values, err := readVersionFile()
		if err != nil {
			return Metadata{}, err
		}
		if appVersion == "" {
			appVersion = values["APP_VERSION"]
		}
		if workspaceSchemaVersion == "" {
			workspaceSchemaVersion = values["WORKSPACE_SCHEMA_VERSION"]
		}
	}
	if appVersion == "" {
		appVersion = strings.TrimSpace(fallback)
	}
	if appVersion == "" {
		return Metadata{}, fmt.Errorf("APP_VERSION is required")
	}
	schemaVersion, err := strconv.Atoi(workspaceSchemaVersion)
	if err != nil || schemaVersion < 1 {
		return Metadata{}, fmt.Errorf("WORKSPACE_SCHEMA_VERSION must be a positive integer")
	}
	return Metadata{AppVersion: appVersion, WorkspaceSchemaVersion: schemaVersion}, nil
}

func readVersionFile() (map[string]string, error) {
	path, err := findVersionFile()
	if err != nil {
		return nil, err
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read VERSION: %w", err)
	}
	values := make(map[string]string, 2)
	for _, line := range strings.Split(string(contents), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		if !found || value == "" || (key != "APP_VERSION" && key != "WORKSPACE_SCHEMA_VERSION") {
			return nil, fmt.Errorf("invalid VERSION entry %q", line)
		}
		if _, exists := values[key]; exists {
			return nil, fmt.Errorf("duplicate VERSION entry %q", key)
		}
		values[key] = value
	}
	return values, nil
}

func findVersionFile() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve working directory: %w", err)
	}
	for {
		candidate := filepath.Join(dir, "VERSION")
		if _, statErr := os.Stat(candidate); statErr == nil {
			return candidate, nil
		} else if !os.IsNotExist(statErr) {
			return "", fmt.Errorf("stat VERSION: %w", statErr)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("VERSION not found from %s", dir)
		}
		dir = parent
	}
}
