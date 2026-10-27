// Command errorcatalog собирает каталог кодов ошибок backend разбором AST (без исполнения кода)
// и объединяет его со словарём смыслов docs/error-guidance.yaml.
//
// Команда завершается с ошибкой, если в исходниках есть код без записи в словаре,
// в словаре есть код, которого нет в исходниках, или код создаётся нелитерально вне описанных мест.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	root := flag.String("root", ".", "корень backend-репозитория")
	guidancePath := flag.String("guidance", "docs/error-guidance.yaml", "словарь смыслов (относительно root)")
	outputPath := flag.String("output", "docs/error-codes.yaml", "сгенерированный каталог (относительно root)")
	list := flag.Bool("list", false, "вывести найденные коды без словаря (для заполнения error-guidance.yaml)")
	flag.Parse()

	scan, err := scanRepository(*root)
	if err != nil {
		fail("scan sources: %v", err)
	}
	if *list {
		printList(scan)
		return
	}

	guidance, err := loadGuidance(filepath.Join(*root, *guidancePath))
	if err != nil {
		fail("%v", err)
	}
	catalog, problems := buildCatalog(scan, guidance)
	if len(problems) > 0 {
		fail("каталог ошибок не согласован с исходниками:\n  %s", strings.Join(problems, "\n  "))
	}
	rendered, err := renderCatalog(catalog)
	if err != nil {
		fail("render catalog: %v", err)
	}
	target := filepath.Join(*root, *outputPath)
	if err := os.WriteFile(target, rendered, 0o644); err != nil {
		fail("write %s: %v", target, err)
	}
	fmt.Printf("errorcatalog: %d codes -> %s\n", len(catalog.Codes), *outputPath)
}

func printList(scan scanResult) {
	facts := aggregate(scan.Occurrences)
	codes := make([]string, 0, len(facts))
	for code := range facts {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	for _, code := range codes {
		fact := facts[code]
		message := ""
		if len(fact.Messages) > 0 {
			message = strings.Join(fact.Messages, " | ")
		}
		fmt.Printf("%s\t%v\t%s\t%s\n", code, fact.Statuses, message, strings.Join(fact.Sources, ","))
	}
	for _, site := range scan.Unresolved {
		fmt.Printf("UNRESOLVED\t%s:%d\t%s\n", site.File, site.Line, site.Function)
	}
}

func fail(message string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, "errorcatalog: "+message+"\n", args...)
	os.Exit(1)
}
