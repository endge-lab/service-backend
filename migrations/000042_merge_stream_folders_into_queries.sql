-- +goose Up
UPDATE streams s
SET folder_id = query_root.id,
    updated_at = NOW()
FROM folders query_root
WHERE query_root.workspace_id = s.workspace_id
  AND query_root.identity = 'root-queries'
  AND query_root.entity_type = 'queries'
  AND (
    s.folder_id IS NULL
    OR EXISTS (
      SELECT 1
      FROM folders current_folder
      WHERE current_folder.workspace_id = s.workspace_id
        AND current_folder.id = s.folder_id
        AND current_folder.entity_type = 'streams'
    )
  );

DELETE FROM folders
WHERE entity_type = 'streams';

-- +goose Down
INSERT INTO folders (
    workspace_id,
    identity,
    display_name,
    entity_type,
    is_root,
    managed_by,
    created_by,
    updated_by
)
SELECT w.id,
       'root-streams',
       'Root streams',
       'streams',
       TRUE,
       'system',
       w.created_by,
       w.updated_by
FROM workspaces w
WHERE NOT EXISTS (
    SELECT 1
    FROM folders f
    WHERE f.workspace_id = w.id
      AND f.identity = 'root-streams'
);

UPDATE streams s
SET folder_id = stream_root.id,
    updated_at = NOW()
FROM folders stream_root,
     folders query_root
WHERE stream_root.workspace_id = s.workspace_id
  AND stream_root.identity = 'root-streams'
  AND stream_root.entity_type = 'streams'
  AND query_root.workspace_id = s.workspace_id
  AND query_root.identity = 'root-queries'
  AND query_root.entity_type = 'queries'
  AND s.folder_id = query_root.id;
