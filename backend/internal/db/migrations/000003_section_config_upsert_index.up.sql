-- Postgres treats NULL as distinct in a plain UNIQUE constraint, so the
-- existing UNIQUE (resume_id, section_type, custom_section_id) never
-- actually prevents duplicate rows when custom_section_id IS NULL (every
-- non-custom section type). That in turn means ON CONFLICT can't target it
-- for an upsert on non-custom section types. This partial unique index
-- covers exactly that case.
CREATE UNIQUE INDEX idx_resume_section_configs_main_unique
    ON resume_section_configs (resume_id, section_type)
    WHERE custom_section_id IS NULL;
