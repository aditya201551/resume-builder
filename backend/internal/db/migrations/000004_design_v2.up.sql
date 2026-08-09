
ALTER TABLE templates ADD COLUMN supported_groups JSONB NOT NULL DEFAULT '[]';

UPDATE templates
SET supported_modes = '["one"]',
    supported_groups = '["layout","fontSize","spacing","entries","headings","font","colors","header","linkStyle","footer"]'
WHERE renderer_key = 'classic';

UPDATE templates
SET default_design = default_design || jsonb_build_object(
    'typography', (default_design->'typography') || '{"nameFontFamily":"inherit","headlineFontSizePt":13,"entryHeaderFontSizePt":13}'::jsonb,
    'heading', (default_design->'heading') || '{"style":"simple"}'::jsonb,
    'linkStyle', '{"showIcon":true,"underline":false,"useAccentColor":false}'::jsonb,
    'footer', '{"showPageNumbers":false,"showEmail":false,"showName":false}'::jsonb
)
WHERE renderer_key = 'classic';

UPDATE resume_designs
SET design = design || jsonb_build_object(
    'typography', (design->'typography') || '{"nameFontFamily":"inherit","headlineFontSizePt":13,"entryHeaderFontSizePt":13}'::jsonb,
    'heading', (design->'heading') || '{"style":"simple"}'::jsonb,
    'linkStyle', '{"showIcon":true,"underline":false,"useAccentColor":false}'::jsonb,
    'footer', '{"showPageNumbers":false,"showEmail":false,"showName":false}'::jsonb
)
WHERE NOT (design ? 'linkStyle');
