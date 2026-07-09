-- +goose Up
CREATE TABLE IF NOT EXISTS folders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NULL,
    entity_type TEXT NOT NULL,
    identity TEXT NOT NULL,
    display_name TEXT NOT NULL,
    description TEXT NULL,
    parent_id UUID NULL REFERENCES folders(id) ON DELETE SET NULL,
    is_root BOOLEAN NOT NULL DEFAULT FALSE,
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMPTZ NULL,
    meta JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT folders_entity_type_check CHECK (entity_type IN ('components', 'converters', 'queries', 'data-views')),
    CONSTRAINT folders_identity_not_empty_check CHECK (btrim(identity) <> ''),
    CONSTRAINT folders_display_name_not_empty_check CHECK (btrim(display_name) <> ''),
    CONSTRAINT folders_not_self_parent_check CHECK (parent_id IS NULL OR parent_id <> id),
    CONSTRAINT folders_root_has_no_parent_check CHECK (NOT is_root OR parent_id IS NULL)
);

CREATE UNIQUE INDEX IF NOT EXISTS folders_project_entity_identity_unique
    ON folders (project_id, entity_type, identity)
    WHERE project_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS folders_global_entity_identity_unique
    ON folders (entity_type, identity)
    WHERE project_id IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS folders_project_entity_root_unique
    ON folders (project_id, entity_type)
    WHERE project_id IS NOT NULL AND is_root;

CREATE UNIQUE INDEX IF NOT EXISTS folders_global_entity_root_unique
    ON folders (entity_type)
    WHERE project_id IS NULL AND is_root;

CREATE OR REPLACE FUNCTION validate_folder_tree()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'UPDATE' AND OLD.is_root THEN
        IF NEW.parent_id IS NOT NULL
            OR NEW.project_id IS DISTINCT FROM OLD.project_id
            OR NEW.entity_type <> OLD.entity_type THEN
            RAISE EXCEPTION 'root folder cannot be moved'
                USING ERRCODE = '23514';
        END IF;
    END IF;

    IF NEW.parent_id IS NULL THEN
        RETURN NEW;
    END IF;

    IF NEW.parent_id = NEW.id THEN
        RAISE EXCEPTION 'folder cannot be parent of itself'
            USING ERRCODE = '23514';
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM folders parent
        WHERE parent.id = NEW.parent_id
            AND parent.project_id IS NOT DISTINCT FROM NEW.project_id
            AND parent.entity_type = NEW.entity_type
    ) THEN
        RAISE EXCEPTION 'folder parent must belong to the same project and entity type'
            USING ERRCODE = '23514';
    END IF;

    IF EXISTS (
        WITH RECURSIVE ancestors AS (
            SELECT id, parent_id
            FROM folders
            WHERE id = NEW.parent_id

            UNION ALL

            SELECT parent.id, parent.parent_id
            FROM folders parent
            INNER JOIN ancestors ON parent.id = ancestors.parent_id
        )
        SELECT 1
        FROM ancestors
        WHERE id = NEW.id
    ) THEN
        RAISE EXCEPTION 'folder cycle is not allowed'
            USING ERRCODE = '23514';
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER folders_validate_tree_before_insert_update
    BEFORE INSERT OR UPDATE ON folders
    FOR EACH ROW
    EXECUTE FUNCTION validate_folder_tree();

-- +goose Down
DROP TRIGGER IF EXISTS folders_validate_tree_before_insert_update ON folders;
DROP FUNCTION IF EXISTS validate_folder_tree();
DROP TABLE IF EXISTS folders;
