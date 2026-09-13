//go:build e2e

package e2e_test

import (
	"bytes"
	"encoding/base64"
	"net/http"
	"strings"
	"testing"

	platformencryption "github.com/endge-lab/service-backend/internal/platform/encryption"
	"github.com/endge-lab/service-backend/test/support"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func TestWorkspacePortabilityRoundTripWithBuildProfilesAndPrivateAI(t *testing.T) {
	provider := support.NewIdentityProvider(t)
	platformToken := bearer(provider.Token(t, support.TokenInput{
		Subject: "portable-platform", Username: "portable-platform", DisplayName: "Portable Platform", Groups: []string{"endge-platform-admins"},
	}))
	adminToken := bearer(provider.Token(t, support.TokenInput{
		Subject: "portable-admin", Username: "portable-admin", DisplayName: "Portable Admin",
	}))

	sourceDatabase := postgresSuite.NewDatabase(t)
	sourceConfig := support.OIDCConfig(provider)
	sourceConfig.WorkspaceSchemaVersion = 10
	sourceApp := support.NewTestApp(t, sourceDatabase, sourceConfig)
	_ = currentUserID(t, sourceApp, platformToken)
	sourceAdminID := currentUserID(t, sourceApp, adminToken)
	createWorkspace(t, sourceApp, platformToken, "portable-source")
	putMembership(t, sourceApp, platformToken, "portable-source", sourceAdminID, "admin")
	sourceAdmin := workspaceHeaders(adminToken, "portable-source")
	sourcePlatform := workspaceHeaders(platformToken, "portable-source")

	shared := decodeObject(t, mustStatus(t, perform(t, sourceApp, http.MethodPost, "/api/v1/build-profiles", buildProfilePayload("shared"), sourceAdmin), fiber.StatusCreated))
	private := decodeObject(t, mustStatus(t, perform(t, sourceApp, http.MethodPost, "/api/v1/build-profiles", buildProfilePayload("private"), sourceAdmin), fiber.StatusCreated))
	privateAISecret := "portable-private-secret"
	privateConnection := decodeObject(t, mustStatus(t, perform(t, sourceApp, http.MethodPost, "/api/v1/ai/provider-connections", map[string]any{
		"name": "Portable private", "adapter": "anthropic", "baseUrl": "https://private.example.test",
		"credential": privateAISecret, "visibility": "private", "enabled": true,
	}, sourceAdmin), fiber.StatusCreated))
	privateConnectionID := stringField(t, privateConnection, "id")
	mustStatus(t, perform(t, sourceApp, http.MethodPost, "/api/v1/ai/model-profiles", map[string]any{
		"connectionId": privateConnectionID, "providerModelId": "portable-model", "displayName": "Portable model", "enabled": true,
	}, sourceAdmin), fiber.StatusCreated).Body.Close()

	publicSecret := "portable-public-secret"
	mustStatus(t, perform(t, sourceApp, http.MethodPost, "/api/v1/ai/provider-connections", map[string]any{
		"name": "Portable public", "adapter": "ollama", "baseUrl": "http://public.example.test",
		"credential": publicSecret, "visibility": "public", "enabled": true,
	}, sourcePlatform), fiber.StatusCreated).Body.Close()
	mustStatus(t, perform(t, sourceApp, http.MethodPost, "/api/v1/queries", map[string]any{
		"identity": "portable-query", "displayName": "Portable query", "source": "query {}", "sourceVersion": 2,
	}, sourceAdmin), fiber.StatusCreated).Body.Close()
	commit := decodeObject(t, mustStatus(t, perform(t, sourceApp, http.MethodPost, "/api/v1/commits", map[string]any{
		"message": "Portable baseline", "revisionPolicy": "preserve", "expectedHeadSequence": currentHeadSequence(t, sourceApp, sourceAdmin),
	}, sourceAdmin), fiber.StatusCreated))
	mustStatus(t, perform(t, sourceApp, http.MethodPost, "/api/v1/releases", map[string]any{
		"identity": "portable-release", "displayName": "Portable release", "sourceCommitId": stringField(t, commit, "id"),
	}, sourceAdmin), fiber.StatusCreated).Body.Close()
	releaseArtifact := decodeObject(t, mustStatus(t, perform(t, sourceApp, http.MethodGet, "/api/v1/releases/portable-release/export", nil, sourceAdmin), fiber.StatusOK))
	if releaseArtifact["buildProfiles"] != nil || releaseArtifact["aiCatalog"] != nil || strings.Contains(stringifyJSON(t, releaseArtifact), privateAISecret) {
		t.Fatalf("release artifact contains operational profile or AI data: %#v", releaseArtifact)
	}
	mustStatus(t, perform(t, sourceApp, http.MethodPost, "/api/v1/domain/backups", map[string]any{
		"description": "Portable isolation",
	}, sourceAdmin), fiber.StatusCreated).Body.Close()
	backupArtifact := decodeObject(t, mustStatus(t, perform(t, sourceApp, http.MethodGet, "/api/v1/domain/backups/last/export", nil, sourceAdmin), fiber.StatusOK))
	if backupArtifact["buildProfiles"] != nil || backupArtifact["aiCatalog"] != nil || strings.Contains(stringifyJSON(t, backupArtifact), privateAISecret) {
		t.Fatalf("backup artifact contains operational profile or AI data: %#v", backupArtifact)
	}

	standard := decodeObject(t, perform(t, sourceApp, http.MethodGet, "/api/v1/domain/export", nil, sourceAdmin))
	standardProfiles, _ := standard["buildProfiles"].([]any)
	if len(standardProfiles) != 1 || strings.Contains(stringifyJSON(t, standard), privateAISecret) || standard["aiCatalog"] != nil {
		t.Fatalf("safe GET export contains private or AI data: %#v", standard)
	}
	standardShared, _ := standardProfiles[0].(map[string]any)
	if stringField(t, standardShared, "identity") != stringField(t, shared, "identity") {
		t.Fatalf("safe GET export did not include only the shared profile: %#v", standardProfiles)
	}

	plainExportResponse := perform(t, sourceApp, http.MethodPost, "/api/v1/domain/export", map[string]any{
		"privateBuildProfiles": "own", "privateAIConnections": "own", "includePublicAI": false,
	}, sourceAdmin)
	assertStatus(t, plainExportResponse, fiber.StatusOK)
	plainExport := decodeObject(t, plainExportResponse)
	if stringField(t, plainExport, "domainVersion") != stringField(t, standard, "domainVersion") {
		t.Fatalf("operational data changed domainVersion: safe=%q extended=%q", stringField(t, standard, "domainVersion"), stringField(t, plainExport, "domainVersion"))
	}
	plainProfiles, _ := plainExport["buildProfiles"].([]any)
	if len(plainProfiles) != 2 || !strings.Contains(stringifyJSON(t, plainExport), privateAISecret) || strings.Contains(stringifyJSON(t, plainExport), publicSecret) {
		t.Fatalf("extended export selection is incorrect: %#v", plainExport)
	}
	regularAI := decodeObject(t, perform(t, sourceApp, http.MethodGet, "/api/v1/ai/provider-connections", nil, sourceAdmin))
	if strings.Contains(stringifyJSON(t, regularAI), privateAISecret) {
		t.Fatal("regular AI API leaked a credential")
	}

	deniedPublic := perform(t, sourceApp, http.MethodPost, "/api/v1/domain/export", map[string]any{
		"privateBuildProfiles": "own", "privateAIConnections": "own", "includePublicAI": true,
	}, sourceAdmin)
	assertStatus(t, deniedPublic, fiber.StatusForbidden)
	deniedPublic.Body.Close()
	platformExport := decodeObject(t, mustStatus(t, perform(t, sourceApp, http.MethodPost, "/api/v1/domain/export", map[string]any{
		"privateBuildProfiles": "all", "privateAIConnections": "all", "includePublicAI": true,
	}, sourcePlatform), fiber.StatusOK))

	password := "portable artifact password"
	encryptedExport := decodeObject(t, mustStatus(t, perform(t, sourceApp, http.MethodPost, "/api/v1/domain/export", map[string]any{
		"privateBuildProfiles": "own", "privateAIConnections": "own", "includePublicAI": false, "password": password,
	}, sourceAdmin), fiber.StatusOK))
	if stringField(t, encryptedExport, "kind") != "endge-encrypted-workspace" || strings.Contains(stringifyJSON(t, encryptedExport), privateAISecret) {
		t.Fatalf("encrypted export envelope is incorrect: %#v", encryptedExport)
	}

	var sourceCiphertext []byte
	if err := sourceDatabase.Pool.QueryRow(t.Context(), `SELECT credential_encrypted FROM ai_provider_connections WHERE id=$1`, privateConnectionID).Scan(&sourceCiphertext); err != nil {
		t.Fatalf("read source credential: %v", err)
	}

	targetDatabase := postgresSuite.NewDatabase(t)
	targetConfig := support.OIDCConfig(provider)
	targetConfig.WorkspaceSchemaVersion = 10
	targetKey := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x5a}, 32))
	targetConfig.Encryption.KeyID = "target-v1"
	targetConfig.Encryption.Key = targetKey
	targetApp := support.NewTestApp(t, targetDatabase, targetConfig)
	_ = currentUserID(t, targetApp, platformToken)
	targetAdminID := currentUserID(t, targetApp, adminToken)
	createWorkspace(t, targetApp, platformToken, "portable-target")
	putMembership(t, targetApp, platformToken, "portable-target", targetAdminID, "admin")
	targetAdmin := workspaceHeaders(adminToken, "portable-target")

	publicImportDenied := perform(t, targetApp, http.MethodPost, "/api/v1/domain/import/plan", platformExport, targetAdmin)
	assertStatus(t, publicImportDenied, fiber.StatusForbidden)
	publicImportDenied.Body.Close()

	plan := decodeObject(t, mustStatus(t, perform(t, targetApp, http.MethodPost, "/api/v1/domain/import/plan", map[string]any{
		"artifact": encryptedExport, "password": password,
	}, targetAdmin), fiber.StatusOK))
	if plan["valid"] != true {
		t.Fatalf("encrypted portability plan is invalid: %#v", plan)
	}
	incoming := objectField(t, plan, "incoming")
	if numberField(t, incoming, "buildProfiles") != 2 || numberField(t, incoming, "aiConnections") != 1 || numberField(t, incoming, "aiModels") != 1 {
		t.Fatalf("encrypted plan counts are incorrect: %#v", incoming)
	}
	imported := decodeObject(t, mustStatus(t, perform(t, targetApp, http.MethodPost, "/api/v1/domain/import", map[string]any{
		"planId": stringField(t, plan, "planId"), "confirmation": "portable-target",
	}, withIfMatch(targetAdmin, stringField(t, plan, "targetETag"))), fiber.StatusCreated))
	importedCounts := objectField(t, imported, "imported")
	if numberField(t, importedCounts, "buildProfiles") != 2 || numberField(t, importedCounts, "aiConnections") != 1 || numberField(t, importedCounts, "aiModels") != 1 {
		t.Fatalf("import counts are incorrect: %#v", importedCounts)
	}

	targetProfiles := listItems(t, decodeObject(t, perform(t, targetApp, http.MethodGet, "/api/v1/build-profiles", nil, targetAdmin)))
	if len(targetProfiles) != 2 {
		t.Fatalf("target build profiles=%d, want 2", len(targetProfiles))
	}
	var targetConnectionID string
	var targetCiphertext []byte
	if err := targetDatabase.Pool.QueryRow(t.Context(), `SELECT id::text,credential_encrypted FROM ai_provider_connections WHERE owner_user_id=$1 AND name='Portable private'`, targetAdminID).Scan(&targetConnectionID, &targetCiphertext); err != nil {
		t.Fatalf("read target credential: %v", err)
	}
	if bytes.Equal(sourceCiphertext, targetCiphertext) || bytes.Contains(targetCiphertext, []byte(privateAISecret)) {
		t.Fatal("target credential was copied or stored as plaintext instead of re-encrypted")
	}
	targetKeyring, err := platformencryption.NewKeyring(platformencryption.Config{
		Current: platformencryption.KeyConfig{ID: "target-v1", Key: targetKey},
	})
	if err != nil {
		t.Fatalf("create target keyring: %v", err)
	}
	decrypted, err := targetKeyring.Decrypt(targetCiphertext, []byte("ai-provider-credential:"+targetConnectionID))
	if err != nil || decrypted != privateAISecret {
		t.Fatalf("target credential was not encrypted with target key: value=%q err=%v", decrypted, err)
	}
	targetRegularAI := decodeObject(t, perform(t, targetApp, http.MethodGet, "/api/v1/ai/provider-connections", nil, targetAdmin))
	if strings.Contains(stringifyJSON(t, targetRegularAI), privateAISecret) {
		t.Fatal("target regular AI API leaked imported credential")
	}

	profiles := plainExport["buildProfiles"].([]any)
	mergeIdentity := stringField(t, profiles[0].(map[string]any), "identity")
	profiles[0].(map[string]any)["displayName"] = "Профиль после merge"
	mergePlan := decodeObject(t, mustStatus(t, perform(t, targetApp, http.MethodPost, "/api/v1/domain/import/plan", plainExport, targetAdmin), fiber.StatusOK))
	mergeResult := perform(t, targetApp, http.MethodPost, "/api/v1/domain/import", map[string]any{
		"planId": stringField(t, mergePlan, "planId"), "confirmation": "portable-target",
	}, withIfMatch(targetAdmin, stringField(t, mergePlan, "targetETag")))
	assertStatus(t, mergeResult, fiber.StatusCreated)
	mergeResult.Body.Close()
	mergedProfiles := listItems(t, decodeObject(t, perform(t, targetApp, http.MethodGet, "/api/v1/build-profiles", nil, targetAdmin)))
	if len(mergedProfiles) != 2 || !hasIdentityWithName(mergedProfiles, mergeIdentity, "Профиль после merge") {
		t.Fatalf("profile merge did not upsert by identity: %#v", mergedProfiles)
	}

	_ = private
}

