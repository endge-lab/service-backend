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
		"docs/Архитектура.md",
		"docs/openapi3.yaml",
		"internal/api/http",
		"internal/api/http/middleware",
		"internal/bootstrap/usecase.go",
		"internal/config",
		"internal/domain/entities",
		"internal/domain/errors",
		"internal/domain/valueobjects",
		"internal/platform",
		"internal/repo",
		"internal/usecase/ports",
		"internal/repo/postgres",
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

func TestDomainMigrationOrder(t *testing.T) {
	root := repoRoot(t)
	expected := []string{
		"000001_init_service_users.sql",
		"000002_init_workspaces.sql",
		"000003_init_tenants.sql",
		"000004_init_projects.sql",
		"000005_init_environments.sql",
		"000006_init_folders.sql",
		"000007_init_versions.sql",
		"000008_init_types.sql",
		"000009_init_stores.sql",
		"000010_init_mocks.sql",
		"000011_init_vocabs.sql",
		"000012_init_queries.sql",
		"000013_init_data_views.sql",
		"000014_init_computations.sql",
		"000015_init_compositions.sql",
		"000016_init_components_legacy.sql",
		"000017_init_components.sql",
		"000018_init_filters.sql",
		"000019_init_converters.sql",
		"000020_init_auth_profiles.sql",
		"000021_init_navigations.sql",
		"000022_seed_swagger_demo_data.sql",
	}

	entries, err := os.ReadDir(filepath.Join(root, "migrations"))
	if err != nil {
		t.Fatalf("read migrations: %v", err)
	}

	actual := make([]string, 0, len(expected))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			actual = append(actual, entry.Name())
		}
	}

	if strings.Join(actual, "\n") != strings.Join(expected, "\n") {
		t.Fatalf("unexpected migration order:\n%s", strings.Join(actual, "\n"))
	}
}

func TestTemplateLayerPackageNames(t *testing.T) {
	root := repoRoot(t)

	expectedPackages := map[string]string{
		"internal/api/http":                     "http",
		"internal/api/http/health":              "health",
		"internal/api/http/middleware":          "middleware",
		"internal/api/http/openapi":             "openapi",
		"internal/api/http/respond":             "respond",
		"internal/api/http/session":             "session",
		"internal/api/http/v1":                  "http",
		"internal/api/http/v1/component_legacy": "component_legacy",
		"internal/api/http/v1/converter":        "converter",
		"internal/api/http/v1/data_view":        "data_view",
		"internal/api/http/v1/folder":           "folder",
		"internal/api/http/v1/project":          "project",
		"internal/api/http/v1/query":            "query",
		"internal/bootstrap":                    "bootstrap",
		"internal/domain/entities":              "entities",
		"internal/domain/errors":                "errors",
		"internal/platform":                     "platform",
		"internal/usecase/ports":                "ports",
		"internal/repo/postgres":                "postgres",
		"internal/repo/postgres/mappers":        "mappers",
		"internal/repo/postgres/sqlc":           "sqlc",
		"internal/usecase/components_legacy":    "components_legacy",
		"internal/usecase/converters":           "converters",
		"internal/usecase/data_views":           "data_views",
		"internal/usecase/folders":              "folders",
		"internal/usecase/projects":             "projects",
		"internal/usecase/queries":              "queries",
		"internal/usecase/session":              "session",
		"internal/usecase/shared":               "shared",
	}

	for relativeDir, expectedPackage := range expectedPackages {
		files := listGoFiles(t, root, relativeDir)
		if len(files) == 0 {
			t.Fatalf("expected go files in %s", relativeDir)
		}

		for _, filePath := range files {
			effectiveExpectedPackage := expectedPackage
			relativeFilePath, err := filepath.Rel(root, filePath)
			if err != nil {
				t.Fatalf("relative file path %s: %v", filePath, err)
			}
			matchedDir := relativeDir
			for expectedDir, expectedDirPackage := range expectedPackages {
				if strings.HasPrefix(relativeFilePath, expectedDir+string(filepath.Separator)) && len(expectedDir) > len(matchedDir) {
					effectiveExpectedPackage = expectedDirPackage
					matchedDir = expectedDir
				}
			}

			if actualPackage := packageName(t, filePath); actualPackage != effectiveExpectedPackage {
				t.Fatalf("unexpected package name in %s: got %s want %s", filePath, actualPackage, effectiveExpectedPackage)
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
				"/internal/repo/postgres",
				"database/sql",
				"github.com/gofiber/",
				"github.com/jackc/pgx",
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

func TestBootstrapRepositoryModulesWireReferenceLayers(t *testing.T) {
	root := repoRoot(t)
	imports := parsedImports(t, filepath.Join(root, "internal/bootstrap/repository.go"))

	requiredImports := []string{
		"/internal/repo/postgres",
		"/internal/usecase/ports",
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
			t.Fatalf("bootstrap/repository.go must import %s", requiredImport)
		}
	}
}

func TestBootstrapUseCaseModulesWireReferenceLayers(t *testing.T) {
	root := repoRoot(t)
	imports := parsedImports(t, filepath.Join(root, "internal/bootstrap/usecase.go"))

	requiredImports := []string{
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

func TestBootstrapHandlerModulesWireReferenceLayers(t *testing.T) {
	root := repoRoot(t)
	imports := parsedImports(t, filepath.Join(root, "internal/bootstrap/handler.go"))

	requiredImports := []string{
		"/internal/api/http",
		"/internal/api/http/middleware",
		"/internal/auth",
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
			t.Fatalf("bootstrap/handler.go must import %s", requiredImport)
		}
	}
}
