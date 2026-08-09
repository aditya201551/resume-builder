
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

CREATE TABLE resume_designs (
    resume_id  UUID PRIMARY KEY REFERENCES resumes(id) ON DELETE CASCADE,
    design     JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

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
