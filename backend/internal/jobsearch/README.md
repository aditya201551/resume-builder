# Job discovery

Manual controls and the job assistant use the same versioned `Intent`. Location
branches are alternatives (OR). A remote country means candidate work location;
an office country/city means attendance location. Work authorization is not
inferred from either. `all` with an office country only matches office roles;
include a separate remote branch to search both.

The first release supports keywords, arrangement, location, seniority, employment,
posting/update age and excluded companies. Salary and sponsorship are not hard
filters yet. The assistant explains unsupported requirements rather than claiming
to enforce them. Resume fit, tailoring and application tracking are separate
future features.

## Boundaries

- `internal/jobsearch`: public intent, validation, eligibility and page types.
- `internal/jobmap`: existing normalized jobs and fixed-host provider clients.
- `internal/service/job_discovery_service.go`: on-demand page orchestration,
  provider selection, budgets, validation and continuation tokens.
- `internal/repository/job_discovery_repo.go`: public page membership cache and
  user-scoped saved-state joins. Migration 000009 adds the page store.
- `internal/jobagent`: Eino structured search planning using `internal/ai`.
- `internal/service/job_assistant_service.go`: authenticated assistant budget and
  bounded conversation validation. No resume mutation tools are provided.
- `frontend/src/pages/JobsPage.tsx`: Discover/Saved with one inspectable search.

`POST /api/jobs/search` accepts `{intent, nextToken?, savedOnly}` and returns
`{jobs, sources, nextToken?, retry}`. Counts in the UI describe loaded, deduplicated
results, not a provider's global inventory. One request fetches at most one page
per provider/location alternative. More pages require a user action. Source
failures do not advance that source's continuation. A changed intent requires a
new continuation. No map viewport is part of this contract.

`POST /api/jobs/assist` accepts `{intent, message, history?}`. Its response contains
`{intent, message, apply}`. The frontend executes an applied intent through the
same discovery endpoint. Manual edits made during an AI request take precedence
over its stale response. The production legacy `GET /api/jobs` path is retired.

Shared provider pages expire after 24 hours and are capped at 500 entries, with
up to 100 normalized jobs accepted per upstream page. Eviction removes unreferenced
discovery-cached jobs while retaining bookmarks. Existing provider request budgets,
rate spacing, retry backoff and expiry checks still apply. Assistant requests are
capped at 30 per user per UTC day; manual search remains available.

JSearch requires a chosen office country and its existing API configuration.
Himalayas remote search can use a chosen candidate country or search all hiring
regions. Bulk-feed imports and unused company-feed workers have been removed. An unavailable or exhausted
source is reported explicitly; no US default or alternative source is silently
substituted.

Keep future providers behind normalized job/page contracts. Alerts, fit analysis,
application activity and optional office maps should depend on these contracts
without changing the meaning of location eligibility.
