UPDATE resume_designs
SET design = (design - 'linkStyle' - 'footer') || jsonb_build_object(
    'typography', (design->'typography') - 'nameFontFamily' - 'headlineFontSizePt' - 'entryHeaderFontSizePt',
    'heading', (design->'heading') || '{"style":"line"}'::jsonb
);

UPDATE templates
SET default_design = (default_design - 'linkStyle' - 'footer') || jsonb_build_object(
        'typography', (default_design->'typography') - 'nameFontFamily' - 'headlineFontSizePt' - 'entryHeaderFontSizePt',
        'heading', (default_design->'heading') || '{"style":"line"}'::jsonb
    ),
    supported_modes = '["one","two","mix"]'
WHERE renderer_key = 'classic';

ALTER TABLE templates DROP COLUMN supported_groups;
