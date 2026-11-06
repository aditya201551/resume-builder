-- Consolidated pre-release baseline. Apply only to a fresh database.
-- Test records and retired Jobs/template-catalog tables are intentionally excluded.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE users (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email          TEXT NOT NULL UNIQUE,
    email_verified BOOLEAN NOT NULL DEFAULT false,
    name           TEXT,
    avatar_url     TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE auth_identities (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider         TEXT NOT NULL CHECK (provider IN ('google', 'github', 'linkedin')),
    provider_user_id TEXT NOT NULL,
    provider_email   TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (provider, provider_user_id)
);
CREATE INDEX idx_auth_identities_user_id ON auth_identities(user_id);

CREATE TABLE resumes (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    label             TEXT NOT NULL,
    full_name         TEXT NOT NULL,
    headline          TEXT,
    email             TEXT,
    phone             TEXT,
    location          TEXT,
    photo_url         TEXT,
    summary           TEXT,
    links             JSONB NOT NULL DEFAULT '[]',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_exported_at  TIMESTAMPTZ
);
CREATE INDEX idx_resumes_user_id ON resumes(user_id);

CREATE TABLE work_experiences (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resume_id       UUID NOT NULL REFERENCES resumes(id) ON DELETE CASCADE,
    company         TEXT NOT NULL,
    company_url     TEXT,
    title           TEXT NOT NULL,
    location        TEXT,
    employment_type TEXT,
    start_date      DATE,
    end_date        DATE,
    is_current      BOOLEAN NOT NULL DEFAULT false,
    content         TEXT NOT NULL DEFAULT '',
    technologies    JSONB NOT NULL DEFAULT '[]',
    sort_order      INTEGER NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    client_id TEXT
);
CREATE INDEX idx_work_experiences_resume_id ON work_experiences(resume_id);

CREATE TABLE educations (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resume_id          UUID NOT NULL REFERENCES resumes(id) ON DELETE CASCADE,
    institution        TEXT NOT NULL,
    degree             TEXT,
    field_of_study     TEXT,
    location           TEXT,
    start_date         DATE,
    end_date           DATE,
    gpa                TEXT,
    honors_description TEXT,
    sort_order         INTEGER NOT NULL DEFAULT 0,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    client_id TEXT
);
CREATE INDEX idx_educations_resume_id ON educations(resume_id);

CREATE TABLE skill_groups (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resume_id  UUID NOT NULL REFERENCES resumes(id) ON DELETE CASCADE,
    group_name TEXT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    client_id TEXT
);
CREATE INDEX idx_skill_groups_resume_id ON skill_groups(resume_id);

CREATE TABLE skill_items (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    skill_group_id UUID NOT NULL REFERENCES skill_groups(id) ON DELETE CASCADE,
    name           TEXT NOT NULL,
    proficiency    TEXT,
    sort_order     INTEGER NOT NULL DEFAULT 0,
    client_id TEXT
);
CREATE INDEX idx_skill_items_group_id ON skill_items(skill_group_id);

CREATE TABLE projects (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resume_id    UUID NOT NULL REFERENCES resumes(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    content      TEXT NOT NULL DEFAULT '',
    role         TEXT,
    technologies JSONB NOT NULL DEFAULT '[]',
    url          TEXT,
    start_date   DATE,
    end_date     DATE,
    sort_order   INTEGER NOT NULL DEFAULT 0,
    client_id TEXT
);
CREATE INDEX idx_projects_resume_id ON projects(resume_id);

CREATE TABLE certifications (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resume_id      UUID NOT NULL REFERENCES resumes(id) ON DELETE CASCADE,
    name           TEXT NOT NULL,
    issuer         TEXT,
    issue_date     DATE,
    expiry_date    DATE,
    credential_url TEXT,
    sort_order     INTEGER NOT NULL DEFAULT 0,
    client_id TEXT
);
CREATE INDEX idx_certifications_resume_id ON certifications(resume_id);

CREATE TABLE languages (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resume_id   UUID NOT NULL REFERENCES resumes(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    proficiency TEXT,
    sort_order  INTEGER NOT NULL DEFAULT 0,
    client_id TEXT
);
CREATE INDEX idx_languages_resume_id ON languages(resume_id);

CREATE TYPE misc_entry_kind AS ENUM ('award', 'publication', 'volunteer');

CREATE TABLE misc_entries (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resume_id     UUID NOT NULL REFERENCES resumes(id) ON DELETE CASCADE,
    kind          misc_entry_kind NOT NULL,
    title         TEXT NOT NULL,
    issuer_or_org TEXT,
    url           TEXT,
    entry_date    DATE,
    description   TEXT,
    sort_order    INTEGER NOT NULL DEFAULT 0,
    client_id TEXT
);
CREATE INDEX idx_misc_entries_resume_id ON misc_entries(resume_id);

CREATE TABLE custom_sections (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resume_id  UUID NOT NULL REFERENCES resumes(id) ON DELETE CASCADE,
    title      TEXT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    client_id TEXT
);
CREATE INDEX idx_custom_sections_resume_id ON custom_sections(resume_id);

CREATE TABLE custom_section_entries (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    custom_section_id UUID NOT NULL REFERENCES custom_sections(id) ON DELETE CASCADE,
    title             TEXT,
    description       TEXT,
    entry_date        DATE,
    sort_order        INTEGER NOT NULL DEFAULT 0,
    client_id TEXT
);
CREATE INDEX idx_custom_section_entries_section_id ON custom_section_entries(custom_section_id);

CREATE TYPE section_type AS ENUM (
    'contact', 'summary', 'work_experience', 'education', 'skills',
    'projects', 'certifications', 'languages', 'awards', 'publications',
    'volunteer', 'custom'
);

CREATE TABLE resume_section_configs (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resume_id               UUID NOT NULL REFERENCES resumes(id) ON DELETE CASCADE,
    section_type            section_type NOT NULL,
    custom_section_id       UUID REFERENCES custom_sections(id) ON DELETE CASCADE,
    is_visible              BOOLEAN NOT NULL DEFAULT true,
    display_title_override  TEXT,
    sort_order              INTEGER NOT NULL DEFAULT 0,
    UNIQUE (resume_id, section_type, custom_section_id)
);
CREATE INDEX idx_resume_section_configs_resume_id ON resume_section_configs(resume_id);

CREATE UNIQUE INDEX idx_resume_section_configs_main_unique
    ON resume_section_configs (resume_id, section_type)
    WHERE custom_section_id IS NULL;

CREATE UNIQUE INDEX idx_work_experiences_resume_client ON work_experiences(resume_id, client_id) WHERE client_id IS NOT NULL;
CREATE UNIQUE INDEX idx_educations_resume_client ON educations(resume_id, client_id) WHERE client_id IS NOT NULL;
CREATE UNIQUE INDEX idx_projects_resume_client ON projects(resume_id, client_id) WHERE client_id IS NOT NULL;
CREATE UNIQUE INDEX idx_certifications_resume_client ON certifications(resume_id, client_id) WHERE client_id IS NOT NULL;
CREATE UNIQUE INDEX idx_languages_resume_client ON languages(resume_id, client_id) WHERE client_id IS NOT NULL;
CREATE UNIQUE INDEX idx_misc_entries_resume_client ON misc_entries(resume_id, client_id) WHERE client_id IS NOT NULL;
CREATE UNIQUE INDEX idx_skill_groups_resume_client ON skill_groups(resume_id, client_id) WHERE client_id IS NOT NULL;
CREATE UNIQUE INDEX idx_skill_items_group_client ON skill_items(skill_group_id, client_id) WHERE client_id IS NOT NULL;
CREATE UNIQUE INDEX idx_custom_sections_resume_client ON custom_sections(resume_id, client_id) WHERE client_id IS NOT NULL;
CREATE UNIQUE INDEX idx_custom_section_entries_section_client ON custom_section_entries(custom_section_id, client_id) WHERE client_id IS NOT NULL;

CREATE TABLE resume_designs (
    resume_id  UUID PRIMARY KEY REFERENCES resumes(id) ON DELETE CASCADE,
    design     JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE feature_flags (
    key TEXT PRIMARY KEY,
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    updated_by TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE feature_flag_audit (
    id BIGSERIAL PRIMARY KEY,
    feature_key TEXT NOT NULL REFERENCES feature_flags(key),
    enabled BOOLEAN NOT NULL,
    actor TEXT NOT NULL,
    changed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE ai_connections (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
 user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 provider TEXT NOT NULL CHECK (provider IN ('openai','anthropic','vercel','openrouter')),
 model TEXT NOT NULL DEFAULT '',
 encrypted_key BYTEA NOT NULL,
 encryption_version TEXT NOT NULL,
 key_suffix TEXT NOT NULL,
 credential_version BIGINT NOT NULL DEFAULT 1,
 pdf_supported BOOLEAN NOT NULL DEFAULT FALSE,
 validated_at TIMESTAMPTZ,
 updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 UNIQUE(user_id, provider), UNIQUE(user_id, id)
);
CREATE TABLE ai_preferences (
 user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
 source TEXT NOT NULL CHECK (source IN ('platform','byok')),
 active_connection_id UUID,
 updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 FOREIGN KEY (user_id, active_connection_id) REFERENCES ai_connections(user_id,id),
 CHECK (source = 'byok' OR active_connection_id IS NULL)
);
CREATE TABLE platform_ai_allowlist (
 user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO feature_flags(key, enabled, updated_by)
VALUES
    ('platform_ai', FALSE, 'migration:baseline'),
    ('platform_ai_public', FALSE, 'migration:baseline'),
    ('byok_ai', FALSE, 'migration:baseline');

INSERT INTO feature_flag_audit(feature_key, enabled, actor)
SELECT key, enabled, 'migration:baseline' FROM feature_flags;
