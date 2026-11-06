# Resume Builder

A React/TypeScript resume editor with a Go API, PostgreSQL storage, AI assistance,
and client-side PDF preview/export.

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

## Runtime feature flags

The shared PostgreSQL feature-flag system is retained, with no features currently
registered. Register keys in `backend/internal/features/catalog.go` before using
flags to control availability. The API and CLI list expose only registered keys;
historical database records do not reactivate removed features.

```sh
npm run features --workspace=backend -- list
npm run features --workspace=backend -- set <registered-key> true --actor your-name
```

Flag updates are audited atomically. The API caches decisions for ten seconds,
and the frontend polls every ten seconds and refreshes on window focus. Unknown
flags and failed registry refreshes default off. Database credentials control
write access; no browser-accessible flag-write endpoint exists. The production
container includes a `features` binary for service exec.

Applied migrations, historical Jobs data, and flag audit records are preserved.
The Jobs application module and its flag registration have been removed.

Optional feature-store integration tests use a disposable schema:

```sh
cd backend
FEATURE_FLAGS_TEST_DATABASE_URL="<test-database-url>" \
  go test ./internal/features/postgres -run TestFeatureFlagsPostgresIntegration -v
```
