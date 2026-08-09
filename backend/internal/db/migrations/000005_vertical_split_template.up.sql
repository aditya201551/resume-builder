
INSERT INTO templates (renderer_key, name, supported_modes, supported_groups, default_design, sort_order)
VALUES (
    'vertical-split',
    'Vertical Split',
    '["two"]',
    '["layout","fontSize","spacing","entries","headings","font","colors","header","linkStyle","footer"]',
    '{
        "templateId": "",
        "layout": {"mode": "two", "columnWidths": {"left": 34, "right": 66}},
        "sectionOrder": {
            "one": {"sections": []},
            "two": {
                "left": [
                    {"sectionType": "skills", "customSectionId": null, "isVisible": true, "titleOverride": null},
                    {"sectionType": "languages", "customSectionId": null, "isVisible": true, "titleOverride": null},
                    {"sectionType": "certifications", "customSectionId": null, "isVisible": true, "titleOverride": null}
                ],
                "right": [
                    {"sectionType": "summary", "customSectionId": null, "isVisible": true, "titleOverride": null},
                    {"sectionType": "work_experience", "customSectionId": null, "isVisible": true, "titleOverride": null},
                    {"sectionType": "education", "customSectionId": null, "isVisible": true, "titleOverride": null},
                    {"sectionType": "projects", "customSectionId": null, "isVisible": true, "titleOverride": null},
                    {"sectionType": "awards", "customSectionId": null, "isVisible": true, "titleOverride": null},
                    {"sectionType": "publications", "customSectionId": null, "isVisible": true, "titleOverride": null},
                    {"sectionType": "volunteer", "customSectionId": null, "isVisible": true, "titleOverride": null}
                ]
            },
            "mix": {"bands": []}
        },
        "page": {"format": "Letter", "marginTop": 40, "marginBottom": 40, "marginLeft": 40, "marginRight": 40},
        "typography": {
            "fontFamily": "Inter", "nameFontFamily": "inherit", "baseFontSizePt": 12.5, "lineHeight": 1.5,
            "nameFontSizePt": 24, "headlineFontSizePt": 13.5, "sectionHeadingFontSizePt": 12, "entryHeaderFontSizePt": 12.5
        },
        "colors": {
            "text": "#1f2328", "accent": "#3457d5", "background": "#ffffff",
            "applyAccent": {"name": false, "headings": true, "dates": false, "icons": true}
        },
        "heading": {"style": "simple", "capitalization": "uppercase"},
        "header": {"photo": {"show": false, "size": "m"}, "alignText": "left", "jobTitlePosition": "below"},
        "entryLayout": {"dateDisplayMode": "right", "subtitleStyle": "italic"},
        "sectionDisplay": {"skills": "bullets", "languages": "text", "certifications": "text"},
        "spacing": {"sectionGap": 16, "entryGap": 10, "bulletGap": 4},
        "linkStyle": {"showIcon": true, "underline": false, "useAccentColor": true},
        "footer": {"showPageNumbers": false, "showEmail": false, "showName": false},
        "dateFormat": "Mon YYYY"
    }'::jsonb,
    1
);
