package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// domainErrorsImport — путь пакета, конструкторы которого создают публичные коды ошибок.
const domainErrorsImport = "github.com/endge-lab/service-backend/internal/domain/errors"

// kindStatus повторяет HTTP-статусы конструкторов internal/domain/errors.
// HTTP-слой (respond.WriteErrorResponse) отдаёт именно AppError.HTTPStatus().
var kindStatus = map[string]int{
	"InvalidInput": 400,
	"Unauthorized": 401,
	"Forbidden":    403,
	"NotFound":     404,
	"Conflict":     409,
	"Internal":     500,
}

// fiberStatus — статусы fiber.Status*, которые middleware пишет в fiber.Map напрямую.
var fiberStatus = map[string]int{
	"StatusBadRequest":          400,
	"StatusUnauthorized":        401,
	"StatusForbidden":           403,
	"StatusNotFound":            404,
	"StatusConflict":            409,
	"StatusTooManyRequests":     429,
	"StatusInternalServerError": 500,
	"StatusBadGateway":          502,
	"StatusServiceUnavailable":  503,
	"StatusGatewayTimeout":      504,
}

// occurrence — одно место в коде, где создаётся ошибка с литеральным кодом.
type occurrence struct {
	Code    string
	Status  int    // 0 — статус задаётся во время выполнения
	Message string // пусто — сообщение не литеральное
	File    string
	Line    int
}

// unresolved — вызов конструктора, код которого нельзя определить без исполнения.
type unresolved struct {
	File     string
	Line     int
	Function string
}

type scanResult struct {
	Occurrences []occurrence
	Unresolved  []unresolved
}

// scanRepository разбирает Go-исходники cmd и internal без тестов и инструментов.
func scanRepository(root string) (scanResult, error) {
	var result scanResult
	for _, dir := range []string{"cmd", "internal"} {
		base := filepath.Join(root, dir)
		err := filepath.WalkDir(base, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, relErr := filepath.Rel(root, path)
			if relErr != nil {
				return relErr
			}
			rel = filepath.ToSlash(rel)
			if entry.IsDir() {
				if rel == "internal/tools" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(rel, ".go") || strings.HasSuffix(rel, "_test.go") || strings.HasSuffix(rel, ".gen.go") {
				return nil
			}
			return scanFile(path, rel, &result)
		})
		if err != nil {
			return scanResult{}, err
		}
	}
	return result, nil
}

func scanFile(path, rel string, result *scanResult) error {
	source, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, source, 0)
	if err != nil {
		return fmt.Errorf("parse %s: %w", rel, err)
	}
	alias := domainErrorsAlias(file)
	inErrorsPackage := strings.HasPrefix(rel, "internal/domain/errors/")
	line := func(node ast.Node) int { return fset.Position(node.Pos()).Line }

	for _, decl := range file.Decls {
		function := "package scope"
		if fn, ok := decl.(*ast.FuncDecl); ok {
			function = fn.Name.Name
		}
		ast.Inspect(decl, func(node ast.Node) bool {
			switch typed := node.(type) {
			case *ast.CallExpr:
				name, ok := constructorName(typed.Fun, alias, inErrorsPackage)
				if ok {
					occ, resolved := readConstructor(name, typed)
					if !resolved && !inErrorsPackage {
						result.Unresolved = append(result.Unresolved, unresolved{File: rel, Line: line(typed), Function: function})
					} else if resolved && occ.Code != "" {
						occ.File, occ.Line = rel, line(typed)
						result.Occurrences = append(result.Occurrences, occ)
					}
				}
				if occ, ok := readFiberMapResponse(typed); ok {
					occ.File, occ.Line = rel, line(typed)
					result.Occurrences = append(result.Occurrences, occ)
				}
			case *ast.CompositeLit:
				if occ, ok := readToolError(typed); ok {
					occ.File, occ.Line = rel, line(typed)
					result.Occurrences = append(result.Occurrences, occ)
				}
			}
			return true
		})
	}
	return nil
}

// domainErrorsAlias возвращает локальное имя импорта internal/domain/errors в файле.
func domainErrorsAlias(file *ast.File) string {
	for _, spec := range file.Imports {
		importPath, _ := strconv.Unquote(spec.Path.Value)
		if importPath != domainErrorsImport {
			continue
		}
		if spec.Name != nil {
			return spec.Name.Name
		}
		return "errors"
	}
	return ""
}

// constructorName распознаёт domainerrors.<Kind>/New/Wrap/Code или неквалифицированный вызов внутри пакета errors.
func constructorName(fun ast.Expr, alias string, inErrorsPackage bool) (string, bool) {
	var name string
	switch typed := fun.(type) {
	case *ast.SelectorExpr:
		pkg, ok := typed.X.(*ast.Ident)
		if !ok || alias == "" || pkg.Name != alias {
			return "", false
		}
		name = typed.Sel.Name
	case *ast.Ident:
		if !inErrorsPackage {
			return "", false
		}
		name = typed.Name
	default:
		return "", false
	}
	if _, ok := kindStatus[name]; ok || name == "New" || name == "Wrap" || name == "Code" {
		return name, true
	}
	return "", false
}

