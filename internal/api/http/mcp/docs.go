package mcp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	// docsFetchTimeout ограничивает один запрос к сайту документации.
	docsFetchTimeout = 10 * time.Second
	// docsMaxBytes — наибольший размер ресурса; больший файл обрезается с указанием продолжения.
	docsMaxBytes = 2 << 20
	// docsCacheTTL — сколько backend отдаёт файл из памяти, не обращаясь к сайту.
	docsCacheTTL = 10 * time.Minute
	// docsCacheMaxBytes ограничивает память кеша: шаблоны позволяют запросить много страниц.
	docsCacheMaxBytes = 32 << 20
	// docsPartMaxChars — наибольший ответ resources/read в символах вместе с заголовком части.
	// MCP-клиенты ограничивают вывод инструмента (Claude Code — около 25 тыс. токенов),
	// поэтому длинный файл отдаётся частями по ?part=N.
	docsPartMaxChars = 60_000
	// docsPartParam — параметр адреса ресурса с номером части.
	docsPartParam = "part"
)

// agentDoc — файл llms-документации, публикуемый как ресурс MCP. Порядок важен:
// агент читает ресурсы сверху вниз, ядро llms-agent.txt — первым.
type agentDoc struct {
	path, name, title, description string
}

var agentDocs = []agentDoc{
	{"llms-agent.txt", "llms-agent", "Ядро для агента — читать первым", "Всё для типовой работы одним файлом (≈45 тыс. токенов, отдаётся частями по ≈60 тыс. символов): руководство агента, токены, MCP, правила версий и лимитов, сценарии, коды ошибок, авторинг Source, все HTTP-операции. Схемы тел — в llms-full/api/{group}.txt, Source каждого типа — в llms-full/reference/{type}.txt."},
	{"llms.txt", "llms", "Оглавление документации", "Оглавление всех страниц с описаниями (llmstxt.org); страницы читаются шаблоном {path}.md."},
	{"agents/index.md", "agents-guide", "Руководство агента", "Порядок работы агента: вход по токену, workspace, чтение и изменение документов, коммиты, лимиты."},
	{"agents/mcp.md", "agents-mcp", "MCP-сервер Endge", "Инструменты, ресурсы и права MCP-сервера backend."},
	{"agents/errors.md", "agents-errors", "Коды ошибок", "Все коды ошибок backend: HTTP-статус, смысл и что делать агенту."},
	{"agents/scenarios.md", "agents-scenarios", "Сквозные сценарии", "Полные последовательности запросов с ответами для типовых задач."},
	{"llms-manifest.json", "llms-manifest", "Состав llms-выгрузки", "Список файлов llms с назначением, размером и оценкой токенов: имена групп API и частей справочника для шаблонов."},
}

// agentDocTemplate — шаблон ресурса для частей документации.
type agentDocTemplate struct {
	uriTemplate, name, title, description, mimeType string
}

var agentDocTemplates = []agentDocTemplate{
	{"llms-full/api/{group}.txt", "llms-api-group", "Группа HTTP API", "Операции группы API со схемами тел и ответов. group — имя группы из списка операций llms-agent.txt, например facets, queries, commits, revisions.", "text/plain"},
	{"llms-full/reference/{+type}.txt", "llms-reference", "Справочник Source по типу", "Авторский Source одного типа документа, например query, component-sfc, type, data-view; контракты пакетов — contracts/<пакет>.", "text/plain"},
	{"llms-full/{section}.txt", "llms-section", "Раздел документации целиком", "Раздел сайта одним файлом, например agents, guides, domain, start.", "text/plain"},
	{"{+path}.md", "docs-page", "Страница документации в Markdown", "Любая страница сайта: путь из llms.txt без расширения, например agents/tokens или reference/query.", "text/markdown"},
	{"{+resource}{?part}", "docs-part", "Часть длинного ресурса", "Ответ resources/read не длиннее ≈60 тыс. символов: длинный файл делится на части по заголовкам. resource — адрес ресурса из списка или шаблонов выше без префикса, part — номер части с 1. Первая строка части: «[Часть N из M: <адрес>. Следующая: <адрес>?part=N+1]»; читайте части по порядку.", ""},
}

// docsPartTemplate — шаблон частей; он совпадает с любым адресом под префиксом, поэтому
// читать разрешено только файлы из agentDocs и других шаблонов (published).
const docsPartTemplate = "{+resource}{?part}"

