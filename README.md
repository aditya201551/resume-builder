# Resume Builder

A React/TypeScript resume editor with a Go API, PostgreSQL storage, AI assistance,
client-side PDF preview/export, and an independently controlled Jobs module.

## Development

Install Node.js 22+, Go 1.26.1+, and PostgreSQL. Run `npm ci` from the root.
The backend reads `backend/.env`: configure `DATABASE_URL`, `JWT_SECRET`,
`AI_PROVIDER`, `AI_API_KEY`, `AI_MODEL`, and your Google/GitHub OAuth settings.
Then run `npm run dev` when ready to start both workspaces. The API applies
database migrations at startup.

```sh
npm run test
npm run lint
npm run build
```

Jobs starts disabled. Manage its runtime flag with trusted database credentials:

```sh
npm run features --workspace=backend -- list
npm run features --workspace=backend -- set jobs.enabled false --actor your-name
npm run features --workspace=backend -- set jobs.enabled true --actor your-name
```

The API applies migration `000010` at startup. Flag changes are audited atomically
and take roughly twenty seconds to reach an active browser through backend caching
and frontend polling. Unknown flags and failed registry refreshes default off.
Disabling Jobs blocks new requests and hides its UI; it preserves saved jobs and
does not cancel work already admitted. Each environment should use its own database.
The actor is an operator-supplied audit label; database credentials control write
access. The production container includes a `features` binary for service exec.

Jobs owns its implementation under `backend/internal/modules/jobs/internal/` and
`frontend/src/features/jobs/`. Application code uses the module's public entry
points. Shared flag evaluation and PostgreSQL storage live in
`backend/internal/features/`; frontend guards live in `frontend/src/platform/features/`.

Optional integration tests use disposable schemas in a supplied test database:

```sh
cd backend
FEATURE_FLAGS_TEST_DATABASE_URL="<test-database-url>" \
  go test ./internal/features/postgres -run TestFeatureFlagsPostgresIntegration -v
JOB_SEARCH_DB_TEST=1 JOB_SEARCH_TEST_DATABASE_URL="<test-database-url>" \
  go test ./internal/modules/jobs/internal/repository -run TestJobDiscoveryPostgresIntegration -v
```
