-- +goose Up
UPDATE workspaces
SET configuration = configuration - 'sse',
    updated_at = NOW(),
    revision = revision + 1
WHERE configuration ? 'sse';

UPDATE projects
SET data = data
        #- '{configuration,sse}'
        #- '{configuration,patch,sse}'
        #- '{configuration,value,sse}',
    updated_at = NOW(),
    revision = revision + 1
WHERE (data #> '{configuration}') ? 'sse'
   OR (data #> '{configuration,patch}') ? 'sse'
   OR (data #> '{configuration,value}') ? 'sse';

UPDATE tenants
SET data = data
        #- '{configuration,sse}'
        #- '{configuration,patch,sse}'
        #- '{configuration,value,sse}',
    updated_at = NOW(),
    revision = revision + 1
WHERE (data #> '{configuration}') ? 'sse'
   OR (data #> '{configuration,patch}') ? 'sse'
   OR (data #> '{configuration,value}') ? 'sse';

UPDATE environments
SET data = data
        #- '{configuration,sse}'
        #- '{configuration,patch,sse}'
        #- '{configuration,value,sse}',
    updated_at = NOW(),
    revision = revision + 1
WHERE (data #> '{configuration}') ? 'sse'
   OR (data #> '{configuration,patch}') ? 'sse'
   OR (data #> '{configuration,value}') ? 'sse';

-- +goose Down
-- Удалённые endpoint-значения не восстанавливаются: они могли содержать
-- environment-specific URL и больше не входят в доменную модель.
SELECT 1;
