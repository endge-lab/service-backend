package postgres

import (
	"strings"
	"testing"
)

// TestReleaseMetadataAndArtifactSelectsStaySeparate защищает обычные release GET/list/create
// от случайного возврата большого JSONB data в metadata query. Artifact разрешено читать
// только отдельным SQL path, который использует ReleaseArtifactReader.
func TestReleaseMetadataAndArtifactSelectsStaySeparate(t *testing.T) {
	metadata := releaseMetadataSelect()
	for _, field := range []string{
		"r.id::text", "r.workspace_id::text", "r.identity", "r.display_name", "r.source_commit_id::text",
		"r.head_sequence", "r.schema_version", "r.checksum", "r.created_at",
	} {
		if !strings.Contains(metadata, field) {
			t.Fatalf("metadata select does not contain %q: %s", field, metadata)
		}
	}
	if strings.Contains(strings.ToLower(metadata), "r.data") {
		t.Fatalf("metadata select must not read release data: %s", metadata)
	}

	artifact := releaseArtifactSelect()
	for _, field := range []string{"id::text", "workspace_id::text", "identity", "checksum", "data"} {
		if !strings.Contains(artifact, field) {
			t.Fatalf("artifact select does not contain %q: %s", field, artifact)
		}
	}
}
