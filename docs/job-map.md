# Jobs

The authenticated Jobs tab has third-party listings and a Direct jobs placeholder. The initial scope is technology roles worldwide. Search operates on the listings actually indexed, not a claim to cover every opening worldwide.

## Run locally

Run `npm install` after pulling the feature, then restart `npm run dev`. The API applies migrations through `000008_external_job_search_cache` and refreshes its sources in the background. Listings arrive incrementally; the page polls while collection is running. No new account or API key is needed for the seeded company feeds, Himalayas, basemap, or bundled city search. A free Himalayas API key is recommended for a dedicated allowance rather than sharing the public limit.

Optional settings in `backend/.env`:

```dotenv
# Use rapidapi for a RapidAPI subscription; openwebninja (default) for a direct key.
JSEARCH_PROVIDER=rapidapi
# Key for the selected provider. Never exposed to the browser.
JSEARCH_API_KEY=
JSEARCH_MONTHLY_LIMIT=200

# Optional free Himalayas key; server-side only. Public access shares an allowance.
HIMALAYAS_API_KEY=

# Extends city search when the bundled gazetteer has no match.
GEOAPIFY_API_KEY=
GEOAPIFY_DAILY_LIMIT=1000

# Employer feeds only: minimum 1h; default 6h. Himalayas syncs daily independently.
JOB_MAP_REFRESH_INTERVAL=6h

# Optional replacement company registry; relative paths resolve from backend/.
JOB_MAP_COMPANIES_FILE=
```

JSearch is optional. Without its key the page still combines curated employer feeds with the Himalayas remote-job feed. With a key, a workplace country, focused map area (under 20 degrees across), or independent remote-work country can trigger a supplemental search. An empty role field searches for technology jobs in that location; an unscoped world view does not call the provider. Each search requests one provider page, not every opening in the area. The search endpoint immediately serves indexed listings while a bounded, 45-second background task imports new results. Browser cancellation does not cancel this task; application shutdown does. The UI polls while collection is active.

Successful JSearch imports are cached for 24 hours from completion. A separate one-minute, token-owned database lock prevents concurrent requests for the same query across instances. Cleanup releases only its own lock, using a fresh five-second context; a crashed process's lock expires after one minute. Provider/budget failures defer that query for five minutes (or longer when `Retry-After` requires it), so UI polling cannot rapidly exhaust the allowance. Application shutdown allows an immediate retry on the next instance. Old canceled queries in `job_search_cache` do not block the new cache. Atomic database counters still enforce a maximum of 200 outbound JSearch requests per UTC calendar month, including unsuccessful attempts, across users and process restarts. No automatic paid fallback is implemented. The quota covers this app's requests only; another app using the same key can consume the vendor's allowance independently.

