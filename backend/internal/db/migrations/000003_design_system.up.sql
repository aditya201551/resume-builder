-- Template/design (customization) system — deliberately decoupled from the
-- content tables above. A resume's visual presentation (template choice,
-- layout mode, colors, typography, section order/visibility/titles) now
-- lives here instead of in resume_section_configs, so content editing and
-- formatting are two independent systems that can be built, tested, and
-- (later) exposed to the AI agent separately.
--
-- This migration only ADDS tables. resume_section_configs is left in place
-- and still authoritative for the current frontend until it migrates onto
-- this system — do not drop it here.

CREATE TABLE templates (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    renderer_key    TEXT NOT NULL UNIQUE,
    name            TEXT NOT NULL,
    thumbnail_url   TEXT,
    tags            JSONB NOT NULL DEFAULT '[]',
    supported_modes JSONB NOT NULL DEFAULT '["one"]',
    default_design  JSONB NOT NULL,
    is_premium      BOOLEAN NOT NULL DEFAULT false,
    published       BOOLEAN NOT NULL DEFAULT true,
    sort_order      INTEGER NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- One design per resume. A missing row means "use the default template's
-- preset" (see service.resolveResumeDesign) rather than every resume
-- needing a row from creation.
CREATE TABLE resume_designs (
    resume_id  UUID PRIMARY KEY REFERENCES resumes(id) ON DELETE CASCADE,
    design     JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Seed the single v1 template. Its default_design reproduces
-- LivePreview.module.css's current hardcoded look field-for-field, so
-- existing resumes render identically the moment this ships — nothing
-- visually changes until a user actually opens the (future) design panel.
INSERT INTO templates (renderer_key, name, supported_modes, default_design, sort_order)
VALUES (
    'classic',
    'Classic',
    '["one", "two", "mix"]',
    '{
        "templateId": "",
        "layout": {"mode": "one", "columnWidths": {"left": 50, "right": 50}},
        "sectionOrder": {
            "one": {"sections": [
                {"sectionType": "summary", "customSectionId": null, "isVisible": true, "titleOverride": null},
                {"sectionType": "work_experience", "customSectionId": null, "isVisible": true, "titleOverride": null},
                {"sectionType": "education", "customSectionId": null, "isVisible": true, "titleOverride": null},
                {"sectionType": "skills", "customSectionId": null, "isVisible": true, "titleOverride": null},
                {"sectionType": "projects", "customSectionId": null, "isVisible": true, "titleOverride": null},
                {"sectionType": "certifications", "customSectionId": null, "isVisible": true, "titleOverride": null},
                {"sectionType": "languages", "customSectionId": null, "isVisible": true, "titleOverride": null},
                {"sectionType": "awards", "customSectionId": null, "isVisible": true, "titleOverride": null},
                {"sectionType": "publications", "customSectionId": null, "isVisible": true, "titleOverride": null},
                {"sectionType": "volunteer", "customSectionId": null, "isVisible": true, "titleOverride": null}
            ]},
            "two": {"left": [], "right": []},
            "mix": {"bands": []}
        },
        "page": {"format": "Letter", "marginTop": 48, "marginBottom": 48, "marginLeft": 56, "marginRight": 56},
        "typography": {"fontFamily": "Georgia", "baseFontSizePt": 13, "lineHeight": 1.55, "nameFontSizePt": 22, "sectionHeadingFontSizePt": 11},
        "colors": {"text": "#14181d", "accent": "#c9603e", "background": "#ffffff", "applyAccent": {"name": false, "headings": false, "dates": false, "icons": false}},
        "heading": {"style": "line", "capitalization": "uppercase"},
        "header": {"photo": {"show": false, "size": "m"}, "alignText": "left", "jobTitlePosition": "below"},
        "entryLayout": {"dateDisplayMode": "right", "subtitleStyle": "italic"},
        "sectionDisplay": {"skills": "text", "languages": "text", "certifications": "text"},
        "spacing": {"sectionGap": 18, "entryGap": 10, "bulletGap": 4},
        "dateFormat": "Mon YYYY"
    }'::jsonb,
    0
);
