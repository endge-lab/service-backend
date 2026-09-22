-- +goose Up
-- Сохраняем оставшиеся документы и их Source как самостоятельные Composition.
-- folder_id сбрасывается: старая папка могла принадлежать разделу проектов.
UPDATE compositions
SET data = (data - 'kindIdentity') || jsonb_build_object('kind', 'library'),
    folder_id = NULL
WHERE lower(btrim(data ->> 'kind')) = 'project';

-- +goose Down
-- Старое владение не восстанавливается: корневой Source принадлежит Project.
SELECT 1;