// publishedPatterns — относительные пути шаблонов agentDocTemplates, кроме шаблона частей.
var publishedPatterns = func() []*regexp.Regexp {
	variable := regexp.MustCompile(`\{(\+?)[a-z]+\}`)
	patterns := make([]*regexp.Regexp, 0, len(agentDocTemplates))
	for _, template := range agentDocTemplates {
		if template.uriTemplate == docsPartTemplate {
			continue
		}
		var pattern strings.Builder
		last := 0
		for _, match := range variable.FindAllStringSubmatchIndex(template.uriTemplate, -1) {
			pattern.WriteString(regexp.QuoteMeta(template.uriTemplate[last:match[0]]))
			if match[3] > match[2] {
				pattern.WriteString(`[A-Za-z0-9._/-]+`)
			} else {
				pattern.WriteString(`[A-Za-z0-9._-]+`)
			}
			last = match[1]
		}
		pattern.WriteString(regexp.QuoteMeta(template.uriTemplate[last:]))
		patterns = append(patterns, regexp.MustCompile("^"+pattern.String()+"$"))
	}
	return patterns
}()

// docsPathPattern — допустимый относительный путь ресурса: латиница, цифры, «.», «_», «-» и «/».
var docsPathPattern = regexp.MustCompile(`^[A-Za-z0-9._/-]+$`)

// docsSource читает файлы llms-документации с AGENT_DOCS_URL и кеширует их в памяти.
// Запросы уходят только под префикс AGENT_DOCS_URL: пути проверяются до запроса,
// редиректы за пределы префикса отклоняются.
type docsSource struct {
	base     string
	prefix   string
	client   *http.Client
	ttl      time.Duration
	maxBytes int
	maxCache int
	// partChars — наибольший размер части ответа в символах.
	partChars int
	agent     string
	now       func() time.Time

	mu          sync.Mutex
	cache       map[string]docsEntry
	cachedBytes int
}

type docsEntry struct {
	text     string
	mimeType string
	expires  time.Time
}

// newDocsSource возвращает nil без AGENT_DOCS_URL: ресурсы документации не публикуются.
func newDocsSource(base, version string) *docsSource {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		return nil
	}
	source := &docsSource{
		base: base, prefix: base + "/", ttl: docsCacheTTL, maxBytes: docsMaxBytes, maxCache: docsCacheMaxBytes, partChars: docsPartMaxChars,
		agent: "endge-backend-mcp/" + version, now: time.Now, cache: map[string]docsEntry{},
	}
	source.client = &http.Client{Timeout: docsFetchTimeout, CheckRedirect: source.checkRedirect}
	return source
}

func (d *docsSource) checkRedirect(request *http.Request, via []*http.Request) error {
	if len(via) >= 3 {
		return errors.New("слишком много перенаправлений")
	}
	if !strings.HasPrefix(request.URL.String(), d.prefix) {
		return fmt.Errorf("перенаправление за пределы AGENT_DOCS_URL запрещено: %s", request.URL.Redacted())
	}
	return nil
}

// uri — абсолютный адрес файла документации.
func (d *docsSource) uri(relative string) string { return d.prefix + relative }

// links — ссылки на основные файлы для endge_status и instructions.
func (d *docsSource) links() []map[string]string {
	if d == nil {
		return nil
	}
	links := make([]map[string]string, 0, len(agentDocs))
	for _, doc := range agentDocs {
		links = append(links, map[string]string{"title": doc.title, "url": d.uri(doc.path)})
	}
	return links
}

// register публикует ресурсы и шаблоны ресурсов документации.
func (d *docsSource) register(server *mcp.Server) {
	if d == nil {
		return
	}
	for _, doc := range agentDocs {
		server.AddResource(&mcp.Resource{URI: d.uri(doc.path), Name: doc.name, Title: doc.title, Description: doc.description, MIMEType: docsMIMEType(doc.path)}, d.read)
	}
	for _, template := range agentDocTemplates {
		server.AddResourceTemplate(&mcp.ResourceTemplate{URITemplate: d.uri(template.uriTemplate), Name: template.name, Title: template.title, Description: template.description, MIMEType: template.mimeType}, d.read)
	}
	server.AddReceivingMiddleware(d.orderResources)
}

