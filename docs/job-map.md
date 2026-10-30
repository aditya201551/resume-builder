# Job map

The authenticated Job map tab has third-party listings and a Direct jobs placeholder. The initial scope is technology roles worldwide. Search operates on the listings actually indexed, not a claim to cover every opening worldwide.

## Run locally

Run `npm install` after pulling the feature, then restart `npm run dev`. The API applies migration `000006_job_map` and refreshes its sources in the background. The first listings normally arrive within a few seconds; the page polls while that refresh is running. No new account or API key is needed for the seeded company feeds, Himalayas, basemap, or bundled city search.

Optional settings in `backend/.env`:

```dotenv
# Direct OpenWeb Ninja key, not a RapidAPI key. Never exposed to the browser.
JSEARCH_API_KEY=
JSEARCH_MONTHLY_LIMIT=200

# Optional free Himalayas key; the public endpoints also work without it.
HIMALAYAS_API_KEY=

# Extends city search when the bundled gazetteer has no match.
GEOAPIFY_API_KEY=
GEOAPIFY_DAILY_LIMIT=1000

# Minimum 1h; default 6h. Existing required app settings remain unchanged.
JOB_MAP_REFRESH_INTERVAL=6h

# Optional replacement company registry; relative paths resolve from backend/.
JOB_MAP_COMPANIES_FILE=
```

JSearch is optional. Without its key the page still combines curated employer feeds with Himalayas remote-job searches. With a key, an explicit role query and workplace country, focused map area, or independent remote-work country can trigger a supplemental search. Repeated queries share a 24-hour cache lease. Atomic database counters enforce a maximum of 200 outbound JSearch requests per UTC calendar month, across users and process restarts. No automatic paid fallback is implemented. The quota covers this app's requests only; another app using the same key can consume the vendor's allowance independently.

## Sources and collection

`backend/internal/jobmap/companies.json` seeds nine real employer boards validated during implementation. It records company websites and ATS board identifiers, not invented office positions or job content. Operators can replace the registry using the configuration above. Supported sources are `ashby`, `greenhouse`, `lever`, and `career_page`. Removing a source from the registry removes its jobs from discovery; existing user bookmarks remain readable.

The worker refreshes two sources concurrently. Successful complete employer feeds close jobs that disappeared. Bounded Himalayas searches, supplemental JSearch responses and single career pages are partial collections and never close unseen jobs merely because they were not returned. Listings expire at their supplied expiry time or fall out of discovery after 72 hours without a successful observation. Unavailable saved jobs retain their closed/unconfirmed state. Unsaved records older than 14 days are pruned.

Himalayas uses five bounded technology searches. Its original application link and source attribution are preserved. Empty versus absent restrictions are distinguished according to the provider's documented schema; country matching also respects its standard-UTC-offset restrictions. Expiry does not guarantee a role cannot close sooner: the original posting remains authoritative.

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

Map clusters represent workplace markers from the current result page, including multiple workplaces for a single job. The UI separately reports unique jobs mapped on that page and matching indexed listings. A cluster opens an accessible list of its loaded jobs, including coincident city-centre markers.

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
