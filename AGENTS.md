# AGENTS.md

This file provides guidance to Codex (Codex.ai/code) when working with code in this repository.

## What this is

A resume builder: Go REST API backend + React/TypeScript frontend, monorepo managed with npm workspaces + Turborepo. Users sign in via OAuth (Google/GitHub), build resumes section-by-section (work experience, education, skills, projects, certifications, languages, custom sections), and export to PDF via headless Chrome.

## Commands

Run from repo root (Turborepo fans out to both workspaces):
```
npm run dev      # backend `go run ./cmd/api` + frontend `vite`, in parallel
npm run build    # backend binary + frontend static bundle
npm run lint     # backend `go vet ./...` + frontend `oxlint`
npm run test     # backend `go test ./...` + frontend `vitest run`
```

Per-workspace (from `backend/` or `frontend/`), same scripts apply individually. Useful scoped invocations:
```
go test ./internal/auth/...              # single package
go test ./internal/auth/ -run TestName   # single test
cd frontend && vitest run src/some.test.ts
```

Do not start dev servers unless explicitly asked — the user runs and tests the app manually. If you do start one for a specific check, kill it immediately afterward.

Backend needs Postgres reachable at `DATABASE_URL` and applies `backend/internal/db/migrations/*.sql` automatically on boot. Config is loaded from `backend/.env`: OAuth settings, `JWT_SECRET`, optional `CHROME_EXEC_PATH`, and optional AI configuration. Platform AI requires `AI_PROVIDER`/`AI_API_KEY`/`AI_MODEL`; encrypted BYOK requires `AI_ENCRYPTION_KEYS` (JSON version-to-base64-32-byte-key map) and `AI_ENCRYPTION_ACTIVE_KEY`. Unconfigured AI modes remain unavailable without preventing manual editing. See README for setup and rotation.

## Runtime feature flags

- `internal/features/catalog.go` owns registered keys: `platform_ai`, `platform_ai_public`, and `byok_ai`, initially disabled.
  Removing a feature removes its key from this catalog. Historical database rows
  do not grant availability or appear in the feature API.
- `features.Service` owns evaluation and bounded caching; `features/postgres.Store`
  owns read-only PostgreSQL access. Unknown/missing flags and failed
  refreshes default off. Keep checks at feature boundaries.
- Manage registered flags directly in PostgreSQL using existing database tools.
  There is no feature-management CLI, browser flag-write endpoint, or implicit OAuth
  admin role. Preserve applied migrations, stored data, and historical audit records.
- `frontend/src/platform/features/` fetches backend decisions and provides route
  guards. `app/` assembles routes and navigation. Shared chat presentation lives in
  `components/chat/`, independent of the resume assistant.
- Reuse this boolean flag system rather than adding another SDK or module registry.

## Backend architecture (`backend/internal/`)

Layering is strict: `handlers` (HTTP adapter) → `service` (business logic) → `repository` (Postgres via pgx). Nothing skips a layer.

- **`repository/`** — one file per entity (`work_experience_repo.go`, `education_repo.go`, etc.), all following the same CRUD + reorder shape against Postgres. `reorder.go` holds the shared reorder-position logic; `json.go` shared JSON(B) column helpers.
- **`service/`** — mirrors the repository layer 1:1, plus:
  - `ownership.go`'s `ensureResumeOwner` is the *only* place resume ownership is checked. Every section-scoped service call goes through it before touching child rows — this is what makes user isolation correct regardless of which adapter (REST today, MCP potentially later) calls in. Don't duplicate ownership checks elsewhere.
  - `EntityService[TInput, TOutput]` (generic interface) is implemented by every simple child-entity service (work experience, education, project, certification, language, misc entry) so they can share one generic HTTP adapter.