// orderResources возвращает resources/list в порядке agentDocs: SDK сортирует ресурсы по URI,
// а агенту нужно видеть ядро llms-agent.txt первым.
func (d *docsSource) orderResources(next mcp.MethodHandler) mcp.MethodHandler {
	rank := make(map[string]int, len(agentDocs))
	for position, doc := range agentDocs {
		rank[d.uri(doc.path)] = position
	}
	return func(ctx context.Context, method string, request mcp.Request) (mcp.Result, error) {
		result, err := next(ctx, method, request)
		if list, ok := result.(*mcp.ListResourcesResult); ok && err == nil {
			slices.SortStableFunc(list.Resources, func(a, b *mcp.Resource) int { return rank[a.URI] - rank[b.URI] })
		}
		return result, err
	}
}

// read — обработчик resources/read для ресурсов и шаблонов. Файл длиннее partChars
// отдаётся частями: без ?part — первая часть, каждая часть с заголовком и адресом следующей.
func (d *docsSource) read(ctx context.Context, request *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	uri := request.Params.URI
	base, part, ok := splitPartURI(uri)
	if !ok || !d.allowed(base) || !d.published(base) {
		return nil, mcp.ResourceNotFoundError(uri)
	}
	text, mimeType, err := d.fetch(ctx, base)
	if err != nil {
		return nil, err
	}
	parts := splitDocParts(text, d.partChars-docsPartFrameChars(base))
	if part > len(parts) {
		return nil, fmt.Errorf("в ресурсе %s %d %s: запросите ?%s=1…%d", base, len(parts), pluralParts(len(parts)), docsPartParam, len(parts))
	}
	if len(parts) > 1 {
		text = framePart(base, parts[part-1], part, len(parts))
	}
	return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{URI: uri, MIMEType: mimeType, Text: text}}}, nil
}

// published — файл опубликован ресурсом или одним из шаблонов (кроме шаблона частей).
func (d *docsSource) published(uri string) bool {
	relative, ok := strings.CutPrefix(uri, d.prefix)
	if !ok {
		return false
	}
	for _, doc := range agentDocs {
		if doc.path == relative {
			return true
		}
	}
	for _, pattern := range publishedPatterns {
		if pattern.MatchString(relative) {
			return true
		}
	}
	return false
}

// splitPartURI отделяет ?part=N от адреса. Без параметра — первая часть; другие параметры
// и некорректный номер не принимаются.
func splitPartURI(uri string) (string, int, bool) {
	base, query, found := strings.Cut(uri, "?")
	if !found {
		return uri, 1, true
	}
	value, ok := strings.CutPrefix(query, docsPartParam+"=")
	if !ok || value == "" || len(value) > 4 || strings.TrimLeft(value, "0123456789") != "" || value[0] == '0' {
		return "", 0, false
	}
	part, err := strconv.Atoi(value)
	if err != nil || part < 1 {
		return "", 0, false
	}
	return base, part, true
}

// partURI — адрес части number файла base.
func partURI(base string, number int) string {
	return fmt.Sprintf("%s?%s=%d", base, docsPartParam, number)
}

// docsPartFrameChars — запас символов под заголовок и концовку части.
func docsPartFrameChars(base string) int {
	return 3*utf8.RuneCountInString(base) + 200
}

// framePart добавляет к части заголовок и концовку с адресом следующей части.
func framePart(base, body string, number, total int) string {
	header := fmt.Sprintf("[Часть %d из %d: %s. Следующая: %s]\n\n", number, total, base, partURI(base, number+1))
	footer := fmt.Sprintf("\n\n[Конец части %d из %d. Продолжение: %s]\n", number, total, partURI(base, number+1))
	if number == total {
		header = fmt.Sprintf("[Часть %d из %d: %s. Это последняя часть]\n\n", number, total, base)
		footer = fmt.Sprintf("\n\n[Конец файла %s]\n", base)
	}
	return header + strings.TrimRight(body, "\n") + footer
}

func pluralParts(count int) string {
	switch {
	case count%10 == 1 && count%100 != 11:
		return "часть"
	case count%10 >= 2 && count%10 <= 4 && (count%100 < 12 || count%100 > 14):
		return "части"
	}
	return "частей"
}

// docLine — строка файла и её свойства для выбора места разреза.
type docLine struct {
	text    string
	chars   int
	heading bool // заголовок Markdown вне блока кода
	blank   bool // пустая строка вне блока кода
	fence   bool // строка открывает блок кода
}

