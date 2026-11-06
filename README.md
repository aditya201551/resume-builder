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

The PostgreSQL feature-flag system has no features currently registered. Register
keys in `backend/internal/features/catalog.go` and manage their database rows with
your existing database tools. The API exposes only registered keys; historical
records do not reactivate removed features.

The application only reads flag values. The API caches decisions for ten seconds,
and the frontend polls every ten seconds and refreshes on window focus. Unknown
flags and failed refreshes default off. Direct database edits are not automatically
written to the historical audit table. No feature-management CLI or browser write
endpoint is included.

Applied migrations, historical Jobs data, and flag audit records are preserved.
The Jobs application module and its flag registration have been removed.

Optional feature-store integration tests use a disposable schema:

```sh
cd backend
FEATURE_FLAGS_TEST_DATABASE_URL="<test-database-url>" \
  go test ./internal/features/postgres -run TestFeatureFlagsPostgresIntegration -v
```
