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

INSERT INTO feature_flags(key, enabled, updated_by)
VALUES ('jobs.enabled', FALSE, 'migration');

INSERT INTO feature_flag_audit(feature_key, enabled, actor)
VALUES ('jobs.enabled', FALSE, 'migration');