// splitDocParts делит текст на части не длиннее limit символов. Разрез ставится перед
// заголовком Markdown вне блока кода, иначе после пустой строки или перед блоком кода,
// иначе по границе строки;
// строка длиннее limit режется по символам. Короткий текст возвращается одной частью.
func splitDocParts(text string, limit int) []string {
	if limit < 1 || utf8.RuneCountInString(text) <= limit {
		return []string{text}
	}
	lines := docLines(text, limit)
	parts := []string{}
	for start := 0; start < len(lines); {
		chars, end := 0, start
		for end < len(lines) && (end == start || chars+lines[end].chars <= limit) {
			chars += lines[end].chars
			end++
		}
		if end < len(lines) {
			end = cutBefore(lines, start, end, limit)
		}
		var part strings.Builder
		for _, line := range lines[start:end] {
			part.WriteString(line.text)
		}
		parts = append(parts, part.String())
		start = end
	}
	return parts
}

// cutBefore выбирает индекс строки, перед которой режется часть lines[start:end].
// Разрез не раньше середины части: заголовок, затем пустая строка или начало блока кода,
// иначе конец части.
func cutBefore(lines []docLine, start, end, limit int) int {
	heading, blank := -1, -1
	chars := 0
	for index := start; index < end; index++ {
		if index > start && chars >= limit/2 {
			// Не отрывать заголовок от следующего за ним заголовка или текста.
			if lines[index].heading && !lines[index-1].heading {
				heading = index
			}
			if (lines[index-1].blank && !lines[index].blank) || (lines[index].fence && !lines[index-1].heading) {
				blank = index
			}
		}
		chars += lines[index].chars
	}
	switch {
	case heading > start:
		return heading
	case blank > start:
		return blank
	}
	return end
}

// docLines разбивает текст на строки с признаками заголовка и пустой строки вне блоков кода.
func docLines(text string, limit int) []docLine {
	lines := []docLine{}
	fence := ""
	for _, raw := range strings.SplitAfter(text, "\n") {
		if raw == "" {
			continue
		}
		trimmed := strings.TrimSpace(raw)
		inCode, opens := fence != "", false
		if marker := fenceMarker(trimmed); marker != "" && (fence == "" || strings.HasPrefix(trimmed, fence)) {
			opens = fence == ""
			if opens {
				fence = marker
			} else {
				fence = ""
			}
			inCode = true
		}
		line := docLine{
			heading: !inCode && isHeading(raw),
			blank:   !inCode && trimmed == "",
			fence:   opens,
		}
		for _, piece := range splitLongLine(raw, limit) {
			line.text, line.chars = piece, utf8.RuneCountInString(piece)
			lines = append(lines, line)
			line.heading, line.fence = false, false
		}
	}
	return lines
}

// fenceMarker — открывающая или закрывающая последовательность блока кода (``` или ~~~).
func fenceMarker(line string) string {
	for _, marker := range []string{"```", "~~~"} {
		if strings.HasPrefix(line, marker) {
			return marker
		}
	}
	return ""
}

// isHeading — заголовок ATX Markdown: от одного до шести «#» и пробел.
func isHeading(line string) bool {
	level := 0
	for level < len(line) && line[level] == '#' {
		level++
	}
	return level >= 1 && level <= 6 && level < len(line) && (line[level] == ' ' || line[level] == '\t')
}

// splitLongLine режет строку длиннее limit на куски по символам.
func splitLongLine(line string, limit int) []string {
	if utf8.RuneCountInString(line) <= limit {
		return []string{line}
	}
	pieces := []string{}
	runes := []rune(line)
	for len(runes) > limit {
		pieces = append(pieces, string(runes[:limit]))
		runes = runes[limit:]
	}
	return append(pieces, string(runes))
}

// allowed пропускает только файлы .md, .txt и манифест под префиксом AGENT_DOCS_URL без «..».
func (d *docsSource) allowed(uri string) bool {
	relative, ok := strings.CutPrefix(uri, d.prefix)
	if !ok || !docsPathPattern.MatchString(relative) {
		return false
	}
	for _, segment := range strings.Split(relative, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return false
		}
	}
	switch path.Ext(relative) {
	case ".md", ".txt":
		return true
	}
	return relative == "llms-manifest.json"
}

