package main

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

// guidanceFile — вручную поддерживаемый словарь смыслов docs/error-guidance.yaml.
type guidanceFile struct {
	// Templates — общие формулировки для похожих кодов; запись ссылается на них через use.
	Templates map[string]guidanceText `yaml:"templates"`
	// Dynamic — места, где код приходит во время выполнения (например, из upstream-сервиса).
	Dynamic []dynamicSource `yaml:"dynamic"`
	// Codes — смысл и действие для каждого кода из исходников.
	Codes map[string]guidanceEntry `yaml:"codes"`
}

type guidanceText struct {
	Meaning string `yaml:"meaning"`
	Action  string `yaml:"action"`
}

type guidanceEntry struct {
	Use     string `yaml:"use"`
	Meaning string `yaml:"meaning"`
	Action  string `yaml:"action"`
	// Status обязателен только для кодов, статус которых задаётся во время выполнения.
	Status int `yaml:"status"`
}

type dynamicSource struct {
	File     string `yaml:"file"`
	Function string `yaml:"function"`
	Note     string `yaml:"note"`
}

// catalogEntry — запись сгенерированного docs/error-codes.yaml.
type catalogEntry struct {
	Code           string   `yaml:"code"`
	Status         int      `yaml:"status"`
	Statuses       []int    `yaml:"statuses,omitempty,flow"`
	DefaultMessage string   `yaml:"defaultMessage"`
	OtherMessages  []string `yaml:"otherMessages,omitempty"`
	Meaning        string   `yaml:"meaning"`
	Action         string   `yaml:"action"`
	Sources        []string `yaml:"sources"`
}

type catalogDynamic struct {
	File     string `yaml:"file"`
	Function string `yaml:"function"`
	Note     string `yaml:"note"`
}

type catalogFile struct {
	Codes   []catalogEntry   `yaml:"codes"`
	Dynamic []catalogDynamic `yaml:"dynamic,omitempty"`
}

func loadGuidance(path string) (guidanceFile, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return guidanceFile{}, err
	}
	var guidance guidanceFile
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	decoder.KnownFields(true)
	if err := decoder.Decode(&guidance); err != nil {
		return guidanceFile{}, fmt.Errorf("parse %s: %w", path, err)
	}
	return guidance, nil
}

// buildCatalog сопоставляет коды из исходников со словарём и возвращает все расхождения сразу.
func buildCatalog(scan scanResult, guidance guidanceFile) (catalogFile, []string) {
	var problems []string
	facts := aggregate(scan.Occurrences)

	allowedDynamic := map[string]dynamicSource{}
	for _, source := range guidance.Dynamic {
		allowedDynamic[source.File+"#"+source.Function] = source
	}
	usedDynamic := map[string]bool{}
	for _, site := range scan.Unresolved {
		key := site.File + "#" + site.Function
		if _, ok := allowedDynamic[key]; !ok {
			problems = append(problems, fmt.Sprintf("%s:%d (%s): код ошибки не литеральный; используйте строковый литерал или опишите место в dynamic", site.File, site.Line, site.Function))
			continue
		}
		usedDynamic[key] = true
	}
	for key := range allowedDynamic {
		if !usedDynamic[key] {
			problems = append(problems, fmt.Sprintf("dynamic %s: в исходниках нет нелитерального кода в этом месте; удалите запись", key))
		}
	}

	codes := make([]string, 0, len(facts))
	for code := range facts {
		codes = append(codes, code)
	}
	sort.Strings(codes)

	var catalog catalogFile
	for _, code := range codes {
		fact := facts[code]
		entry, ok := guidance.Codes[code]
		if !ok {
			problems = append(problems, fmt.Sprintf("код %q (%s) отсутствует в error-guidance.yaml", code, strings.Join(fact.Sources, ", ")))
			continue
		}
		text, err := resolveText(entry, guidance.Templates)
		if err != nil {
			problems = append(problems, fmt.Sprintf("код %q: %v", code, err))
			continue
		}
		statuses := fact.Statuses
		if len(statuses) == 0 {
			if entry.Status == 0 {
				problems = append(problems, fmt.Sprintf("код %q: статус задаётся во время выполнения, укажите status в error-guidance.yaml", code))
				continue
			}
			statuses = []int{entry.Status}
		} else if entry.Status != 0 {
			problems = append(problems, fmt.Sprintf("код %q: статус известен из кода (%v), уберите status из error-guidance.yaml", code, fact.Statuses))
		}
		item := catalogEntry{
			Code:    code,
			Status:  statuses[0],
			Meaning: text.Meaning,
			Action:  text.Action,
			Sources: fact.Sources,
		}
		if len(statuses) > 1 {
			item.Statuses = statuses
		}
		if len(fact.Messages) > 0 {
			item.DefaultMessage = fact.Messages[0]
			item.OtherMessages = fact.Messages[1:]
		}
		catalog.Codes = append(catalog.Codes, item)
	}
	for code := range guidance.Codes {
		if _, ok := facts[code]; !ok {
			problems = append(problems, fmt.Sprintf("код %q есть в error-guidance.yaml, но не создаётся в исходниках; удалите запись", code))
		}
	}
	for _, source := range guidance.Dynamic {
		catalog.Dynamic = append(catalog.Dynamic, catalogDynamic(source))
	}
	sort.Strings(problems)
	return catalog, problems
}

func resolveText(entry guidanceEntry, templates map[string]guidanceText) (guidanceText, error) {
	text := guidanceText{Meaning: entry.Meaning, Action: entry.Action}
	if entry.Use != "" {
		template, ok := templates[entry.Use]
		if !ok {
			return guidanceText{}, fmt.Errorf("неизвестный шаблон %q", entry.Use)
		}
		if text.Meaning == "" {
			text.Meaning = template.Meaning
		}
		if text.Action == "" {
			text.Action = template.Action
		}
	}
	if strings.TrimSpace(text.Meaning) == "" || strings.TrimSpace(text.Action) == "" {
		return guidanceText{}, fmt.Errorf("нужны meaning и action")
	}
	return text, nil
}

const catalogHeader = `# Code generated by internal/tools/errorcatalog; DO NOT EDIT.
# Каталог кодов ошибок backend: собирается из исходников (код, HTTP-статус, сообщение, файлы)
# и словаря docs/error-guidance.yaml (meaning — что произошло, action — что делать агенту).
# Тот же code приходит в HTTP-ответе {code, message, details} и в ошибке MCP-инструмента {error, status}.
`

func renderCatalog(catalog catalogFile) ([]byte, error) {
	var buffer bytes.Buffer
	buffer.WriteString(catalogHeader)
	encoder := yaml.NewEncoder(&buffer)
	encoder.SetIndent(2)
	if err := encoder.Encode(catalog); err != nil {
		return nil, err
	}
	if err := encoder.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}
