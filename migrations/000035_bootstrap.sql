-- +goose Up
INSERT INTO service_users (id, provider_id, subject, issuer, username, display_name, active, is_system)
VALUES ('00000000-0000-0000-0000-000000000001', 'system', 'system', 'urn:endge:system', 'system', 'Endge System', TRUE,
        TRUE);

-- +goose Down
DELETE
FROM service_users u
WHERE u.id = '00000000-0000-0000-0000-000000000001'
  AND NOT EXISTS (SELECT 1 FROM integrations i WHERE i.created_by = u.id OR i.updated_by = u.id)
  AND NOT EXISTS (SELECT 1 FROM document_revisions r WHERE r.created_by = u.id)
  AND NOT EXISTS (SELECT 1 FROM mutation_batches b WHERE b.actor_user_id = u.id);
