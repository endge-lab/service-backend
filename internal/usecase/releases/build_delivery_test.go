package releases

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

func TestDeliveryReleaseBuild(t *testing.T) {
	program := func(id, tenant, channel string) map[string]any {
		return map[string]any{
			"version": 1, "exportScope": "startup-composition", "executionGraphVersion": 1,
			"programId": id, "compilerVersion": "test",
			"context": map[string]any{"workspace": "source", "facets": map[string]string{"tenant": tenant, "channel": channel}},
			"catalog": map[string]any{"workspace": map[string]string{"identity": "source", "startupCompositionIdentity": "app"}},
			"artifacts": map[string]any{"composition:app": map[string]any{"payload": map[string]any{"resources": []map[string]string{
				{"kind": "module", "identity": "selected", "version": "1.0.0"},
			}}}},
		}
	}
	metadata := entities.ReleaseBuildMetadata{
		Version: 1, Runtime: "ts-browser", Scope: "startup-composition", ContextMode: "effective-context", FileFormat: "gzip", ProgramID: "first", CompilerVersion: "test",
	}
	module := func(identity string) map[string]any {
		return map[string]any{
			"manifest": map[string]any{"identity": identity, "version": "1.0.0", "formatVersion": 1, "hostApi": 1},
			"js":       "export default () => ({})",
		}
	}
	delivery := map[string]any{
		"variants": []map[string]any{program("second", "customer-a", "mobile")},
		"modules":  []map[string]any{module("selected")},
	}
	envelope := map[string]any{
		"format": "endge-bundle", "version": 2, "bundle": program("first", "customer-a", "web"), "delivery": delivery,
	}
	validate := func() error {
		var buffer bytes.Buffer
		writer := gzip.NewWriter(&buffer)
		if err := json.NewEncoder(writer).Encode(envelope); err != nil {
			t.Fatal(err)
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		_, err := validateReleaseBuild(buffer.Bytes(), metadata)
		return err
	}
	if err := validate(); err != nil {
		t.Fatal(err)
	}
	delivery["modules"] = []map[string]any{module("selected"), module("foreign")}
	if validate() == nil {
		t.Fatal("unrelated embedded module was accepted")
	}
	delivery["modules"] = []map[string]any{}
	if validate() == nil {
		t.Fatal("incomplete embedded modules were accepted")
	}
	delete(delivery, "modules")
	if err := validate(); err != nil {
		t.Fatal(err)
	}
	delivery["variants"] = []map[string]any{program("second", "customer-b", "mobile")}
	if validate() == nil {
		t.Fatal("different tenant was accepted")
	}
}

func TestDeliveryReleaseBuildFromCore(t *testing.T) {
	filename := os.Getenv("ENDGE_DELIVERY_TEST_FILE")
	if filename == "" {
		t.Skip("Requires the Core integration test artifact")
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		Bundle releaseProgram `json:"bundle"`
	}
	if err := json.NewDecoder(io.LimitReader(reader, maxDecodedBuildBytes)).Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	metadata := entities.ReleaseBuildMetadata{
		Version: 1, Runtime: "ts-browser", Scope: "startup-composition", ContextMode: "effective-context", FileFormat: "gzip",
		ProgramID: envelope.Bundle.ProgramID, CompilerVersion: envelope.Bundle.CompilerVersion,
	}
	validated, err := validateReleaseBuild(data, metadata)
	if err != nil || validated == nil {
		t.Fatalf("Core artifact rejected: %v", err)
	}
	if !bytes.Contains(validated.Context, []byte("deliveryFacets")) {
		t.Fatal("release metadata does not report every delivery context")
	}
}
