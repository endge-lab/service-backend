package mcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var docsRequests atomic.Int64

// newDocsServer раздаёт llms-файлы под /endge-docs, как сайт документации с base.
func newDocsServer(t *testing.T) *httptest.Server {
	t.Helper()
	files := map[string]string{
		"/endge-docs/llms-agent.txt":                         "# Endge: ядро для ИИ-агента\nправила",
		"/endge-docs/llms.txt":                               "# Endge\n- [Токены](/agents/tokens.md)",
		"/endge-docs/agents/index.md":                        "# Endge для ИИ-агентов",
		"/endge-docs/agents/mcp.md":                          "# MCP",
		"/endge-docs/agents/errors.md":                       "# Коды ошибок",
		"/endge-docs/agents/scenarios.md":                    "# Сценарии",
		"/endge-docs/llms-manifest.json":                     `{"files":[]}`,
		"/endge-docs/llms-full/api/facets.txt":               "# API: фасеты\nPOST /api/v1/facets",
		"/endge-docs/llms-full/reference/contracts/core.txt": "# Контракты core",
		"/endge-docs/llms-full/agents.txt":                   "# Раздел агентов",
		"/endge-docs/reference/query.md":                     "# Query",
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		docsRequests.Add(1)
		switch r.URL.Path {
		case "/endge-docs/redirect.md":
			http.Redirect(w, r, "https://example.com/evil.md", http.StatusFound)
			return
		case "/endge-docs/spa.md":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte("<!doctype html>"))
			return
		case "/endge-docs/broken.md":
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		body, ok := files[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server
}

func readResource(t *testing.T, session *mcp.ClientSession, uri string) (*mcp.ReadResourceResult, error) {
	t.Helper()
	return session.ReadResource(context.Background(), &mcp.ReadResourceParams{URI: uri})
}

func TestMCPResourcesServeDocumentationText(t *testing.T) {
	h := newHarness(t)
	session := h.connect(t, "read")
	base := h.docs.URL + "/endge-docs/"
	resources, err := session.ListResources(context.Background(), nil)
	if err != nil || len(resources.Resources) != len(agentDocs) || resources.Resources[0].Name != "llms-agent" || resources.Resources[0].URI != base+"llms-agent.txt" {
		t.Fatalf("resources must start with the agent core: %+v %v", resources, err)
	}
	templates, err := session.ListResourceTemplates(context.Background(), nil)
	if err != nil || len(templates.ResourceTemplates) != len(agentDocTemplates) {
		t.Fatalf("resource templates: %+v %v", templates, err)
	}
	cases := map[string]string{
		base + "llms-agent.txt":                         "ядро для ИИ-агента",
		base + "agents/index.md":                        "Endge для ИИ-агентов",
		base + "llms-manifest.json":                     `"files"`,
		base + "llms-full/api/facets.txt":               "POST /api/v1/facets",
		base + "llms-full/reference/contracts/core.txt": "Контракты core",
		base + "llms-full/agents.txt":                   "Раздел агентов",
		base + "reference/query.md":                     "# Query",
	}
	for uri, want := range cases {
		read, err := readResource(t, session, uri)
		if err != nil || !strings.Contains(read.Contents[0].Text, want) {
			t.Fatalf("read %s: %+v %v", uri, read, err)
		}
	}
	if read, _ := readResource(t, session, base+"reference/query.md"); read.Contents[0].MIMEType != "text/markdown" {
		t.Fatalf("page mime type: %+v", read.Contents[0])
	}
	for name, uri := range map[string]string{
		"missing page":       base + "agents/missing.md",
		"path traversal":     base + "agents/../../secret.md",
		"other extension":    base + "assets/app.js",
		"outside prefix":     "https://example.com/llms.txt",
		"html fallback":      base + "spa.md",
		"redirect elsewhere": base + "redirect.md",
		"server error":       base + "broken.md",
	} {
		if _, err := readResource(t, session, uri); err == nil {
			t.Fatalf("%s must fail: %s", name, uri)
		}
	}
	_, err = readResource(t, session, base+"broken.md")
	if err == nil || !strings.Contains(err.Error(), "HTTP 502") || !strings.Contains(err.Error(), base+"broken.md") {
		t.Fatalf("network error must name the status and the link: %v", err)
	}
}

func TestDocsSourceCachesAndTruncates(t *testing.T) {
	server := newDocsServer(t)
	source := newDocsSource(server.URL+"/endge-docs", "test")
	ctx := context.Background()
	uri := source.uri("llms-agent.txt")
	before := docsRequests.Load()
	for range 3 {
		if text, mimeType, err := source.fetch(ctx, uri); err != nil || mimeType != "text/plain" || !strings.HasPrefix(text, "# Endge") {
			t.Fatalf("fetch: %q %q %v", text, mimeType, err)
		}
	}
	if got := docsRequests.Load() - before; got != 1 {
		t.Fatalf("cached file must be fetched once, got %d requests", got)
	}
	// После TTL файл запрашивается снова.
	now := time.Now()
	source.now = func() time.Time { return now.Add(docsCacheTTL + time.Second) }
	if _, _, err := source.fetch(ctx, uri); err != nil || docsRequests.Load()-before != 2 {
		t.Fatalf("expired entry must be refetched: %v", err)
	}

	source.maxBytes = 30
	text := source.truncate(uri, []byte("строка один\nстрока два\nстрока три\n"))
	if !strings.HasPrefix(text, "строка один\n\n\n[Ресурс обрезан") || !strings.Contains(text, "Полный файл: "+uri) || !strings.Contains(text, source.uri("llms-full/api/{group}.txt")) {
		t.Fatalf("truncated text must say where to continue: %q", text)
	}
	source.maxBytes = docsMaxBytes

	source.maxCache = 10
	source.store("a", docsEntry{text: "123456", expires: now.Add(time.Hour)})
	source.store("b", docsEntry{text: "1234567", expires: now.Add(2 * time.Hour)})
	if _, ok := source.cache["a"]; ok || source.cachedBytes != 7 {
		t.Fatalf("cache must evict the oldest entry: %v %d", source.cache, source.cachedBytes)
	}
}

func TestDocsSourceAllowsOnlyDocumentationFiles(t *testing.T) {
	source := newDocsSource("https://docs.test/endge-docs/", "test")
	allowed := []string{"llms-agent.txt", "agents/index.md", "llms-full/reference/contracts/core/index.txt", "llms-manifest.json"}
	for _, relative := range allowed {
		if !source.allowed(source.uri(relative)) {
			t.Fatalf("%s must be allowed", relative)
		}
	}
	denied := []string{
		"https://docs.test/endge-docsX/llms.txt", "https://docs.test/llms.txt", source.uri("../llms.txt"), source.uri("a//b.md"),
		source.uri("a/./b.md"), source.uri("index.html"), source.uri("a.md?x=1"), source.uri("a%2F..%2Fb.md"), source.uri("contracts-manifest.json"),
	}
	for _, uri := range denied {
		if source.allowed(uri) {
			t.Fatalf("%s must be denied", uri)
		}
	}
}

func TestMCPWithoutDocsURLHasNoResources(t *testing.T) {
	useCases, _ := newFakeUseCases()
	server := newServer(Config{Version: "0.17.0"}, useCases, Contracts{}, nil, permissions{}, newDocsSource("", "0.17.0"))
	client := mcp.NewClient(&mcp.Implementation{Name: "t", Version: "1"}, nil)
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	if _, err := server.Connect(context.Background(), serverTransport, nil); err != nil {
		t.Fatal(err)
	}
	session, err := client.Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	if session.InitializeResult().Capabilities.Resources != nil {
		t.Fatalf("resources capability must be absent without AGENT_DOCS_URL")
	}
}
