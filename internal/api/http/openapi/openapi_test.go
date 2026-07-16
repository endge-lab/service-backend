package openapi

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestGeneratedOpenAPISpecMatchesCheckedInContract(t *testing.T) {
	contractPath := filepath.Join("..", "..", "..", "..", "docs", "openapi3.yaml")
	contract, err := os.ReadFile(contractPath)
	if err != nil {
		t.Fatalf("read checked-in OpenAPI contract: %v", err)
	}

	if !bytes.Equal(openAPI3YAML, contract) {
		t.Fatal("generated OpenAPI bytes are stale; run make docs")
	}
}
