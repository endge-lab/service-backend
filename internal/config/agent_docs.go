package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

// loadAgentDocsURL читает AGENT_DOCS_URL — корень сайта документации с llms-разделами
// (llms.txt, llms-full/*.txt). Пустое значение отключает ресурсы документации в MCP.
func loadAgentDocsURL() (string, error) {
	return normalizeAgentDocsURL(os.Getenv("AGENT_DOCS_URL"))
}

func normalizeAgentDocsURL(raw string) (string, error) {
	value := strings.TrimRight(strings.TrimSpace(raw), "/")
	if value == "" {
		return "", nil
	}
	parsed, err := url.Parse(value)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("AGENT_DOCS_URL must be an absolute http(s) URL without query and fragment")
	}
	return value, nil
}
