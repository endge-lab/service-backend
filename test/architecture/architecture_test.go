package architecture_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current file")
	}

	return filepath.Join(filepath.Dir(currentFile), "..", "..")
}

func listGoFiles(t *testing.T, root string, relativeDir string) []string {
	t.Helper()

	dir := filepath.Join(root, relativeDir)
	var files []string

	err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		files = append(files, path)
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", relativeDir, err)
	}

	return files
}

func parsedImports(t *testing.T, filePath string) []string {
	t.Helper()

	fileSet := token.NewFileSet()
	parsed, err := parser.ParseFile(fileSet, filePath, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse imports %s: %v", filePath, err)
	}

	imports := make([]string, 0, len(parsed.Imports))
	for _, imported := range parsed.Imports {
		imports = append(imports, strings.Trim(imported.Path.Value, `"`))
	}

	return imports
}

func packageName(t *testing.T, filePath string) string {
	t.Helper()

	fileSet := token.NewFileSet()
	parsed, err := parser.ParseFile(fileSet, filePath, nil, parser.PackageClauseOnly)
	if err != nil {
		t.Fatalf("parse package name %s: %v", filePath, err)
	}

	return parsed.Name.Name
}

func TestTemplateRequiredPathsExist(t *testing.T) {
	root := repoRoot(t)

	requiredPaths := []string{
		"docs/architecture.md",
		"docs/openapi.yaml",
		"internal/api/http",
		"internal/bootstrap/usecase.go",
		"internal/config",
		"internal/domain/entities",
		"internal/domain/errors",
		"internal/domain/valueobjects",
		"internal/middleware",
		"internal/platform",
		"internal/ports",
		"internal/repo/postgres",
		"internal/services",
		"internal/usecase",
		"test/contract",
		"test/e2e",
		"test/integration",
	}

	for _, relativePath := range requiredPaths {
		if _, err := os.Stat(filepath.Join(root, relativePath)); err != nil {
			t.Fatalf("required architecture path is missing: %s", relativePath)
		}
	}
}

func TestTemplateLayerPackageNames(t *testing.T) {
	root := repoRoot(t)

	expectedPackages := map[string]string{
		"internal/api/http":            "http",
		"internal/bootstrap":           "bootstrap",
		"internal/domain/entities":     "entities",
		"internal/domain/errors":       "errors",
		"internal/domain/valueobjects": "valueobjects",
		"internal/middleware":          "middleware",
		"internal/platform":            "platform",
		"internal/ports":               "ports",
		"internal/repo/postgres":       "postgres",
		"internal/services":            "services",
		"internal/usecase":             "usecase",
	}

	for relativeDir, expectedPackage := range expectedPackages {
		files := listGoFiles(t, root, relativeDir)
		if len(files) == 0 {
			t.Fatalf("expected go files in %s", relativeDir)
		}

		for _, filePath := range files {
			if actualPackage := packageName(t, filePath); actualPackage != expectedPackage {
				t.Fatalf("unexpected package name in %s: got %s want %s", filePath, actualPackage, expectedPackage)
			}
		}
	}
}

func TestTemplateDependencyBoundaries(t *testing.T) {
	root := repoRoot(t)

	type dependencyRule struct {
		relativeDir      string
		forbiddenImports []string
	}

	rules := []dependencyRule{
		{
			relativeDir: "internal/domain",
			forbiddenImports: []string{
				"/internal/api/http",
				"/internal/middleware",
				"/internal/repo/postgres",
				"database/sql",
				"github.com/gofiber/",
				"github.com/jackc/pgx",
				"go.uber.org/fx",
			},
		},
		{
			relativeDir: "internal/services",
			forbiddenImports: []string{
				"/internal/api/http",
				"/internal/repo/postgres",
				"database/sql",
				"github.com/gofiber/",
				"github.com/jackc/pgx",
				"go.uber.org/fx",
			},
		},
		{
			relativeDir: "internal/usecase",
			forbiddenImports: []string{
				"/internal/api/http",
				"/internal/repo/postgres",
				"database/sql",
				"github.com/gofiber/",
				"github.com/jackc/pgx",
				"go.uber.org/fx",
			},
		},
		{
			relativeDir: "internal/api/http",
			forbiddenImports: []string{
				"/internal/repo/postgres",
			},
		},
		{
			relativeDir: "internal/repo/postgres",
			forbiddenImports: []string{
				"/internal/api/http",
			},
		},
	}

	for _, rule := range rules {
		for _, filePath := range listGoFiles(t, root, rule.relativeDir) {
			for _, importedPath := range parsedImports(t, filePath) {
				for _, forbidden := range rule.forbiddenImports {
					if strings.Contains(importedPath, forbidden) {
						t.Fatalf("forbidden dependency %s found in %s", importedPath, filePath)
					}
				}
			}
		}
	}
}

func TestBootstrapUseCaseModulesWireReferenceLayers(t *testing.T) {
	root := repoRoot(t)
	imports := parsedImports(t, filepath.Join(root, "internal/bootstrap/usecase.go"))

	requiredImports := []string{
		"/internal/api/http",
		"/internal/auth",
		"/internal/middleware",
		"/internal/ports",
		"/internal/repo/postgres",
		"/internal/services",
		"/internal/usecase",
	}

	for _, requiredImport := range requiredImports {
		found := false
		for _, importedPath := range imports {
			if strings.Contains(importedPath, requiredImport) {
				found = true
				break
			}
		}

		if !found {
			t.Fatalf("bootstrap/usecase.go must import %s", requiredImport)
		}
	}
}
