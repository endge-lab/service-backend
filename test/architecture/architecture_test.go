package architecture_test

import (
	"go/ast"
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

// TestTemplateRequiredPathsExist проверяет обязательные каталоги слоёв и checked-in OpenAPI.
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

// TestDomainMigrationOrder фиксирует непрерывную последовательность всех MVP-миграций.
func TestDomainMigrationOrder(t *testing.T) {
	root := repoRoot(t)
	expected := []string{
		"000001_service_users.sql",
		"000002_integrations.sql",
		"000003_workspaces.sql",
		"000004_workspace_memberships.sql",
		"000005_workspace_integrations.sql",
		"000006_folders.sql",
		"000007_projects.sql",
		"000008_tenants.sql",
		"000009_environments.sql",
		"000010_project_environments.sql",
		"000011_auth_profiles.sql",
		"000012_i18n_bundles.sql",
		"000013_navigations.sql",
		"000014_styles.sql",
		"000015_vocabs.sql",
		"000016_types.sql",
		"000017_queries.sql",
		"000018_data_views.sql",
		"000019_stores.sql",
		"000020_streams.sql",
		"000021_updates.sql",
		"000022_mocks.sql",
		"000023_components.sql",
		"000024_filters.sql",
		"000025_computations.sql",
		"000026_compositions.sql",
		"000027_actions.sql",
		"000028_converters.sql",
		"000029_mutation_batches.sql",
		"000030_workspace_commits.sql",
		"000031_document_revisions.sql",
		"000032_workspace_commit_changes.sql",
		"000033_workspace_commit_integrations.sql",
		"000034_releases.sql",
		"000035_bootstrap.sql",
		"000036_workspace_snapshots.sql",
		"000037_configurator_auth_sessions.sql",
		"000038_default_workspace_configuration.sql",
		"000039_bootstrap_default_domain.sql",
		"000040_project_navigation.sql",
		"000041_configurator_auth_session_identity.sql",
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

// TestTemplateLayerPackageNames не позволяет транспортным и прикладным пакетам смешиваться.
func TestTemplateLayerPackageNames(t *testing.T) {
	root := repoRoot(t)

	expectedPackages := map[string]string{
		"internal/api/http":                   "http",
		"internal/api/http/configurator_auth": "configurator_auth",
		"internal/api/http/health":            "health",
		"internal/api/http/middleware":        "middleware",
		"internal/api/http/openapi":           "openapi",
		"internal/api/http/respond":           "respond",
		"internal/api/http/v1/action":         "action",
		"internal/api/http/v1/auth_profile":   "auth_profile",
		"internal/api/http/v1/backup":         "backup",
		"internal/api/http/v1/commit":         "commit",
		"internal/api/http/v1/component":      "component",
		"internal/api/http/v1/composition":    "composition",
		"internal/api/http/v1/computation":    "computation",
		"internal/api/http/v1/converter":      "converter",
		"internal/api/http/v1/data_view":      "data_view",
		"internal/api/http/v1/domain":         "domain",
		"internal/api/http/v1/environment":    "environment",
		"internal/api/http/v1/filter":         "filter",
		"internal/api/http/v1/folder":         "folder",
		"internal/api/http/v1/i18n_bundle":    "i18n_bundle",
		"internal/api/http/v1/integration":    "integration",
		"internal/api/http/v1/mock":           "mock",
		"internal/api/http/v1/navigation":     "navigation",
		"internal/api/http/v1/project":        "project",
		"internal/api/http/v1/query":          "query",
		"internal/api/http/v1/release":        "release",
		"internal/api/http/v1/revision":       "revision",
		"internal/api/http/v1/session":        "session",
		"internal/api/http/v1/shared":         "shared",
		"internal/api/http/v1/store":          "store",
		"internal/api/http/v1/stream":         "stream",
		"internal/api/http/v1/style":          "style",
		"internal/api/http/v1/tenant":         "tenant",
		"internal/api/http/v1/type":           "domain_type",
		"internal/api/http/v1/update":         "update",
		"internal/api/http/v1/vocab":          "vocab",
		"internal/api/http/v1/workspace":      "workspace",
		"internal/bootstrap":                  "bootstrap",
		"internal/domain/entities":            "entities",
		"internal/domain/errors":              "errors",
		"internal/platform":                   "platform",
		"internal/repo/postgres":              "postgres",
		"internal/repo/postgres/sqlc":         "sqlc",
		"internal/usecase/actions":            "actions",
		"internal/usecase/auth_profiles":      "auth_profiles",
		"internal/usecase/backups":            "backups",
		"internal/usecase/commits":            "commits",
		"internal/usecase/components":         "components",
		"internal/usecase/compositions":       "compositions",
		"internal/usecase/computations":       "computations",
		"internal/usecase/converters":         "converters",
		"internal/usecase/data_views":         "data_views",
		"internal/usecase/documents":          "documents",
		"internal/usecase/environments":       "environments",
		"internal/usecase/filters":            "filters",
		"internal/usecase/folders":            "folders",
		"internal/usecase/history":            "history",
		"internal/usecase/shared":             "shared",
		"internal/usecase/workspace_state":    "workspace_state",
		"internal/usecase/i18n_bundles":       "i18n_bundles",
		"internal/usecase/integrations":       "integrations",
		"internal/usecase/mocks":              "mocks",
		"internal/usecase/navigations":        "navigations",
		"internal/usecase/portable":           "portable",
		"internal/usecase/ports":              "ports",
		"internal/usecase/projects":           "projects",
		"internal/usecase/queries":            "queries",
		"internal/usecase/releases":           "releases",
		"internal/usecase/revisions":          "revisions",
		"internal/usecase/session":            "session",
		"internal/usecase/stores":             "stores",
		"internal/usecase/streams":            "streams",
		"internal/usecase/styles":             "styles",
		"internal/usecase/tenants":            "tenants",
		"internal/usecase/types":              "types",
		"internal/usecase/updates":            "updates",
		"internal/usecase/vocabs":             "vocabs",
		"internal/usecase/workspaces":         "workspaces",
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

// TestDocumentTablesHaveOwningLayers проверяет владельца каждого документа во всех слоях.
func TestDocumentTablesHaveOwningLayers(t *testing.T) {
	root := repoRoot(t)
	resources := []struct {
		collection string
		usecase    string
		http       string
		repository string
		port       string
	}{
		{"projects", "projects", "project", "projects_repository.go", "ProjectRepository"},
		{"tenants", "tenants", "tenant", "tenants_repository.go", "TenantRepository"},
		{"environments", "environments", "environment", "environments_repository.go", "EnvironmentRepository"},
		{"folders", "folders", "folder", "folders_repository.go", "FolderRepository"},
		{"types", "types", "type", "types_repository.go", "TypeRepository"},
		{"queries", "queries", "query", "queries_repository.go", "QueryRepository"},
		{"data-views", "data_views", "data_view", "data_views_repository.go", "DataViewRepository"},
		{"compositions", "compositions", "composition", "compositions_repository.go", "CompositionRepository"},
		{"stores", "stores", "store", "stores_repository.go", "StoreRepository"},
		{"streams", "streams", "stream", "streams_repository.go", "StreamRepository"},
		{"updates", "updates", "update", "updates_repository.go", "UpdateRepository"},
		{"mocks", "mocks", "mock", "mocks_repository.go", "MockRepository"},
		{"components", "components", "component", "components_repository.go", "ComponentRepository"},
		{"actions", "actions", "action", "actions_repository.go", "ActionRepository"},
		{"filters", "filters", "filter", "filters_repository.go", "FilterRepository"},
		{"converters", "converters", "converter", "converters_repository.go", "ConverterRepository"},
		{"computations", "computations", "computation", "computations_repository.go", "ComputationRepository"},
		{"vocabs", "vocabs", "vocab", "vocabs_repository.go", "VocabRepository"},
		{"i18n-bundles", "i18n_bundles", "i18n_bundle", "i18n_bundles_repository.go", "I18nBundleRepository"},
		{"auth-profiles", "auth_profiles", "auth_profile", "auth_profiles_repository.go", "AuthProfileRepository"},
		{"navigations", "navigations", "navigation", "navigations_repository.go", "NavigationRepository"},
		{"styles", "styles", "style", "styles_repository.go", "StyleRepository"},
	}

	portsSource, err := os.ReadFile(filepath.Join(root, "internal/usecase/ports/documents.go"))
	if err != nil {
		t.Fatalf("read document ports: %v", err)
	}
	for _, resource := range resources {
		paths := []string{
			filepath.Join("internal/usecase", resource.usecase, "usecase.go"),
			filepath.Join("internal/api/http/v1", resource.http, "handler.go"),
			filepath.Join("internal/api/http/v1", resource.http, "routes.go"),
			filepath.Join("internal/api/http/v1", resource.http, "transport.go"),
			filepath.Join("internal/api/http/v1", resource.http, "usecase.go"),
			filepath.Join("internal/repo/postgres", resource.repository),
		}
		for _, relativePath := range paths {
			if _, err := os.Stat(filepath.Join(root, relativePath)); err != nil {
				t.Errorf("collection %s has no owning layer file %s", resource.collection, relativePath)
			}
		}
		if !strings.Contains(string(portsSource), "type "+resource.port+" interface") {
			t.Errorf("collection %s has no repository port %s", resource.collection, resource.port)
		}
	}
}

// TestUseCaseNamingConvention закрепляет согласованные имена UseCase без Service и Interactor.
func TestUseCaseNamingConvention(t *testing.T) {
	root := repoRoot(t)
	for _, filePath := range listGoFiles(t, root, "internal/usecase") {
		if filepath.Base(filePath) == "service.go" {
			t.Fatalf("application implementation file must describe its role instead of using service.go: %s", filePath)
		}

		fileSet := token.NewFileSet()
		parsed, err := parser.ParseFile(fileSet, filePath, nil, 0)
		if err != nil {
			t.Fatalf("parse usecase file %s: %v", filePath, err)
		}
		for _, declaration := range parsed.Decls {
			switch typed := declaration.(type) {
			case *ast.GenDecl:
				for _, specification := range typed.Specs {
					typeSpec, ok := specification.(*ast.TypeSpec)
					if ok && typeSpec.Name.Name == "Service" {
						t.Fatalf("application type must be UseCase or have a responsibility-specific name in %s", filePath)
					}
				}
			case *ast.FuncDecl:
				if typed.Name.Name == "NewService" {
					t.Fatalf("application constructor must be NewUseCase or have a responsibility-specific name in %s", filePath)
				}
			}
		}
	}
}

// TestHTTPResourcePackagesOwnTheirAdapters проверяет локальное владение handler, routes, transport и port.
func TestHTTPResourcePackagesOwnTheirAdapters(t *testing.T) {
	root := repoRoot(t)
	v1Root := filepath.Join(root, "internal/api/http/v1")
	entries, err := os.ReadDir(v1Root)
	if err != nil {
		t.Fatalf("read HTTP v1 packages: %v", err)
	}

	requiredFiles := []string{"handler.go", "routes.go", "transport.go", "usecase.go"}
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "shared" {
			continue
		}
		for _, fileName := range requiredFiles {
			relativePath := filepath.Join("internal/api/http/v1", entry.Name(), fileName)
			if _, err := os.Stat(filepath.Join(root, relativePath)); err != nil {
				t.Errorf("HTTP resource package %s does not own %s", entry.Name(), fileName)
			}
		}
		if _, err := os.Stat(filepath.Join(v1Root, entry.Name(), "resource.go")); !os.IsNotExist(err) {
			t.Errorf("HTTP resource package %s still contains aggregate resource.go", entry.Name())
		}
	}

	if _, err := os.Stat(filepath.Join(v1Root, "shared", "document_handler.go")); !os.IsNotExist(err) {
		t.Fatal("shared DocumentHandler must not own resource CRUD")
	}
}

// TestHTTPConstructorsDependOnUseCasePorts запрещает handler-конструкторам зависеть от реализации use case.
func TestHTTPConstructorsDependOnUseCasePorts(t *testing.T) {
	root := repoRoot(t)
	for _, filePath := range listGoFiles(t, root, "internal/api/http/v1") {
		if filepath.Base(filePath) != "handler.go" {
			continue
		}
		fileSet := token.NewFileSet()
		parsed, err := parser.ParseFile(fileSet, filePath, nil, 0)
		if err != nil {
			t.Fatalf("parse handler %s: %v", filePath, err)
		}
		for _, declaration := range parsed.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Name.Name != "NewHandler" || len(function.Type.Params.List) == 0 {
				continue
			}
			identifier, ok := function.Type.Params.List[0].Type.(*ast.Ident)
			if !ok || identifier.Name != "UseCase" {
				t.Fatalf("NewHandler must depend on local UseCase interface in %s", filePath)
			}
		}
	}
}

// TestExportedTransportFunctionsHaveGoDoc требует GoDoc у экспортируемых функций транспорта.
func TestExportedTransportFunctionsHaveGoDoc(t *testing.T) {
	root := repoRoot(t)
	for _, filePath := range listGoFiles(t, root, "internal/api/http/v1") {
		fileSet := token.NewFileSet()
		parsed, err := parser.ParseFile(fileSet, filePath, nil, parser.ParseComments)
		if err != nil {
			t.Fatalf("parse transport %s: %v", filePath, err)
		}
		for _, declaration := range parsed.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || !function.Name.IsExported() {
				continue
			}
			if function.Doc == nil || !strings.HasPrefix(strings.TrimSpace(function.Doc.Text()), function.Name.Name) {
				t.Fatalf("exported transport function %s in %s needs Russian GoDoc beginning with its name", function.Name.Name, filePath)
			}
		}
	}
}

// TestHTTPUseCasePortsDoNotExposeUntypedMaps не допускает map[string]any на границе transport-usecase.
func TestHTTPUseCasePortsDoNotExposeUntypedMaps(t *testing.T) {
	root := repoRoot(t)
	for _, filePath := range listGoFiles(t, root, "internal/api/http/v1") {
		if filepath.Base(filePath) != "usecase.go" {
			continue
		}
		source, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("read usecase port %s: %v", filePath, err)
		}
		if strings.Contains(string(source), "map[string]any") {
			t.Fatalf("HTTP usecase port exposes map[string]any in %s", filePath)
		}
	}
}

// TestMVPPackagesAreRemoved запрещает возврат временных общих пакетов mvp.
func TestMVPPackagesAreRemoved(t *testing.T) {
	root := repoRoot(t)
	for _, relativeDir := range []string{"internal/usecase/mvp", "internal/api/http/v1/mvp"} {
		if _, err := os.Stat(filepath.Join(root, relativeDir)); os.IsNotExist(err) {
			continue
		}
		if files := listGoFiles(t, root, relativeDir); len(files) != 0 {
			t.Fatalf("legacy aggregate package %s still contains production files", relativeDir)
		}
	}
}

// TestHTTPHandlersDoNotImportRelationResolver сохраняет разрешение связей в прикладном слое.
func TestHTTPHandlersDoNotImportRelationResolver(t *testing.T) {
	root := repoRoot(t)

	for _, filePath := range listGoFiles(t, root, "internal/api/http/v1") {
		for _, importedPath := range parsedImports(t, filePath) {
			if strings.Contains(importedPath, "/internal/usecase/relations") {
				t.Fatalf("HTTP handler layer must delegate relation resolution to usecases, found %s in %s", importedPath, filePath)
			}
		}
	}
}

// TestPublicTransportDoesNotExposeForeignUUIDs закрепляет identity-based публичный контракт связей.
func TestPublicTransportDoesNotExposeForeignUUIDs(t *testing.T) {
	root := repoRoot(t)

	for _, filePath := range listGoFiles(t, root, "internal/api/http/v1") {
		if !strings.HasSuffix(filePath, "_transport.go") && !strings.HasSuffix(filePath, "transport.go") {
			continue
		}
		assertNoForeignUUIDTransportFields(t, filePath)
	}
}

func assertNoForeignUUIDTransportFields(t *testing.T, filePath string) {
	t.Helper()

	fileSet := token.NewFileSet()
	parsed, err := parser.ParseFile(fileSet, filePath, nil, 0)
	if err != nil {
		t.Fatalf("parse transport %s: %v", filePath, err)
	}

	ast.Inspect(parsed, func(node ast.Node) bool {
		field, ok := node.(*ast.Field)
		if !ok || field.Tag == nil || !strings.Contains(field.Tag.Value, "json:") || !isUUIDType(field.Type) {
			return true
		}
		for _, name := range field.Names {
			if name.Name != "ID" {
				t.Fatalf("public transport must use relation identity instead of foreign UUID %s in %s", name.Name, filePath)
			}
		}
		return true
	})
}

func isUUIDType(expression ast.Expr) bool {
	switch value := expression.(type) {
	case *ast.StarExpr:
		return isUUIDType(value.X)
	case *ast.SelectorExpr:
		identifier, ok := value.X.(*ast.Ident)
		return ok && identifier.Name == "uuid" && value.Sel.Name == "UUID"
	default:
		return false
	}
}

// TestTemplateDependencyBoundaries проверяет направленность импортов чистой архитектуры.
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

// TestBootstrapRepositoryModulesWireReferenceLayers проверяет явную сборку repository adapters.
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

// TestBootstrapUseCaseModulesWireReferenceLayers проверяет явную сборку прикладных сценариев.
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

// TestBootstrapHandlerModulesWireReferenceLayers проверяет явную сборку transport adapters.
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
