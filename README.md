# Resume Builder

A React/TypeScript resume editor with a Go API, PostgreSQL storage, AI assistance,
and client-side PDF preview/export.

## Development

Install Node.js 22+, Go 1.26.1+, and PostgreSQL. Run `npm ci` from the root.
The backend reads `backend/.env`: configure `DATABASE_URL`, `JWT_SECRET`,
and your Google/GitHub OAuth settings. Platform AI and BYOK are independently
optional and disabled by default; see the AI setup below.
Then run `npm run dev` when ready to start both workspaces. The API applies
database migrations at startup.

```sh
npm run test
npm run lint
npm run build
```

## Runtime feature flags

The registered keys are `platform_ai`, `platform_ai_public`, and `byok_ai`, all initially false.
Manage their database rows with existing database tools. The API exposes only
registered keys; historical records do not reactivate removed features.

The application only reads flag values. The API caches decisions for ten seconds,
and the frontend polls every ten seconds and refreshes on window focus. Unknown
flags and failed refreshes default off. Direct database edits are not automatically
written to the historical audit table. No feature-management CLI or browser write
endpoint is included.

The pre-release schema is consolidated into one baseline up/down pair,
`000001_init_schema`. It creates the 20 current application tables and seeds
only the three disabled flags and their initial audit entries. Retired Jobs
tables, the database template catalog, and test records are not included.
The migration engine maintains `schema_migrations` separately.

This baseline replaces the old local-only migration history. A database from
that history must be backed up and reset before applying it; changing its
recorded version alone does not update its schema. Once using this baseline,
add new numbered migrations and preserve applied files and data. Rolling the
baseline down removes all application tables and their data.

Optional feature-store integration tests use a disposable schema:

```sh
cd backend
FEATURE_FLAGS_TEST_DATABASE_URL="<test-database-url>" \
  go test ./internal/features/postgres -run TestFeatureFlagsPostgresIntegration -v
```

## AI connections

When BYOK is enabled, the account menu's **AI connections** screen manages one saved connection per
provider: OpenAI, Anthropic, Vercel AI Gateway, and OpenRouter. Save a key, load
models, save a compatible model, test it, then select **Use this connection**.
Connection tests send synthetic tool-call requests and may incur provider charges.
The selected personal model powers both chat and native PDF import; chat-only
models cannot import PDFs. Keys are never included in resume drafts or returned
to the browser after saving.

The connections page and connection controls are hidden when `byok_ai` is off,
including direct navigation to `/settings/ai`. Platform-only users use the
assistant directly, without seeing model details. If a saved BYOK preference
becomes unavailable, an explicit **Use Platform AI** action is offered when
Platform AI is available; the app does not silently switch billing sources.

To enable encrypted personal key storage, configure these server-only variables:

```dotenv
AI_ENCRYPTION_KEYS={"v1":"<base64-encoded-32-byte-random-key>"}
AI_ENCRYPTION_ACTIVE_KEY=v1
```

Generate the key using a cryptographically secure random generator. Store it
outside version control and back it up separately from the database. To rotate,
add another version to `AI_ENCRYPTION_KEYS` and select it as the active version.
Retain old versions until all connections encrypted under them have been replaced.
Missing or invalid encryption configuration leaves BYOK unavailable without
preventing manual editing. The feature flag must also be enabled.

Platform AI additionally requires `AI_PROVIDER`, `AI_API_KEY`, and `AI_MODEL`.
Provider values are `openai`, `anthropic`, `vercel`, or `openrouter`; Vercel takes
an AI Gateway key. `AGENT_MODEL` optionally overrides the platform chat model,
while imports use `AI_MODEL`. `AGENT_MAX_TOKENS` defaults to 16384 and must be
between 512 and 32768. Missing platform configuration leaves Platform AI
unavailable without preventing BYOK or manual editing.

Platform access is invite-only unless `platform_ai_public` is enabled. Public
access means every signed-in user, and still requires `platform_ai` and valid
platform credentials. Anonymous requests are never permitted. Manage invitations
using internal user UUIDs:

```sql
INSERT INTO platform_ai_allowlist(user_id) VALUES ('<user-uuid>')
ON CONFLICT DO NOTHING;
```

Enable either mode and record its audit entry in the same database transaction:

```sql
BEGIN;
UPDATE feature_flags SET enabled = TRUE, updated_by = '<operator>', updated_at = now()
WHERE key = 'byok_ai'; -- or platform_ai
INSERT INTO feature_flag_audit(feature_key, enabled, actor)
VALUES ('byok_ai', TRUE, '<operator>');
COMMIT;
```

For a public release, enable both platform flags and record the changes together:

```sql
BEGIN;
UPDATE feature_flags SET enabled = TRUE, updated_by = '<operator>', updated_at = now()
WHERE key IN ('platform_ai', 'platform_ai_public');
INSERT INTO feature_flag_audit(feature_key, enabled, actor)
SELECT key, enabled, '<operator>' FROM feature_flags
WHERE key IN ('platform_ai', 'platform_ai_public');
COMMIT;
```

Disabling only `platform_ai_public` restores invite-only access without changing
the allowlist or `byok_ai`. Disabling `platform_ai` blocks Platform AI for everyone.
The baseline initializes the public flag disabled. On a fresh database, startup
applies the baseline before any flags can be enabled.

Without a saved preference, eligible users default to Platform AI when enabled;
other users configure BYOK when enabled. Public access never replaces an explicit
BYOK selection, including one whose key was deleted. Explicit preferences persist across flag
changes. Disabling a mode blocks new calls, connection tests and model discovery;
running operations finish, and encrypted keys remain available for deletion.
There is no automatic fallback between billing sources. Turn modes off through
their flags to preserve credentials, preferences and audit records; the baseline
down migration is a destructive full-schema rollback.

Direct OpenAI currently uses Chat Completions with a filtered set of compatible
tool models, `store:false`, and model-specific token/reasoning options. Responses-only
tool models are excluded. Gateway PDF capability is advertised rather than a
guarantee of extraction quality; OpenRouter uses native parsing without OCR fallback.
Provider endpoints are fixed. Cookie-authenticated AI mutations require the
configured `FRONTEND_URL` origin; bearer-token clients without a session cookie
are supported. AI work has per-account concurrency and rate limits and bounded
execution time.
