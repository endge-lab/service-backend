package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const repositoryRoot = "../../.."

// TestGuidanceCoversEveryErrorCode проверяет, что словарь описывает каждый код из исходников и не содержит лишних.
func TestGuidanceCoversEveryErrorCode(t *testing.T) {
	scan, err := scanRepository(repositoryRoot)
	if err != nil {
		t.Fatalf("scan sources: %v", err)
	}
	if len(scan.Occurrences) == 0 {
		t.Fatal("scanner found no error codes")
	}
	guidance, err := loadGuidance(filepath.Join(repositoryRoot, "docs", "error-guidance.yaml"))
	if err != nil {
		t.Fatalf("load guidance: %v", err)
	}
	if _, problems := buildCatalog(scan, guidance); len(problems) > 0 {
		t.Fatalf("error-guidance.yaml is out of sync:\n%s", strings.Join(problems, "\n"))
	}
}

// TestCheckedInCatalogIsCurrent проверяет, что docs/error-codes.yaml перегенерирован после изменения кодов.
func TestCheckedInCatalogIsCurrent(t *testing.T) {
	scan, err := scanRepository(repositoryRoot)
	if err != nil {
		t.Fatalf("scan sources: %v", err)
	}
	guidance, err := loadGuidance(filepath.Join(repositoryRoot, "docs", "error-guidance.yaml"))
	if err != nil {
		t.Fatalf("load guidance: %v", err)
	}
	catalog, problems := buildCatalog(scan, guidance)
	if len(problems) > 0 {
		t.Skip("guidance is out of sync; see TestGuidanceCoversEveryErrorCode")
	}
	rendered, err := renderCatalog(catalog)
	if err != nil {
		t.Fatalf("render catalog: %v", err)
	}
	current, err := os.ReadFile(filepath.Join(repositoryRoot, "docs", "error-codes.yaml"))
	if err != nil {
		t.Fatalf("read checked-in catalog: %v", err)
	}
	if !bytes.Equal(rendered, current) {
		t.Fatal("docs/error-codes.yaml is stale; run make docs")
	}
}

// TestBuildCatalogReportsMissingAndExtraCodes проверяет оба направления сверки словаря.
func TestBuildCatalogReportsMissingAndExtraCodes(t *testing.T) {
	scan := scanResult{Occurrences: []occurrence{{Code: "known", Status: 400, Message: "m", File: "a.go"}, {Code: "missing", Status: 404, File: "b.go"}}}
	guidance := guidanceFile{Codes: map[string]guidanceEntry{
		"known": {Meaning: "m", Action: "a"},
		"stale": {Meaning: "m", Action: "a"},
	}}
	_, problems := buildCatalog(scan, guidance)
	joined := strings.Join(problems, "\n")
	if !strings.Contains(joined, `"missing"`) || !strings.Contains(joined, `"stale"`) {
		t.Fatalf("expected missing and stale codes to be reported, got:\n%s", joined)
	}
}

// TestScannerRecognisesEveryConstructorForm проверяет разбор конструкторов, fiber.Map и toolError.
func TestScannerRecognisesEveryConstructorForm(t *testing.T) {
	directory := t.TempDir()
	source := `package sample

import (
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/gofiber/fiber/v2"
)

type toolError struct{ Error, Message string }

func handler(c *fiber.Ctx, code string) error {
	_ = domainerrors.Conflict("kind_code", "kind message")
	_ = domainerrors.New("new_code", "new message", 428)
	_ = domainerrors.Wrap(nil, "wrap_code", "wrap message", 503)
	_ = toolError{Error: "tool_code", Message: "tool message"}
	_ = domainerrors.InvalidInput(code, "dynamic")
	return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"code": "map_code", "message": "map message"})
}
`
	for _, dir := range []string{"cmd", "internal/sample"} {
		if err := os.MkdirAll(filepath.Join(directory, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(directory, "internal", "sample", "sample.go"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	scan, err := scanRepository(directory)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	want := map[string]int{"kind_code": 409, "new_code": 428, "wrap_code": 503, "tool_code": 0, "map_code": 403}
	facts := aggregate(scan.Occurrences)
	for code, status := range want {
		fact, ok := facts[code]
		if !ok {
			t.Fatalf("code %q not found", code)
		}
		if status == 0 && len(fact.Statuses) != 0 || status != 0 && (len(fact.Statuses) != 1 || fact.Statuses[0] != status) {
			t.Fatalf("code %q statuses = %v, want %d", code, fact.Statuses, status)
		}
	}
	if len(scan.Unresolved) != 1 || scan.Unresolved[0].Function != "handler" {
		t.Fatalf("unresolved = %+v, want one dynamic call in handler", scan.Unresolved)
	}
}
