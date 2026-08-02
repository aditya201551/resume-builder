-- Design v2: template capability gating + a handful of newly-wired design
-- fields (separate name font, two more font-size sliders, link styling,
-- footer). See internal/design/schema.go's Schema() for the full field
-- catalog these support.

-- Which design-panel nav categories (Schema() group keys) a template
-- actually exposes. Cross-referenced by the frontend alongside the existing
-- supported_modes — a template can support a field's *shape* (e.g. "layout"
-- generically) while only allowing a subset of its values (supported_modes
-- narrows layout.mode's options specifically).
ALTER TABLE templates ADD COLUMN supported_groups JSONB NOT NULL DEFAULT '[]';

-- "classic" only ever renders single-column — supported_modes was seeded as
-- ["one","two","mix"] in 000003 before any of this was wired up, which was
-- never actually true. Correcting it now that the design panel reads it.
UPDATE templates
SET supported_modes = '["one"]',
    supported_groups = '["layout","fontSize","spacing","entries","headings","font","colors","header","linkStyle","footer"]'
WHERE renderer_key = 'classic';

-- Merge the new keys into the seeded default_design, and correct
-- heading.style from "line" to "simple". heading.style was validated but
-- never rendered before this migration's accompanying code change wires it
-- up — every existing resume already visually looks like "simple" (no
-- decoration), so this is a default-value fix, not a behavior change.
UPDATE templates
SET default_design = default_design || jsonb_build_object(
    'typography', (default_design->'typography') || '{"nameFontFamily":"inherit","headlineFontSizePt":13,"entryHeaderFontSizePt":13}'::jsonb,
    'heading', (default_design->'heading') || '{"style":"simple"}'::jsonb,
    'linkStyle', '{"showIcon":true,"underline":false,"useAccentColor":false}'::jsonb,
    'footer', '{"showPageNumbers":false,"showEmail":false,"showName":false}'::jsonb
)
WHERE renderer_key = 'classic';

-- Backfill any resume that already saved a design row before this migration
-- ran, the same way. Guarded so re-running this against a row that somehow
-- already has linkStyle (post-migration) is a no-op.
UPDATE resume_designs
SET design = design || jsonb_build_object(
    'typography', (design->'typography') || '{"nameFontFamily":"inherit","headlineFontSizePt":13,"entryHeaderFontSizePt":13}'::jsonb,
    'heading', (design->'heading') || '{"style":"simple"}'::jsonb,
    'linkStyle', '{"showIcon":true,"underline":false,"useAccentColor":false}'::jsonb,
    'footer', '{"showPageNumbers":false,"showEmail":false,"showName":false}'::jsonb
)
WHERE NOT (design ? 'linkStyle');