// fetch возвращает текст из кеша или с сайта документации.
func (d *docsSource) fetch(ctx context.Context, uri string) (string, string, error) {
	if entry, ok := d.lookup(uri); ok {
		return entry.text, entry.mimeType, nil
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return "", "", mcp.ResourceNotFoundError(uri)
	}
	request.Header.Set("Accept", "text/plain, text/markdown, application/json")
	request.Header.Set("User-Agent", d.agent)
	response, err := d.client.Do(request)
	if err != nil {
		return "", "", fmt.Errorf("документация недоступна: %s (%v); откройте ссылку напрямую или повторите позже", uri, unwrapURLError(err))
	}
	defer response.Body.Close()
	contentType, _, _ := mime.ParseMediaType(response.Header.Get("Content-Type"))
	// Сайт может отдать страницу 404 или SPA-заглушку HTML со статусом 200: это не файл документации.
	if response.StatusCode == http.StatusNotFound || response.StatusCode == http.StatusGone || contentType == "text/html" {
		return "", "", mcp.ResourceNotFoundError(uri)
	}
	if response.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("документация недоступна: %s ответил HTTP %d; откройте ссылку напрямую или повторите позже", uri, response.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, int64(d.maxBytes)+1))
	if err != nil {
		return "", "", fmt.Errorf("документация недоступна: не удалось дочитать %s (%v); повторите позже", uri, err)
	}
	text := d.truncate(uri, raw)
	mimeType := docsMIMEType(uri)
	d.store(uri, docsEntry{text: text, mimeType: mimeType, expires: d.now().Add(d.ttl)})
	return text, mimeType, nil
}

// truncate обрезает файл больше лимита по границе строки и говорит, где продолжение.
func (d *docsSource) truncate(uri string, raw []byte) string {
	if len(raw) <= d.maxBytes {
		return string(raw)
	}
	cut := raw[:d.maxBytes]
	if line := bytes.LastIndexByte(cut, '\n'); line > 0 {
		cut = cut[:line+1]
	}
	// Не оставлять оборванный UTF-8 символ в конце.
	for len(cut) > 0 && !utf8.Valid(cut) {
		cut = cut[:len(cut)-1]
	}
	return string(cut) + fmt.Sprintf("\n\n[Ресурс обрезан: файл больше %d МиБ. Полный файл: %s. Читайте его по частям: %s, %s, %s или страницы %s.]\n",
		d.maxBytes>>20, uri, d.uri("llms-full/api/{group}.txt"), d.uri("llms-full/reference/{type}.txt"), d.uri("llms-full/{section}.txt"), d.uri("{path}.md"))
}

func (d *docsSource) lookup(uri string) (docsEntry, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	entry, ok := d.cache[uri]
	if !ok || !d.now().Before(entry.expires) {
		return docsEntry{}, false
	}
	return entry, true
}

// store кладёт файл в кеш; при превышении памяти сначала удаляет устаревшие, затем самые старые записи.
func (d *docsSource) store(uri string, entry docsEntry) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if previous, ok := d.cache[uri]; ok {
		d.cachedBytes -= len(previous.text)
	}
	d.cache[uri] = entry
	d.cachedBytes += len(entry.text)
	now := d.now()
	for key, value := range d.cache {
		if d.cachedBytes <= d.maxCache {
			return
		}
		if !now.Before(value.expires) {
			d.cachedBytes -= len(value.text)
			delete(d.cache, key)
		}
	}
	for d.cachedBytes > d.maxCache && len(d.cache) > 1 {
		oldest := ""
		for key, value := range d.cache {
			if key != uri && (oldest == "" || value.expires.Before(d.cache[oldest].expires)) {
				oldest = key
			}
		}
		if oldest == "" {
			return
		}
		d.cachedBytes -= len(d.cache[oldest].text)
		delete(d.cache, oldest)
	}
}

func docsMIMEType(name string) string {
	switch path.Ext(name) {
	case ".md":
		return "text/markdown"
	case ".json":
		return "application/json"
	}
	return "text/plain"
}

// unwrapURLError убирает из сообщения повтор метода и адреса, оставляя причину.
func unwrapURLError(err error) error {
	var urlErr interface{ Unwrap() error }
	if errors.As(err, &urlErr) {
		if inner := urlErr.Unwrap(); inner != nil {
			return inner
		}
	}
	return err
}
