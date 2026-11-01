# Worldwide tech job data strategy

Research checked November 1, 2026. Scope updated by user: worldwide technology jobs, modular architecture, AI and manual search, no maps for now, no whole-Himalayas crawl. Recommendations below are design proposals; provider capability and price statements are separately sourced.

## Which sources should power the first version?

### Takeaway
Use Himalayas for remote search, retain JSearch as the general aggregator, and keep a small optional set of direct company boards. A worldwide audience is achievable with country-aware searches, but comprehensive worldwide inventory is not established by these providers' documentation or by the nine boards currently configured.

### Cited Findings
- Himalayas has filtered remote search at `/jobs/api/search`: country, keywords, seniority, employment type, company, timezone, sort and page. Its country parameter accepts ISO codes; country-based search includes worldwide matches unless explicitly excluded. Search pages may contain fewer than 20 results before the end: continue only when the user asks, and stop on an empty page, not a short page. `totalCount` is an estimate that can be up to a day old; results can move between pages, so deduplicate by `guid`. A complete country inventory would require a full browse walk instead, which is outside this requested design. — [Himalayas API](https://himalayas.app/api)
- Himalayas documents free keyed limits of 60 requests/minute and 10,000/UTC day, shared across account keys. Unkeyed callers share a public allowance. Its API refreshes daily, and requests must run through a backend because browser CORS is unavailable. Individual-job checks can report open/closed status; answers can be cached ten minutes. — [Himalayas API reference](https://himalayas.app/docs/remote-jobs-api)
- JSearch supports international searches including India; use country and language plus a location in the query. `/search-v2` uses a returned cursor. `work_from_home=true` selects remote jobs; false or omission returns both remote and onsite, not onsite-only. Current listed direct prices: free 200 requests/month; Pro USD25/month for 10,000, then USD0.003/additional request; pay-as-you-go USD0.005/use. Account/plan pricing must be rechecked when purchasing. Its example includes `work_arrangement` and salary fields, but missing salary is expected. — [JSearch capabilities, FAQ and pricing](https://www.openwebninja.com/api/jsearch)
- Ashby's public posting endpoint is per named company board. Crucially, `isRemote=true` can mean either Remote or Hybrid: use `workplaceType` for distinction. It supports secondary locations, compensation when requested, published time, apply URL and `isListed`; unlisted jobs should not appear in a public list. Missing fields remain missing. — [Ashby public posting API](https://developers.ashbyhq.com/docs/public-job-posting-api)
- Greenhouse provides unauthenticated public GET job-board endpoints identified by board token; descriptions can be included with `content=true`. This is a company-board API, not a global query service. — [Greenhouse Job Board API](https://docs.greenhouse.io/job-board)
- Lever exposes per-site postings with skip/limit and location filters; `workplaceType` distinguishes unspecified, on-site, remote and hybrid but is not itself filterable. Its country field can be unknown. — [Lever Postings API](https://github.com/lever/postings-api)
- Adzuna is a possible later search source: its API supports keyword/location querying and country-qualified paths. Search returns a description snippet, not guaranteed full content. Exact India API availability for the intended account and its taxonomy should be tested rather than assumed from the Indian website. — [Adzuna overview](https://developer.adzuna.com/overview), [Search API](https://developer.adzuna.com/docs/search)
- Current Adzuna terms say commercial use has a 14-day evaluation trial and then requires a licence agreement; free access is for evaluation, and limits/plans are account-specific. It has prescribed attribution, and acquisition data must be removed from pages on termination. Do not treat it as an unrestricted free fallback. — [Adzuna API terms](https://developer.adzuna.com/docs/terms_of_service)
- Jooble's official help, modified August16,2026, states each country domain requires its own API key and a free key has a lifetime quota of500 requests, not500/month. It supports keyword/location/page queries. Its similarly named ATS API publishes employer jobs and is a different product from public vacancy search. — [Jooble REST API help](https://jooblehelpcenter.freshdesk.com/en/support/solutions/articles/60001448238-rest-api-documentation)

### Inferences
- Start with the two existing aggregator adapters. Adding many providers now adds duplicate handling, licence rules, filters and paging differences before solving the primary product problem. Expand after country-by-country coverage tests identify a concrete gap.
- Preserve country, language and currency separately. Global user support does not imply that every remote role accepts applicants globally. Onsite location and remote eligibility need separate meanings even if their UI shares country pickers.
- Keep office arrangements unknown when the source does not reliably classify them. Do not infer onsite simply because a remote flag is false; hybrid can coexist with a false remote flag in some provider schemas, and Ashby's boolean has the opposite ambiguity.
- Curated ATS feeds can improve employer provenance, but nine boards cannot reasonably represent worldwide technology demand. They are optional enrichment, not the primary broad-search strategy.
- Recheck exact JSearch billable units for multi-page requests and active account terms before budgeting. Two providers, query variants, pagination and AI refinements can multiply calls; the free200/month is a prototype allowance, not a sustainable production-search budget.

### Gaps
- No independent, current measurement of recall or freshness for the app's target roles/countries/languages was performed. Provider coverage marketing is not evidence of exhaustive results.
- No reliable uniform native hybrid filter across all providers was established. An adapter may need to filter fetched pages locally, then explain partial coverage.
- No costed plan for the user's expected traffic can be given without searches/user, pages/search, cache hit rate and AI usage assumptions.

## How should fetching, caching and eligibility work without bulk import?

### Takeaway
Fetch only requested searches and pages, with shared query caches, bounded retention and explicit source budgets. Treat job text and provider metadata as evidence; eligibility, freshness and result counts should not be invented from a map, a resume, or the amount already cached.

### Cited Findings
- Himalayas permits displaying its jobs on owned apps/sites, requires the unchanged API `applicationLink` for Apply, visible source credit/link, removal after expiry, server-only keys, and caching where possible. It prohibits onward syndication to other job boards/aggregators/search engines. — [Himalayas terms, API section113](https://himalayas.app/terms)
- OpenWeb Ninja's October14,2026 terms license authorized API output for use in broader commercial products, but only to the extent of its rights; the customer remains responsible for original-source rights, terms and privacy obligations. No universally applicable exact JSearch cache-retention duration was established from that page. — [OpenWeb Ninja terms, section2](https://www.openwebninja.com/terms)
- Himalayas' API page describes country eligibility as unrestricted or matching location restrictions plus a compatible standard UTC offset. Its data dictionary instead represents country entries as objects (`alpha2/name/slug`) and timezone restrictions as UTC strings; it says empty timezone arrays are unrestricted. The current Go adapter expects string countries and numeric offsets. — [Himalayas API](https://himalayas.app/api), [Data dictionary](https://himalayas.app/docs/data-dictionary), [local adapter](C:/Users/adity/OneDrive/Desktop/resume-builder/backend/internal/jobmap/himalayas.go)
- A read-only live public GET on November1 to `https://himalayas.app/jobs/api/search?country=IN&q=software%20engineer&page=1` returned20 jobs with top-level `comments,updatedAt,offset,limit,totalCount,jobs`. Nonempty country restrictions were string arrays such as `["India"]`, and timezone offsets were numeric floats including5.5. This is an observation of that request, not a guarantee for all endpoints; the live shape matched the current adapter and conflicted with the dictionary. The returned estimated count1174 is query/time-specific and must not become a coverage claim. — [Exact tested API URL](https://himalayas.app/jobs/api/search?country=IN&q=software%20engineer&page=1)

### Inferences
- Recommended flow: normalized search specification -> capability-aware provider selection -> cache lookup -> one page from selected providers on miss -> normalize/validate/deduplicate -> deterministic eligibility checks -> bounded ranking -> result session. AI and manual search must use this same flow.
- Key a shared provider cache by provider, normalized query, country/eligibility, language, arrangement, supported filters, sort and page/cursor; do not include private resume text. Cache membership as ordered provider job IDs so results from unrelated searches do not silently enter a session. Store private per-user match explanations separately.
- Set bounded retention and row/byte budgets with eviction. A24-hour Himalayas query cache is a reasonable initial proposal consistent with its daily updates; it is a proposed app policy, not a prescribed provider TTL. Other providers need provider-specific policy. On details/apply, check availability when supported. Hide expired/closed listings and retain only permitted bookmark/status information.
- Use single-flight locks across instances, timeouts, backoff, concurrency limits, per-provider hard spend caps and per-user search/AI limits. Cancellation of a browser should not create infinite abandoned searches; background work should be bounded.
- Never fetch all pages just to fill an exact counter. Maintain each provider's cursor separately, buffer returned results for stable paging, deduplicate within fetched batches and advance only when the user loads more. A short filtered page is honest; don't hide potentially unbounded upstream crawling behind one request.
- Report “20 results loaded” or “results from these sources” and explain estimates where used. Do not sum provider estimates into an exact unique global count. Reranking a fetched batch does not mean those are the best jobs worldwide.
- Deduplicate exact provider IDs first. Cross-provider application URL normalization can suggest duplicates but must preserve each source's mandated Apply URL. Same company+title is insufficient: separate requisitions, countries or work arrangements may be distinct.
- Represent eligibility as eligible, ineligible, or unknown with evidence. Country names, multi-country groups and half/quarter-hour offsets need normalization; country permission is not visa sponsorship or legal work authorization. Unknown source restrictions should remain visible as uncertainty.
- Capture fresh real-response fixtures for both search and detail endpoints before reusing adapters; support verified schema variants deliberately rather than assuming all current documentation agrees.

### Gaps
- The Himalayas dictionary and observed JSON contract conflict; this needs validation/compatibility tests during implementation, not an assertion that a wholesale adapter rewrite is already necessary.
- A public company-board GET endpoint is not by itself a verified blanket third-party republication licence for every employer. Validate terms/permissions as the company registry expands. Do not describe public API access as unlimited free global data rights.
- No raw provider keys or account settings were accessed during this research.

## What can be reused from the current repository?

### Takeaway
Reuse provider parsing, stable IDs, bookmarks, availability checks, budget/backoff infrastructure and the handler-service-repository layering. Replace the full-index filtering and single-page aggregation assumptions with query-scoped bounded search sessions.

### Cited Findings
- The current configuration supports OpenWeb Ninja or RapidAPI JSearch routes and enforces a maximum200 monthly requests even if a higher value is configured. That limit is an app prototype guard; it is not the API's universal production limit. It also loads a curated registry of nine Ashby/Greenhouse boards. — [config.go](C:/Users/adity/OneDrive/Desktop/resume-builder/backend/internal/jobmap/config.go), [companies.json](C:/Users/adity/OneDrive/Desktop/resume-builder/backend/internal/jobmap/companies.json)
- `FetchJSearch` calls `search-v2` with `num_pages=1`, decodes jobs, and does not decode/pass through a continuation cursor. The existing UI's later pages are local cached-job pages, not full upstream pagination. Ashby/Greenhouse/Lever parsing is already implemented. — [providers.go](C:/Users/adity/OneDrive/Desktop/resume-builder/backend/internal/jobmap/providers.go)
- JSearch query work has a persisted24-hour cache claim, shared request budget, a45-second task deadline and bounded background gate. The current service reads all cached visible jobs and filters/ranks them in Go, not query-scoped paginated SQL. — [jsearch_service.go](C:/Users/adity/OneDrive/Desktop/resume-builder/backend/internal/service/jsearch_service.go), [job_service.go](C:/Users/adity/OneDrive/Desktop/resume-builder/backend/internal/service/job_service.go), [job_repo.go](C:/Users/adity/OneDrive/Desktop/resume-builder/backend/internal/repository/job_repo.go)
- The full Himalayas worker was disabled in this chat's completed change. Existing individual availability checks remain, and source output overrides stale persisted refreshing/error statuses with paused. — [job_workers.go](C:/Users/adity/OneDrive/Desktop/resume-builder/backend/internal/service/job_workers.go), [job_service.go](C:/Users/adity/OneDrive/Desktop/resume-builder/backend/internal/service/job_service.go)

### Inferences
- A modest cached dataset does not automatically choke Postgres. The scalable-design concern is the cumulative request pattern: global scans per search, large descriptions, perpetual import, repeated provider calls and unbounded AI context. Measure memory/latency instead of attributing crashes to a row count alone.
- Keep the bulk crawl disabled. Add a Himalayas filtered-search adapter and JSearch cursor support behind a provider-neutral search interface. Prefer additive migrations and reuse stable job IDs/saved records so the redesign does not erase bookmarks.
- Do not rebuild the entire backend solely to remove maps. Separate providers, normalization/eligibility, retrieval/cache, ranking, and user activity; UI navigation and the AI orchestrator can evolve independently.

### Gaps
- These are source-code observations, not a production load benchmark. This task did not modify application code, buy a provider plan, or generate any paid requests.
