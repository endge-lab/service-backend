package releases

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

func TestReleaseBuildValidation(t *testing.T) {
	envelope := map[string]any{"format": "endge-bundle", "version": 1, "bundle": map[string]any{"version": 1, "programId": "build-a", "compilerVersion": "program-v4", "context": map[string]any{"project": "project-a", "configuration": map[string]any{"large": "configuration"}}, "catalog": map[string]any{"documents": map[string]any{}, "folders": map[string]any{}}, "artifacts": map[string]any{}}}
	metadata := entities.ReleaseBuildMetadata{Version: 1, ProgramID: "build-a", CompilerVersion: "program-v4", Runtime: "ts-browser", Scope: "complete-model", ContextMode: "effective-context", FileFormat: "gzip", SizeBytes: 999, Checksum: "untrusted"}
	pack := func(value any) []byte {
		t.Helper()
		raw, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		var output bytes.Buffer
		writer := gzip.NewWriter(&output)
		if _, err := writer.Write(raw); err != nil {
			t.Fatal(err)
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		return output.Bytes()
	}
	data := pack(envelope)
	actual, err := validateReleaseBuild(data, metadata)
	if err != nil {
		t.Fatal(err)
	}
	if actual.SizeBytes != int64(len(data)) || len(actual.Checksum) != 64 || string(actual.Context) != `{"project":"project-a"}` {
		t.Fatalf("incorrect derived metadata: %+v", actual)
	}
	t.Run("mismatched program", func(t *testing.T) {
		invalid := metadata
		invalid.ProgramID = "other"
		if _, err := validateReleaseBuild(data, invalid); err == nil {
			t.Fatal("accepted another program")
		}
	})
	t.Run("unsupported metadata", func(t *testing.T) {
		invalid := metadata
		invalid.Version = 2
		if _, err := validateReleaseBuild(data, invalid); err == nil {
			t.Fatal("accepted future metadata")
		}
	})
	t.Run("truncated gzip", func(t *testing.T) {
		if _, err := validateReleaseBuild(data[:len(data)-4], metadata); err == nil {
			t.Fatal("accepted truncated gzip")
		}
	})
	t.Run("corrupt gzip", func(t *testing.T) {
		corrupt := bytes.Clone(data)
		corrupt[len(corrupt)-8] ^= 1
		if _, err := validateReleaseBuild(corrupt, metadata); err == nil {
			t.Fatal("accepted invalid CRC")
		}
	})
	t.Run("inspection forbidden", func(t *testing.T) {
		envelope["inspection"] = map[string]any{}
		if _, err := validateReleaseBuild(pack(envelope), metadata); err == nil {
			t.Fatal("accepted runtime recording")
		}
		delete(envelope, "inspection")
	})
	t.Run("unsupported container", func(t *testing.T) {
		envelope["version"] = 2
		if _, err := validateReleaseBuild(pack(envelope), metadata); err == nil {
			t.Fatal("accepted future container")
		}
		envelope["version"] = 1
	})
	t.Run("compressed limit", func(t *testing.T) {
		if _, err := validateReleaseBuild(make([]byte, MaxReleaseBuildBytes+1), metadata); err == nil {
			t.Fatal("accepted oversized input")
		}
	})
}