func TestWorkspacePortabilityLoginAmbiguityAndAtomicRollback(t *testing.T) {
	database := postgresSuite.NewDatabase(t)
	provider := support.NewIdentityProvider(t)
	configuration := support.OIDCConfig(provider)
	configuration.WorkspaceSchemaVersion = 10
	app := support.NewTestApp(t, database, configuration)
	platform := bearer(provider.Token(t, support.TokenInput{
		Subject: "rollback-platform", Username: "rollback-platform", DisplayName: "Rollback Platform", Groups: []string{"endge-platform-admins"},
	}))
	admin := bearer(provider.Token(t, support.TokenInput{
		Subject: "rollback-admin", Username: "rollback-admin", DisplayName: "Rollback Admin",
	}))
	_ = currentUserID(t, app, platform)
	adminID := currentUserID(t, app, admin)
	createWorkspace(t, app, platform, "rollback-source")
	createWorkspace(t, app, platform, "rollback-target")
	putMembership(t, app, platform, "rollback-source", adminID, "admin")
	putMembership(t, app, platform, "rollback-target", adminID, "admin")
	source := workspaceHeaders(admin, "rollback-source")
	target := workspaceHeaders(admin, "rollback-target")
	mustStatus(t, perform(t, app, http.MethodPost, "/api/v1/build-profiles", buildProfilePayload("shared"), source), fiber.StatusCreated).Body.Close()
	mustStatus(t, perform(t, app, http.MethodPost, "/api/v1/build-profiles", buildProfilePayload("private"), source), fiber.StatusCreated).Body.Close()
	mustStatus(t, perform(t, app, http.MethodPost, "/api/v1/ai/provider-connections", map[string]any{
		"name": "Rollback private", "adapter": "anthropic", "baseUrl": "https://rollback.example.test",
		"credential": "rollback-secret", "visibility": "private", "enabled": true,
	}, source), fiber.StatusCreated).Body.Close()

	exported := decodeObject(t, mustStatus(t, perform(t, app, http.MethodPost, "/api/v1/domain/export", map[string]any{
		"privateBuildProfiles": "own", "privateAIConnections": "own", "includePublicAI": false,
	}, source), fiber.StatusOK))
	exportedWorkspace := objectField(t, exported, "workspace")
	exportedWorkspace["displayName"] = "Rollback must not persist"
	delete(exported, "domainVersion")
	plan := decodeObject(t, mustStatus(t, perform(t, app, http.MethodPost, "/api/v1/domain/import/plan", exported, target), fiber.StatusOK))
	if plan["valid"] != true {
		t.Fatalf("rollback plan is invalid before ambiguity: %#v", plan)
	}

	var beforeName string
	var beforeCommits int
	if err := database.Pool.QueryRow(t.Context(), `SELECT display_name FROM workspaces WHERE identity='rollback-target'`).Scan(&beforeName); err != nil {
		t.Fatalf("read target name: %v", err)
	}
	if err := database.Pool.QueryRow(t.Context(), `SELECT count(*) FROM workspace_commits c JOIN workspaces w ON w.id=c.workspace_id WHERE w.identity='rollback-target'`).Scan(&beforeCommits); err != nil {
		t.Fatalf("count target commits: %v", err)
	}
	duplicateID := uuid.NewString()
	if _, err := database.Pool.Exec(t.Context(), `INSERT INTO service_users(id,provider_id,subject,issuer,username,display_name) VALUES($1,'duplicate-login',$2,'urn:endge:duplicate','rollback-admin','Duplicate login')`, duplicateID, "subject-"+duplicateID); err != nil {
		t.Fatalf("insert ambiguous login: %v", err)
	}

	failedImport := perform(t, app, http.MethodPost, "/api/v1/domain/import", map[string]any{
		"planId": stringField(t, plan, "planId"), "confirmation": "rollback-target",
	}, withIfMatch(target, stringField(t, plan, "targetETag")))
	assertStatus(t, failedImport, fiber.StatusConflict)
	failedImport.Body.Close()
	var afterName string
	var afterCommits int
	if err := database.Pool.QueryRow(t.Context(), `SELECT display_name FROM workspaces WHERE identity='rollback-target'`).Scan(&afterName); err != nil {
		t.Fatalf("read rolled back target name: %v", err)
	}
	if err := database.Pool.QueryRow(t.Context(), `SELECT count(*) FROM workspace_commits c JOIN workspaces w ON w.id=c.workspace_id WHERE w.identity='rollback-target'`).Scan(&afterCommits); err != nil {
		t.Fatalf("count rolled back target commits: %v", err)
	}
	if afterName != beforeName || afterCommits != beforeCommits {
		t.Fatalf("failed adjunct import was not atomic: name %q -> %q, commits %d -> %d", beforeName, afterName, beforeCommits, afterCommits)
	}

	ambiguousPlan := decodeObject(t, mustStatus(t, perform(t, app, http.MethodPost, "/api/v1/domain/import/plan", exported, target), fiber.StatusOK))
	if ambiguousPlan["valid"] != true {
		t.Fatalf("ambiguous owners should be warnings with skipped private records: %#v", ambiguousPlan)
	}
	incoming := objectField(t, ambiguousPlan, "incoming")
	if numberField(t, incoming, "buildProfiles") != 1 || numberField(t, incoming, "skippedBuildProfiles") != 1 ||
		incoming["aiConnections"] != nil || numberField(t, incoming, "skippedAIConnections") != 1 {
		t.Fatalf("ambiguity counts are incorrect: %#v", incoming)
	}
	if warnings := stringifyJSON(t, ambiguousPlan["warnings"]); !strings.Contains(warnings, "ambiguous") {
		t.Fatalf("ambiguity warnings are missing: %s", warnings)
	}
}

func mustStatus(t *testing.T, response *http.Response, expected int) *http.Response {
	t.Helper()
	assertStatus(t, response, expected)
	return response
}

func hasIdentityWithName(items []any, identity, name string) bool {
	for _, raw := range items {
		item, _ := raw.(map[string]any)
		if item["identity"] == identity && item["displayName"] == name {
			return true
		}
	}
	return false
}
