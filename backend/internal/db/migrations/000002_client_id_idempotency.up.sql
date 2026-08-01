-- Every create request from the frontend originates from a client-generated
-- tempId (see frontend/src/lib/tempId.ts) held in the local draft before any
-- network call happens. If the browser dies mid-flush — after the POST
-- reaches the server and creates the row, but before the client durably
-- records the server-assigned id back onto that draft entity — the next
-- flush has no way to know the row already exists and re-POSTs it, creating
-- a duplicate. client_id lets the create endpoints treat a retry with the
-- same tempId as a no-op that returns the original row instead of a second
-- insert.

ALTER TABLE work_experiences ADD COLUMN client_id TEXT;
CREATE UNIQUE INDEX idx_work_experiences_resume_client ON work_experiences(resume_id, client_id) WHERE client_id IS NOT NULL;

ALTER TABLE educations ADD COLUMN client_id TEXT;
CREATE UNIQUE INDEX idx_educations_resume_client ON educations(resume_id, client_id) WHERE client_id IS NOT NULL;

ALTER TABLE projects ADD COLUMN client_id TEXT;
CREATE UNIQUE INDEX idx_projects_resume_client ON projects(resume_id, client_id) WHERE client_id IS NOT NULL;

ALTER TABLE certifications ADD COLUMN client_id TEXT;
CREATE UNIQUE INDEX idx_certifications_resume_client ON certifications(resume_id, client_id) WHERE client_id IS NOT NULL;

ALTER TABLE languages ADD COLUMN client_id TEXT;
CREATE UNIQUE INDEX idx_languages_resume_client ON languages(resume_id, client_id) WHERE client_id IS NOT NULL;

ALTER TABLE misc_entries ADD COLUMN client_id TEXT;
CREATE UNIQUE INDEX idx_misc_entries_resume_client ON misc_entries(resume_id, client_id) WHERE client_id IS NOT NULL;

ALTER TABLE skill_groups ADD COLUMN client_id TEXT;
CREATE UNIQUE INDEX idx_skill_groups_resume_client ON skill_groups(resume_id, client_id) WHERE client_id IS NOT NULL;

-- skill_items has no resume_id of its own — scoped to its parent group,
-- same boundary groupBelongsToResume already checks ownership against.
ALTER TABLE skill_items ADD COLUMN client_id TEXT;
CREATE UNIQUE INDEX idx_skill_items_group_client ON skill_items(skill_group_id, client_id) WHERE client_id IS NOT NULL;

ALTER TABLE custom_sections ADD COLUMN client_id TEXT;
CREATE UNIQUE INDEX idx_custom_sections_resume_client ON custom_sections(resume_id, client_id) WHERE client_id IS NOT NULL;

ALTER TABLE custom_section_entries ADD COLUMN client_id TEXT;
CREATE UNIQUE INDEX idx_custom_section_entries_section_client ON custom_section_entries(custom_section_id, client_id) WHERE client_id IS NOT NULL;
