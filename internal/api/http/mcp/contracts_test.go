package mcp

import (
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	validator "github.com/endge-lab/service-kit-go/pkg/validator"
)

// TestContractsCoverDocumentCollections не даёт добавить коллекцию в MCP без DTO REST.
func TestContractsCoverDocumentCollections(t *testing.T) {
	contracts := BindContracts(validator.NewValidator())
	for _, collection := range entities.DocumentCollections {
		contract, ok := contracts.Documents[collection]
		if !ok {
			t.Fatalf("collection %s has no REST contract", collection)
		}
		if err := contract.Patch.Validate([]byte(`{}`)); err != nil {
			t.Fatalf("collection %s: empty patch must pass DTO validation: %v", collection, err)
		}
		if err := contract.Create.Validate([]byte(`{}`)); err == nil {
			t.Fatalf("collection %s: create without identity must fail", collection)
		}
	}
	if len(contracts.Documents) != len(entities.DocumentCollections) {
		t.Fatalf("contracts %d, collections %d", len(contracts.Documents), len(entities.DocumentCollections))
	}
	if err := contracts.Commit.Validate([]byte(`{"message":"m","expectedHeadSequence":0}`)); err != nil {
		t.Fatalf("valid commit body: %v", err)
	}
}
