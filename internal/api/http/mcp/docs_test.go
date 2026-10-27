package mcp

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf8"

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
		"/endge-docs/llms-full/long.txt":                     longDoc(),
		"/endge-docs/llms-full.txt":                          longDoc(),
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

// longDoc — файл ≈200 тыс. символов из разделов с заголовками и блоками кода.
func longDoc() string {
	var text strings.Builder
	text.WriteString("# Длинный раздел\n\n")
	for section := 1; section <= 60; section++ {
		fmt.Fprintf(&text, "## Раздел %d\n\n", section)
		text.WriteString("```bash\n# комментарий в коде, а не заголовок\necho ok\n```\n\n")
		for paragraph := 0; paragraph < 50; paragraph++ {
			text.WriteString("Строка текста раздела документации для проверки частей ресурса.\n")
		}
		text.WriteString("\n")
	}
	return text.String()
}

func TestMCPResourcesServeLongFilesInParts(t *testing.T) {
	h := newHarness(t)
	session := h.connect(t, "read")
	base := h.docs.URL + "/endge-docs/llms-full/long.txt"
	whole := longDoc()
	first, err := readResource(t, session, base)
	if err != nil {
		t.Fatal(err)
	}
	header := first.Contents[0].Text[:strings.Index(first.Contents[0].Text, "\n")]
	var total int
	if _, err := fmt.Sscanf(header, "[Часть 1 из %d:", &total); err != nil || total < 3 || !strings.Contains(header, "Следующая: "+base+"?part=2]") {
		t.Fatalf("first part must name the count and the next part: %q %v", header, err)
	}
	var joined strings.Builder
	for part := 1; part <= total; part++ {
		uri := fmt.Sprintf("%s?part=%d", base, part)
		read, err := readResource(t, session, uri)
		if err != nil {
			t.Fatalf("read %s: %v", uri, err)
		}
		text := read.Contents[0].Text
		if read.Contents[0].URI != uri || read.Contents[0].MIMEType != "text/plain" {
			t.Fatalf("part %d contents: %+v", part, read.Contents[0])
		}
		if chars := utf8.RuneCountInString(text); chars > docsPartMaxChars {
			t.Fatalf("part %d has %d characters, limit %d", part, chars, docsPartMaxChars)
		}
		lines := strings.Split(text, "\n")
		body := strings.Join(lines[2:len(lines)-3], "\n")
		if part > 1 && !strings.HasPrefix(body, "## Раздел ") {
			t.Fatalf("part %d must start at a heading: %q", part, body[:80])
		}
		if part < total && !strings.Contains(text, fmt.Sprintf("Продолжение: %s?part=%d]", base, part+1)) {
			t.Fatalf("part %d must end with the next part link", part)
		}
		if part == total && !strings.Contains(lines[0], "Это последняя часть") {
			t.Fatalf("last part must say so: %q", lines[0])
		}
		joined.WriteString(body)
		joined.WriteString("\n")
	}
	// Части без заголовков и концовок дают тот же текст (с точностью до пустых строк на стыках).
	if strings.Join(strings.Fields(joined.String()), " ") != strings.Join(strings.Fields(whole), " ") {
		t.Fatalf("parts must join back into the file")
	}
	if _, err := readResource(t, session, fmt.Sprintf("%s?part=%d", base, total+1)); err == nil || !strings.Contains(err.Error(), fmt.Sprintf("?part=1…%d", total)) {
		t.Fatalf("part out of range must name the valid range: %v", err)
	}
	// Короткий файл отдаётся целиком, без заголовка части, и с ?part=1 тоже.
	short := h.docs.URL + "/endge-docs/agents/index.md"
	for _, uri := range []string{short, short + "?part=1"} {
		if read, err := readResource(t, session, uri); err != nil || read.Contents[0].Text != "# Endge для ИИ-агентов" {
			t.Fatalf("short file %s: %+v %v", uri, read, err)
		}
	}
	for name, uri := range map[string]string{
		"unpublished file":     h.docs.URL + "/endge-docs/llms-full.txt",
		"unpublished part":     h.docs.URL + "/endge-docs/llms-full.txt?part=2",
		"zero part":            base + "?part=0",
		"leading zero":         base + "?part=02",
		"other parameter":      base + "?x=1",
		"extra parameter":      base + "?part=1&x=1",
		"part outside prefix":  "https://example.com/llms.txt?part=1",
		"traversal with parts": h.docs.URL + "/endge-docs/agents/../../secret.md?part=1",
	} {
		if _, err := readResource(t, session, uri); err == nil {
			t.Fatalf("%s must fail: %s", name, uri)
		}
	}
}

func TestSplitDocPartsCutsAtHeadingsOutsideCode(t *testing.T) {
	text := "# A\n" + strings.Repeat("текст\n", 20) + "```\n" + strings.Repeat("# не заголовок\n", 10) + "```\n" + "## B\n" + strings.Repeat("текст\n", 10)
	parts := splitDocParts(text, 200)
	if strings.Join(parts, "") != text {
		t.Fatalf("parts must keep every character")
	}
	for index, part := range parts {
		if utf8.RuneCountInString(part) > 200 {
			t.Fatalf("part %d is longer than the limit", index)
		}
		if index > 0 && strings.HasPrefix(part, "# не заголовок") {
			t.Fatalf("a part must not start inside a code block: %q", part)
		}
	}
	if len(parts) != 3 || !strings.HasPrefix(parts[1], "```") || !strings.HasPrefix(parts[2], "## B") {
		t.Fatalf("the cuts must fall before the code block and before the heading B: %q", parts)
	}
	// Без заголовков режется после пустой строки, а строка длиннее лимита — по символам.
	parts = splitDocParts(strings.Repeat("абзац\n", 20)+"\n"+strings.Repeat("абзац\n", 20), 150)
	if len(parts) < 2 || !strings.HasPrefix(parts[1], "абзац") || !strings.HasSuffix(parts[0], "\n") {
		t.Fatalf("paragraph cut: %q", parts)
	}
	long := strings.Repeat("ж", 250)
	if parts := splitDocParts(long, 100); len(parts) != 3 || strings.Join(parts, "") != long {
		t.Fatalf("long line must be cut by characters: %d", len(parts))
	}
	if parts := splitDocParts("короткий", 100); len(parts) != 1 {
		t.Fatalf("short text is one part")
	}
}