// readConstructor извлекает код, статус и сообщение. resolved=false — код не литеральный.
func readConstructor(name string, call *ast.CallExpr) (occurrence, bool) {
	args := call.Args
	switch name {
	case "Code":
		if len(args) != 1 {
			return occurrence{}, false
		}
		code, ok := stringLiteral(args[0])
		return occurrence{Code: code}, ok
	case "New":
		return readCodeMessageStatus(args, 0)
	case "Wrap":
		return readCodeMessageStatus(args, 1)
	default:
		if len(args) != 2 {
			return occurrence{}, false
		}
		code, ok := stringLiteral(args[0])
		if !ok {
			return occurrence{}, false
		}
		message, _ := stringLiteral(args[1])
		return occurrence{Code: code, Status: kindStatus[name], Message: message}, true
	}
}

func readCodeMessageStatus(args []ast.Expr, offset int) (occurrence, bool) {
	if len(args) != offset+3 {
		return occurrence{}, false
	}
	code, ok := stringLiteral(args[offset])
	if !ok {
		return occurrence{}, false
	}
	message, _ := stringLiteral(args[offset+1])
	status := 0
	if literal, ok := args[offset+2].(*ast.BasicLit); ok && literal.Kind == token.INT {
		status, _ = strconv.Atoi(literal.Value)
	}
	return occurrence{Code: code, Status: status, Message: message}, true
}

// readFiberMapResponse распознаёт c.Status(fiber.StatusX).JSON(fiber.Map{"code": "...", "message": "..."}).
func readFiberMapResponse(call *ast.CallExpr) (occurrence, bool) {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "JSON" || len(call.Args) != 1 {
		return occurrence{}, false
	}
	literal, ok := call.Args[0].(*ast.CompositeLit)
	if !ok || !isSelector(literal.Type, "fiber", "Map") {
		return occurrence{}, false
	}
	fields := map[string]string{}
	for _, element := range literal.Elts {
		pair, ok := element.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, keyOK := stringLiteral(pair.Key)
		value, valueOK := stringLiteral(pair.Value)
		if keyOK && valueOK {
			fields[key] = value
		}
	}
	code := fields["code"]
	if code == "" {
		return occurrence{}, false
	}
	status := 0
	if statusCall, ok := selector.X.(*ast.CallExpr); ok && len(statusCall.Args) == 1 {
		if statusSelector, ok := statusCall.Args[0].(*ast.SelectorExpr); ok && isSelector(statusSelector, "fiber", statusSelector.Sel.Name) {
			status = fiberStatus[statusSelector.Sel.Name]
		}
	}
	return occurrence{Code: code, Status: status, Message: fields["message"]}, true
}

// readToolError распознаёт toolError{Error: "..."} — безопасный код MCP-инструмента.
func readToolError(literal *ast.CompositeLit) (occurrence, bool) {
	ident, ok := literal.Type.(*ast.Ident)
	if !ok || ident.Name != "toolError" {
		return occurrence{}, false
	}
	var occ occurrence
	for _, element := range literal.Elts {
		pair, ok := element.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := pair.Key.(*ast.Ident)
		if !ok {
			continue
		}
		value, ok := stringLiteral(pair.Value)
		if !ok {
			continue
		}
		switch key.Name {
		case "Error":
			occ.Code = value
		case "Message":
			occ.Message = value
		}
	}
	return occ, occ.Code != ""
}

func isSelector(expr ast.Expr, pkg, name string) bool {
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	ident, ok := selector.X.(*ast.Ident)
	return ok && ident.Name == pkg && selector.Sel.Name == name
}

func stringLiteral(expr ast.Expr) (string, bool) {
	literal, ok := expr.(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		return "", false
	}
	value, err := strconv.Unquote(literal.Value)
	return value, err == nil
}

// codeFacts — сведения о коде, собранные из всех мест создания.
type codeFacts struct {
	Statuses []int
	Messages []string // по убыванию частоты
	Sources  []string
}

func aggregate(occurrences []occurrence) map[string]*codeFacts {
	type counter struct {
		statuses map[int]bool
		messages map[string]int
		sources  map[string]bool
	}
	counters := map[string]*counter{}
	for _, occ := range occurrences {
		current := counters[occ.Code]
		if current == nil {
			current = &counter{statuses: map[int]bool{}, messages: map[string]int{}, sources: map[string]bool{}}
			counters[occ.Code] = current
		}
		if occ.Status != 0 {
			current.statuses[occ.Status] = true
		}
		if occ.Message != "" {
			current.messages[occ.Message]++
		}
		current.sources[occ.File] = true
	}
	facts := make(map[string]*codeFacts, len(counters))
	for code, current := range counters {
		fact := &codeFacts{}
		for status := range current.statuses {
			fact.Statuses = append(fact.Statuses, status)
		}
		sort.Ints(fact.Statuses)
		for message := range current.messages {
			fact.Messages = append(fact.Messages, message)
		}
		sort.Slice(fact.Messages, func(i, j int) bool {
			left, right := fact.Messages[i], fact.Messages[j]
			if current.messages[left] != current.messages[right] {
				return current.messages[left] > current.messages[right]
			}
			return left < right
		})
		for source := range current.sources {
			fact.Sources = append(fact.Sources, source)
		}
		sort.Strings(fact.Sources)
		facts[code] = fact
	}
	return facts
}
