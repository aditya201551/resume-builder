-- Resume Builder — initial schema
-- Data ownership model: each resume is a self-contained document.
-- Auth: provider-agnostic — a user can hold multiple SSO identities (Google, GitHub, later LinkedIn).

CREATE EXTENSION IF NOT EXISTS pgcrypto; -- for gen_random_uuid()

-- USERS (provider-agnostic identity — no per-provider fields here)
CREATE TABLE users (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email          TEXT NOT NULL UNIQUE,
    email_verified BOOLEAN NOT NULL DEFAULT false,
    name           TEXT,
    avatar_url     TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- AUTH IDENTITIES (one row per linked SSO provider account)
-- provider_user_id is the provider's stable subject identifier (Google "sub", GitHub numeric "id", ...),
-- never the email, since email can change on the provider side while the subject stays fixed.
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

-- TEMPLATES (code-defined components; DB tracks identity + region schema only)
CREATE TABLE templates (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                  TEXT NOT NULL,
    slug                  TEXT NOT NULL UNIQUE,
    regions               JSONB NOT NULL DEFAULT '["main"]',
    default_region_map    JSONB NOT NULL DEFAULT '{}',
    preview_thumbnail_url TEXT,
    is_active             BOOLEAN NOT NULL DEFAULT true,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Seed the two starter templates
INSERT INTO templates (name, slug, regions, default_region_map) VALUES
(
    'Vertical Split',
    'vertical-split',
    '["sidebar", "main"]',
    '{
        "contact": "sidebar", "skills": "sidebar", "languages": "sidebar", "certifications": "sidebar",
        "summary": "main", "work_experience": "main", "education": "main", "projects": "main",
        "awards": "main", "publications": "main", "volunteer": "main", "custom": "main"
    }'
),
(
    'Horizontal Split',
    'horizontal-split',
    '["main"]',
    '{
        "contact": "main", "summary": "main", "work_experience": "main", "education": "main",
        "skills": "main", "projects": "main", "certifications": "main", "languages": "main",
        "awards": "main", "publications": "main", "volunteer": "main", "custom": "main"
    }'
);

-- RESUMES (parent document — owns contact info + summary directly)
CREATE TABLE resumes (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    label             TEXT NOT NULL,
    template_id       UUID REFERENCES templates(id),
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

-- WORK EXPERIENCE
CREATE TABLE work_experiences (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resume_id       UUID NOT NULL REFERENCES resumes(id) ON DELETE CASCADE,
    company         TEXT NOT NULL,
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
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_work_experiences_resume_id ON work_experiences(resume_id);

-- EDUCATION
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
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_educations_resume_id ON educations(resume_id);

-- SKILLS (groups -> items)
CREATE TABLE skill_groups (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resume_id  UUID NOT NULL REFERENCES resumes(id) ON DELETE CASCADE,
    group_name TEXT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX idx_skill_groups_resume_id ON skill_groups(resume_id);

CREATE TABLE skill_items (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    skill_group_id UUID NOT NULL REFERENCES skill_groups(id) ON DELETE CASCADE,
    name           TEXT NOT NULL,
    proficiency    TEXT,
    sort_order     INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX idx_skill_items_group_id ON skill_items(skill_group_id);

-- PROJECTS
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
    sort_order   INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX idx_projects_resume_id ON projects(resume_id);

-- CERTIFICATIONS
CREATE TABLE certifications (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resume_id      UUID NOT NULL REFERENCES resumes(id) ON DELETE CASCADE,
    name           TEXT NOT NULL,
    issuer         TEXT,
    issue_date     DATE,
    expiry_date    DATE,
    credential_url TEXT,
    sort_order     INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX idx_certifications_resume_id ON certifications(resume_id);

-- LANGUAGES
CREATE TABLE languages (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resume_id   UUID NOT NULL REFERENCES resumes(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    proficiency TEXT,
    sort_order  INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX idx_languages_resume_id ON languages(resume_id);

-- AWARDS / PUBLICATIONS / VOLUNTEER (merged via kind enum)
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
    sort_order    INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX idx_misc_entries_resume_id ON misc_entries(resume_id);

-- CUSTOM SECTIONS
CREATE TABLE custom_sections (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resume_id  UUID NOT NULL REFERENCES resumes(id) ON DELETE CASCADE,
    title      TEXT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX idx_custom_sections_resume_id ON custom_sections(resume_id);

CREATE TABLE custom_section_entries (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    custom_section_id UUID NOT NULL REFERENCES custom_sections(id) ON DELETE CASCADE,
    title             TEXT,
    description       TEXT,
    entry_date        DATE,
    sort_order        INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX idx_custom_section_entries_section_id ON custom_section_entries(custom_section_id);

-- SECTION CONFIG (per-resume visibility, order, title overrides, region)
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
    region                  TEXT,
    is_visible              BOOLEAN NOT NULL DEFAULT true,
    display_title_override  TEXT,
    sort_order              INTEGER NOT NULL DEFAULT 0,
    UNIQUE (resume_id, section_type, custom_section_id)
);
CREATE INDEX idx_resume_section_configs_resume_id ON resume_section_configs(resume_id);
