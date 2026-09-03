package config

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"strings"

	kitconfig "github.com/endge-lab/service-kit-go/config"
)

func loadHTTPBasePath(base *kitconfig.ServiceConfig) (string, error) {
	basePath, err := normalizeHTTPBasePath(os.Getenv("HTTP_BASE_PATH"))
	if err != nil {
		return "", err
	}
	if err := validatePublicURLBasePath(base.App.PublicURL, basePath); err != nil {
		return "", err
	}
	return basePath, nil
}

func normalizeHTTPBasePath(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" || value == "/" {
		return "", nil
	}
	if !strings.HasPrefix(value, "/") {
		return "", fmt.Errorf("HTTP_BASE_PATH must start with /")
	}
	if strings.ContainsAny(value, "?#\\%") || path.Clean(value) != value {
		return "", fmt.Errorf("HTTP_BASE_PATH must be a clean URL path without a trailing slash")
	}
	return value, nil
}

func validatePublicURLBasePath(publicURL, basePath string) error {
	if basePath == "" {
		return nil
	}
	parsed, err := url.Parse(publicURL)
	if err != nil {
		return fmt.Errorf("PUBLIC_URL must be a valid URL: %w", err)
	}
	if strings.TrimRight(parsed.Path, "/") != basePath {
		return fmt.Errorf("PUBLIC_URL path must match HTTP_BASE_PATH")
	}
	return nil
}
