// Command openapienrich дополняет сгенерированный swag OpenAPI-документ сведениями для агентов:
// описаниями свойств из docs/openapi-glossary.yaml, примерами тел из docs/openapi-examples.yaml,
// описанием BearerAuth и info.version из APP_VERSION файла VERSION.
//
// Существующие описания и примеры из аннотаций кода не перезаписываются. Команда завершается
// с ошибкой, если у свойства нет описания или в словарях остались записи, которые ничему не соответствуют.
package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"os"
	"strings"

	"go.yaml.in/yaml/v3"
)

func main() {
	input := flag.String("input", "docs/openapi3.yaml", "OpenAPI-документ, который обогащается на месте")
	glossaryPath := flag.String("glossary", "docs/openapi-glossary.yaml", "словарь описаний")
	examplesPath := flag.String("examples", "docs/openapi-examples.yaml", "примеры тел по operationId")
	versionPath := flag.String("version-file", "VERSION", "файл VERSION с APP_VERSION")
	flag.Parse()

	if err := run(*input, *glossaryPath, *examplesPath, *versionPath); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "openapienrich: %v\n", err)
		os.Exit(1)
	}
}

func run(inputPath, glossaryPath, examplesPath, versionPath string) error {
	raw, err := os.ReadFile(inputPath)
	if err != nil {
		return err
	}
	var document yaml.Node
	if err := yaml.Unmarshal(raw, &document); err != nil {
		return fmt.Errorf("parse %s: %w", inputPath, err)
	}
	var words glossary
	if err := decode(glossaryPath, &words, true); err != nil {
		return err
	}
	// Примеры содержат произвольный YAML (yaml.Node), поэтому строгий режим к ним не применяется;
	// пустые записи отлавливаются ниже.
	var samples examples
	if err := decode(examplesPath, &samples, false); err != nil {
		return err
	}
	for id, entry := range samples.Operations {
		if entry.Request == nil && entry.Response == nil {
			return fmt.Errorf("%s: у %s нет ни request, ни response", examplesPath, id)
		}
	}
	version, err := readAppVersion(versionPath)
	if err != nil {
		return err
	}

	result, err := enrich(&document, words, samples, version)
	if err != nil {
		return err
	}
	var problems []string
	for _, name := range result.Undescribed {
		problems = append(problems, "нет описания свойства "+name+" (добавьте в properties или schemas словаря)")
	}
	for _, name := range result.UnusedGlossary {
		problems = append(problems, "свойство "+name+" из properties словаря не встречается в OpenAPI")
	}
	for _, name := range result.UnusedOverrides {
		problems = append(problems, "переопределение "+name+" из schemas словаря ничему не соответствует или перекрыто описанием из кода")
	}
	for _, id := range result.UnusedExamples {
		problems = append(problems, "пример для operationId "+id+" не применён: операции нет или у неё уже есть пример из кода")
	}
	if len(problems) > 0 {
		return fmt.Errorf("словари OpenAPI не согласованы:\n  %s", strings.Join(problems, "\n  "))
	}

	var output bytes.Buffer
	encoder := yaml.NewEncoder(&output)
	encoder.SetIndent(2)
	if err := encoder.Encode(&document); err != nil {
		return err
	}
	if err := encoder.Close(); err != nil {
		return err
	}
	if err := os.WriteFile(inputPath, output.Bytes(), 0o644); err != nil {
		return err
	}
	fmt.Printf("openapienrich: version %s; described properties %d/%d; operations with body examples %d/%d\n",
		version, result.DescribedProperties, result.Properties, result.ExampleOperations, result.BodyOperations)
	return nil
}

func decode(path string, target any, strict bool) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	decoder.KnownFields(strict)
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}

func readAppVersion(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if value, ok := strings.CutPrefix(strings.TrimSpace(scanner.Text()), "APP_VERSION="); ok {
			return strings.TrimSpace(value), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("APP_VERSION is missing in %s", path)
}
