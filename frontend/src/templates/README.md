# Resume templates

Templates are code modules. Each template owns a complete `defaults` object for its
layout, fonts, colors, margins, spacing, entry styles, and section arrangement. A resume
stores its selected `templateId` and its own `ResumeDesign`, including user overrides.
Selecting a different template applies its defaults while retaining section visibility,
custom titles, and custom sections. Selecting the current template preserves its edits.
The Design panel's **Restore defaults** action explicitly reapplies the current
template's starting formatting and layout. Content, section visibility, custom titles,
and custom sections remain intact. Undo formatting restores the previous customization
in one step.

## Catalog and capabilities

`backend/internal/design/templates.json` is the single catalog source. Go embeds it,
and the frontend imports it at build time. `GET /api/templates` serves the same catalog
for API consumers and AI tools without querying Postgres. Only **Vertical**, a
single-column layout with a full-width header, is currently registered.

Each definition has a stable string ID, renderer key, revision, display metadata,
supported layout modes, capabilities, individual supported design field paths, and
`defaults` (all `ResumeDesign` fields except `templateId`). This JSON is the single
source of default formatting for both frontend and backend. The compatibility exports
in `defaultResumeDesign.ts` and `design.Default` read it and return independent copies.
The shared design schema defines field types/options/ranges. The template capability
list filters the Design panel and validates AI formatting actions. Layout-specific
settings can remain stored while irrelevant to the current template.

`templates/vertical/VerticalPdfTemplate.tsx` owns page composition. Shared
`templates/shared/pdfParts.tsx` supplies headers, sections, entries, and footers.
`templates/pdfRegistry.ts` connects renderer keys to PDF components. A template receives
resolved content and the shared design; it never fetches or mutates resume entities.

## Add or update a template

1. Add its definition to `backend/internal/design/templates.json`, including complete
   `defaults` matched to its intended composition. Reuse the shared controls, and
   declare only fields its renderer actually honors. Defaults must pass design
   validation and use a supported layout mode; Go checks these when loading the catalog.
2. Implement its PDF layout using shared parts and register it in `pdfRegistry.ts`.
   New document structures may add shared parts; avoid copying entity render logic.
3. Add capability, section-order, pagination, and PDF rendering coverage. Gallery cards
   are driven by catalog metadata.
4. Increment the template revision when changing a renderer or its visual behavior.
   Revision participates in the PDF cache identity, so the new build cannot reuse the
   previous template's document artifact.
5. Deploy the app. Existing resumes select the updated template on the next app load;
   content and saved formatting overrides are preserved. Missing settings are filled
   from that resume's selected template defaults. Previously downloaded PDFs are files
   and do not change. This is a code deployment workflow, not live database publishing.

## Existing resumes

The old catalog used database UUIDs and had Classic/Vertical Split renderers. Go reads
and frontend draft hydration normalize retired IDs to `vertical`. Split/mixed section
orders flatten into the active single-column list, preserving custom sections, hidden
state, and title overrides, then appending any remaining references once. Unsynced
version-2 local drafts are retained. Legacy layout preferences stay dormant in JSON;
normalization is idempotent and does not replace saved styling with defaults. No
database migration or local-storage version change is needed: the persisted design
JSON retains the same flat structure. Defaults affect new resumes, missing settings,
and deliberate switches to a different template.

There is no active template repository/database lookup. Historical migrations and the
legacy `templates` table are retained for compatibility; catalog edits there have no
effect. Resume content and shared formatting still persist in Postgres.

The old Chrome export endpoint has a separate HTML renderer in `registry.ts`. It
supports Vertical as a compatibility path and is not used by client PDF preview or
download. Adding a PDF template does not automatically add legacy HTML export support.