- **`api/handlers/`** — one handler per concern. `EntityHandler[TInput, TOutput]` (`entity_handler.go`) is the single generic HTTP adapter for every `EntityService`-shaped entity — when adding a new simple child entity, wire it through this generic instead of writing a new handler. Non-generic entities (Skill, CustomSection, SectionConfig, ContentBlock, Resume itself, Agent) get bespoke handlers because their shapes diverge (nested items, upserts, etc.).
- **`api/router.go`** — all routes registered here; `registerEntityRoutes` wires the CRUD+reorder pattern for each `EntityHandler`. AI routes remain registered independently of runtime flags; `AIHandler` enforces flags and same-origin protection at their module boundary.
- **`auth/`** — JWT issuance/validation (`jwt.go`), OAuth provider config for Google/GitHub (`oauth.go`), signed OAuth state (`state.go`), user-ID-from-context helper (`context.go`). `api/middleware/auth.go`'s `RequireAuth` wraps protected routes; login/callback/logout and the PDF export's `/export/data` route are intentionally unprotected (the latter self-validates via a short-lived `export_token` query param since headless Chrome has no session cookie).
- **`ai/`** — provider configuration, model discovery/capabilities, Eino clients, inline PDF requests, and versioned AES-GCM key encryption. Supports `anthropic`, `openai`, `vercel` (AI Gateway), and `openrouter` at fixed endpoints. Direct OpenAI uses compatible Chat Completions tool models; Responses-only tool models are excluded. Clients return `model.ToolCallingChatModel` and normalize embedded stream errors. Zero dependency on API/service/repository packages. `AISettingsService` resolves account-owned encrypted credentials; keys never enter drafts, prompts, or public DTOs.
- **`agent/`** — the resume assistant, built on Eino via `internal/ai`, with shared prompts/tools/proposals. A request-scoped runner captures its selected credentials/model for the entire turn. `AGENT_MODEL` overrides platform chat only; personal connections share one selected model for chat/import. `AGENT_MAX_TOKENS` bounds output and the agent has a bounded tool loop. No dependency on API or main. Missing platform credentials do not prevent startup.
- **AI access/settings** — repository → service → handler layering applies to connections, preferences and invitations. Platform AI requires `platform_ai` and either `platform_ai_public` or `platform_ai_allowlist` membership; public access still requires authentication and platform configuration. `byok_ai` is independent. Flags are evaluated at the shared AI HTTP boundary. Never silently switch billing sources or replace explicit BYOK preferences when opening public access. Deleting an active key retains an explicit BYOK preference with no connection. Keep the import service's concurrency state shared while passing request-scoped parsers.
- **`service/export_service.go`** — drives headless Chrome (chromedp) against the frontend's print-only route to render a resume to PDF. Page dimensions are pinned to match `LivePreview.module.css`'s on-screen preview box exactly (see the constants at the top of the file) — if that CSS's page size changes, update these too.

## Frontend architecture (`frontend/src/`)

React 19 + TypeScript + Vite + Tailwind v4 + shadcn/ui (`components/ui/`) + TanStack Query + react-router.

- **Local-first draft/sync model** — the resume editor does not save on every keystroke to the server. Edits go into a local reducer-driven draft (`hooks/resumeDraftReducer.ts`, held in `hooks/useResumeDraft.tsx`'s context) that's persisted to `localStorage` (`lib/resumeDraftStorage.ts`) immediately. Periodically/on-demand, `lib/resumeDraftSync.ts`'s `flushDraft` diffs the draft against the last-synced snapshot and replays only the changed entities against the backend, resolving temp IDs (client-generated, prefixed `temp-`, see `lib/tempId.ts`) to real server IDs as parents sync before children. Each entity type's flush step is isolated in its own try/catch so one failure doesn't block others from syncing.
- On mount, the draft hydrates from `localStorage` if present (may hold edits never flushed, e.g. a closed tab) and only hits `GET /api/resumes/{id}/full` if no local draft exists yet.
- `hooks/useEditorPanel.ts` / `useEntryPanel.ts` manage the open/dirty state of individual entry-editing panels (e.g. one work experience entry); `registerPanel` in `useResumeDraft` folds "has an open panel with uncommitted form edits" into the app-wide unsaved-changes signal used by `EditorNavbar`.
- **`components/editor/sections/*`** — one component per resume section type, each rendering a list of that section's entries plus its add/edit panel.
- **`lib/api.ts` / `lib/http.ts`** — typed fetch wrapper (`apiGet`/`apiPost`/`apiPatch`/`apiDelete`) and `HttpError`.
- **`pages/PrintPage.tsx`** — the print-only route the backend's PDF export drives via headless Chrome; keep its layout in lockstep with `LivePreview.tsx`/`LivePreview.module.css` since the exported PDF must match the live preview pixel-for-pixel (see `export_service.go` above).

## Cross-cutting notes

- Prefer reuse over writing code from scratch. Before implementing a capability, inspect existing project code and look for suitable maintained libraries, modules, or components. Reuse or adapt an established solution when it fits; write custom code only for behavior the available solutions do not cover.

- Keep the repository minimal. Add files, folders, abstractions, helper commands, dependencies, or documentation only when necessary for current functionality, meaningful tests, or lasting operations. Prefer editing existing files and using established tools. Do not add convenience CLIs or speculative scaffolding for tasks already covered by direct database access or existing tools unless explicitly requested. A research request does not imply creating research documents; remove temporary artifacts before finishing.

- Always remove files, code, styles, dependencies, and tests made unused by a change. Trace references and run the relevant checks; do not retain obsolete implementations for hypothetical future use. Preserve applied migrations and user data.

- Go workspace (`go.work`) currently has one module (`backend`); `frontend` and any future `shared/*` npm workspace packages are separate from the Go module graph.
- Migrations are forward-and-back paired (`*.up.sql` / `*.down.sql`) under `backend/internal/db/migrations/`, applied automatically at API boot — don't hand-edit an already-applied migration; add a new one.
- The local-only pre-release migration history was intentionally replaced with `000001_init_schema`, containing 20 active tables and disabled seeds for the three current flags. Legacy databases from the old history require a backed-up reset, not a forced version change. The baseline excludes retired Jobs/template-catalog tables and test data. Preserve this baseline and add numbered migrations for subsequent changes.