For RapidAPI, subscribe to [JSearch](https://rapidapi.com/letscrape-6bRBa3QguO5/api/jsearch) and set `JSEARCH_PROVIDER=rapidapi`. Requests use `jsearch.p.rapidapi.com/search-v2` with `X-RapidAPI-Key` and `X-RapidAPI-Host`. Direct keys use `api.openwebninja.com/jsearch/search-v2` with `x-api-key`. Both adapters read the current `data.jobs` response and request one page per search. There is no automatic retry through the other provider. Restart the backend after changing these settings.

## Sources and collection

`backend/internal/jobmap/companies.json` seeds nine real employer boards validated during implementation. It records company websites and ATS board identifiers, not invented office positions or job content. Operators can replace the registry using the configuration above. Supported sources are `ashby`, `greenhouse`, `lever`, and `career_page`. Removing a source from the registry removes its jobs from discovery; existing user bookmarks remain readable.

The employer worker refreshes two sources concurrently. Successful complete employer feeds close jobs that disappeared. Supplemental JSearch responses and single career pages never close unseen jobs merely because they were not returned. Listings expire at their supplied expiry time or fall out of discovery after 72 hours without a successful observation or availability check. Unavailable saved jobs retain their closed/unconfirmed state. Unsaved records not observed or checked for 14 days are pruned.

Himalayas walks `GET /jobs/api?limit=20`, following opaque `nextCursor` values until the feed ends. Each page's technology listings and cursor are committed together. A separate worker starts on boot and checks every minute for a due or interrupted sync. Completed walks repeat after 24 hours; 10-minute passes fit inside shared 15-minute leases. Reaching a pass's time limit logs `himalayas_sync_checkpointed` and keeps the source refreshing, allowing the next pass to continue without a forced 15-minute idle period. Timers start before the initial pass. An invalid cursor restarts the next walk without deleting cached jobs. Short pages do not end a walk if a cursor remains. The visible Jobs page polls our database-backed API every five seconds during collection and once a minute otherwise, so it also notices resumed passes.

Provider requests are spaced at least 1.1 seconds apart through a shared database lease and capped at 10,000 attempts per UTC day across this app, including detail checks. HTTP 429 honors `Retry-After`; exhausted local budgets wait until the next UTC day. A provider account used by another integration can reach its own allowance sooner. Anonymous access can be throttled by unrelated public callers. The backend uses `HIMALAYAS_API_KEY` as a bearer token when present; it never sends it to the browser.

Opening a Himalayas job checks its documented single-job endpoint, with a 10-minute shared check cache and a six-second request deadline. A 200 refreshes the record; a 410 marks it closed while preserving bookmarks. A 404, timeout, malformed response, or provider outage leaves availability unchanged and shows a verification warning. A cached feed page cannot reopen a confirmed closed job; a later explicit successful check can. The cursor feed is not an atomic snapshot, so even a completed walk does not automatically close absent jobs.

Original application links and visible Himalayas attribution are preserved. Empty versus absent restrictions are distinguished according to the provider's documented schema; country matching also respects its standard-UTC-offset restrictions. Remote jobs never receive office markers. Himalayas `pubDate` is stored as `sourceUpdatedAt`, separately from original posting dates, fetch timestamps, and explicit availability checks. List dates are labelled Posted or Updated; sorting and date filters use the posting date when available, otherwise the source update date. Refreshing a record never promotes it using its fetch timestamp. The migration converts previously stored Himalayas posting dates to source update dates.

A replacement company registry is an array like:

```json
[
  {"name":"Linear","website":"https://linear.app","provider":"ashby","board":"linear"},
  {"name":"Your employer","website":"https://example.org","provider":"career_page","careersUrl":"https://example.org/careers"}
]
```

Career-page collection is opt-in through that operator-controlled file. It fetches only the selected public HTTPS page, checks robots.txt conservatively, blocks private-network destinations and cross-host redirects, and reads `JobPosting` JSON-LD, including `@graph`. It does not execute JavaScript, follow arbitrary links, log in, bypass access controls or perform unrestricted crawling. Pages without usable structured data report a source error. HTML-selector scraping and JobSpy are research alternatives, not runtime dependencies in this release.

Public feed access and an open-source scraper licence do not themselves establish permission to republish a source's data. Source attribution, permitted display/storage, original application links and provider-specific terms still apply when expanding the registry.

## Location semantics

Physical workplaces, remote hiring geography, and source identity are separate fields. Ashby's explicit workplace type overrides its broader `isRemote` flag (which may also be true for hybrid roles). Multiple physical locations remain attached to one listing. Source text is retained, and missing arrangements remain unknown.

The bundled gazetteer contains 2,098 major cities and 252 country/territory records derived from GeoNames. City positions are approximate centroids and are labelled accordingly. Unknown places remain unmapped; no random displacement, headquarters substitution, invented office address or precise-distance claim is used. User location is requested on a button click with high accuracy enabled and cached positions disabled, rounded before nearest-city lookup, and never stored as a user profile attribute. Device coordinates drive the map; the nearest-city response supplies a label only. If city lookup fails, the map still centres on the device reading. Coarse browser readings show an accuracy notice, and a later manual city choice cancels a pending location action.

Workplace bounds/country never change the independently selected remote-work country. Unknown remote regions require explicit opt-in when a remote country is selected. Remote jobs have no map markers. Geographic and timezone matching do not establish citizenship, visa or employment eligibility.

When a map is displayed, initial load, pan, zoom and resize automatically search the actual visible bounds after a 400ms pause. There is no Search this area button. Viewport changes replace the current browser history entry, reset pagination, and avoid rounded-coordinate feedback loops. Previous results remain visible while the next area loads; superseded requests are cancelled. Listing selection highlights its marker without moving the camera, so an asynchronous result cannot pull the view away from the user's area. Native MapLibre camera calls use `center: [longitude, latitude]`; React's `initialViewState` latitude/longitude fields are not valid native jump coordinates.

Map clusters represent workplaces across all matching indexed results, independently of the paginated list. The search API returns lightweight `mapJobs` without descriptions or application data, and excludes remote roles, closed jobs and offices outside the selected area. Multiple matching workplaces for one job can produce multiple markers. `mapped` counts unique jobs across all list pages and `remoteCount` counts all matching remote roles. The UI reports both separately from the combined indexed total. Area searches show physical workplaces before remote roles in the list. A cluster opens an accessible list of its jobs, including coincident city-centre markers.

## Architecture

- `internal/jobmap`: provider-neutral models, normalization, public-feed adapters, structured career-page collection, gazetteer and pure search rules.
- `internal/repository/job_repo.go`: listing persistence, user-scoped bookmarks, refresh leases and usage counters.
- `internal/service/job_service.go`: refresh orchestration, optional provider search, filtering and geocoder budgets.
- `internal/api/handlers/job_handler.go`: authenticated HTTP adaptation and query validation.
- `frontend/src/pages/JobMapPage.tsx`: URL-backed search, source sections, results and mobile views.
- `frontend/src/components/jobs`: location search, details, and lazy MapLibre map.

The feature uses existing Go/pgx/Postgres and React/TanStack Query. It does not require PostGIS, Redis, Python, a new service or a paid model. PostGIS can replace in-memory spatial filtering if the indexed corpus grows enough to justify it. The map renderer and CSS load only when a map is displayed; the Direct jobs page does not fetch job data or geolocation.

## Verification

Normal `npm run test`, `npm run build` and `npm run lint` remain the entry points. Opt-in checks from `backend/`:

```powershell
$env:JOB_MAP_LIVE_TEST='1'
go test ./internal/jobmap -run TestLivePublicFeeds -v
Remove-Item Env:JOB_MAP_LIVE_TEST

$env:JOB_MAP_DB_TEST='1'
go test ./internal/repository -run TestJobMapPostgresIntegration -v
Remove-Item Env:JOB_MAP_DB_TEST
```

The database check reads only `DATABASE_URL` from the local `.env`, or accepts `JOB_MAP_TEST_DATABASE_URL`. It creates and drops a disposable schema, never modifying the app's users/listings. It checks forward/back migrations, authenticated routing, bookmark isolation, closed listings and concurrent budget reservation. Do not enable live-network checks in ordinary CI runs.

## Attribution and references

- GeoNames: https://www.geonames.org/ — cities15000, countryInfo and timeZones downloads, retrieved 2026-10-31, CC BY 4.0. Derived `places.json` retains aliases, city centroids, country metadata and standard timezone offsets. See `scripts/update-job-map-places.py` to regenerate it.
- GeoNames licence: https://download.geonames.org/export/dump/readme.txt
- OpenFreeMap: https://openfreemap.org/ — basemap; required map-data attribution remains visible.
- OpenStreetMap: https://www.openstreetmap.org/copyright
- Himalayas: https://himalayas.app/api and https://himalayas.app/terms
- JSearch: https://www.openwebninja.com/api/jsearch
- Geoapify: https://apidocs.geoapify.com/docs/geocoding/forward-geocoding/
- Ashby: https://developers.ashbyhq.com/docs/public-job-posting-api
- Greenhouse: https://docs.greenhouse.io/job-board
- Lever: https://github.com/lever/postings-api

Internal publishing, employer submissions, matching, resume tailoring, alerts and routing/commute estimates remain future features.

## Startup and diagnostic logs

Startup starts both collection workers in the background; it does not force a fresh download of every provider. Employer boards check their shared refresh lease (six hours by default). Himalayas resumes a persisted cursor, or starts a new walk when the last completed walk is at least 24 hours old, subject to backoff and another process's lease. JSearch only runs for an eligible user search with its key configured. Existing database listings remain available during collection.

SIGTERM or Ctrl+C cancels background work, drains HTTP requests, and waits up to 20 seconds for shutdown. Status writes and lease releases use bounded cleanup contexts that survive cancellation. Interrupted employer refreshes also release their gate. A hard kill can still leave leases until expiry; minute-level schedule checks recover after expiry. Canceled client requests log `http_request_canceled` at INFO and do not produce a synthetic HTTP 500. Actual operation timeouts return 504; genuine unexpected failures remain 500.

The backend emits JSON logs to stdout with explicit INFO, WARN and ERROR levels. Set optional `LOG_LEVEL=debug` in the backend environment for cache/check-skip diagnostics; the default is info. Allowed values are debug, info, warn and error. Restart or redeploy to apply a level change.

Filter Railway runtime logs by event names such as `job_workers_started`, `himalayas_sync_started`, `himalayas_sync_skipped`, `himalayas_sync_progress`, `himalayas_sync_completed`, `himalayas_sync_paused`, `himalayas_check_failed`, `employer_source_failed`, `jsearch_failed`, or `http_request_failed`. Progress is emitted after the first committed page and every 25 pages, with page/record counts, elapsed time and whether more pages remain. Skip logs include the reason and next sync/retry time where known. Failure logs identify the operation, source, HTTP status/retry delay where available, and number of committed pages. Status writes, checkpoint reads and lease releases also log failures.

HTTP failures log method, route pattern, status and elapsed time, excluding query strings, request bodies, cookies and authorization headers. Known environment secrets are redacted from error attributes; OAuth response bodies and provider request URLs are excluded. Resume-import and agent failures log error codes/types rather than extracted resume content, chat prompts or tool arguments. Normal startup is logged at INFO, rather than appearing as an error merely because the standard logger wrote to stderr.
