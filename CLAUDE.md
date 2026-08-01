# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

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

Backend needs Postgres reachable at `DATABASE_URL` and applies `backend/internal/db/migrations/*.sql` automatically on boot (see `db.RunMigrations` in `cmd/api/main.go`). Config is loaded from `backend/.env` (see that file for the full list of variables — OAuth client IDs, `JWT_SECRET`, optional `ANTHROPIC_API_KEY` for the AI assistant, optional `CHROME_EXEC_PATH` for PDF export).

## Backend architecture (`backend/internal/`)

Layering is strict: `handlers` (HTTP adapter) → `service` (business logic) → `repository` (Postgres via pgx). Nothing skips a layer.

- **`repository/`** — one file per entity (`work_experience_repo.go`, `education_repo.go`, etc.), all following the same CRUD + reorder shape against Postgres. `reorder.go` holds the shared reorder-position logic; `json.go` shared JSON(B) column helpers.
- **`service/`** — mirrors the repository layer 1:1, plus:
  - `ownership.go`'s `ensureResumeOwner` is the *only* place resume ownership is checked. Every section-scoped service call goes through it before touching child rows — this is what makes user isolation correct regardless of which adapter (REST today, MCP potentially later) calls in. Don't duplicate ownership checks elsewhere.
  - `EntityService[TInput, TOutput]` (generic interface) is implemented by every simple child-entity service (work experience, education, project, certification, language, misc entry) so they can share one generic HTTP adapter.
- **`api/handlers/`** — one handler per concern. `EntityHandler[TInput, TOutput]` (`entity_handler.go`) is the single generic HTTP adapter for every `EntityService`-shaped entity — when adding a new simple child entity, wire it through this generic instead of writing a new handler. Non-generic entities (Skill, CustomSection, SectionConfig, ContentBlock, Resume itself, Agent) get bespoke handlers because their shapes diverge (nested items, upserts, etc.).
- **`api/router.go`** — all routes registered here; `registerEntityRoutes` wires the CRUD+reorder pattern for each `EntityHandler`. The Agent route is only registered `if h.Agent != nil` (see below).
- **`auth/`** — JWT issuance/validation (`jwt.go`), OAuth provider config for Google/GitHub (`oauth.go`), signed OAuth state (`state.go`), user-ID-from-context helper (`context.go`). `api/middleware/auth.go`'s `RequireAuth` wraps protected routes; login/callback/logout and the PDF export's `/export/data` route are intentionally unprotected (the latter self-validates via a short-lived `export_token` query param since headless Chrome has no session cookie).
- **`agent/`** — the AI assistant (content rewrite suggestions), built on Eino + Claude. Deliberately has zero dependency on `cmd/api` or `internal/api` — only `internal/service`, `internal/auth`, and Eino/Claude — so it can be split into its own service later without changing this package (see `agent/doc.go` for the exact split plan). It runs in-process today, called directly from `agent_handler.go`. Entirely optional: absent `ANTHROPIC_API_KEY`, `agent.LoadConfig()` returns an error, `main.go` logs a warning, and the router skips registering the agent route (`h.Agent` stays nil) rather than failing to boot.
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

- Go workspace (`go.work`) currently has one module (`backend`); `frontend` and any future `shared/*` npm workspace packages are separate from the Go module graph.
- Migrations are forward-and-back paired (`*.up.sql` / `*.down.sql`) under `backend/internal/db/migrations/`, applied automatically at API boot — don't hand-edit an already-applied migration; add a new one.
