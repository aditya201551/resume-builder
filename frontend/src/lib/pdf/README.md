# Client PDF pipeline

The editor uses Forme 0.29 in a dedicated worker to generate the document. A separate,
persistent PDF.js worker parses those bytes, and visible pages are painted on canvas
with selectable text and clickable links. Download saves the very same PDF blob.
Neither preview nor download waits for backend sync or calls the Chrome export route.

- `useResumePdf.tsx`: per-editor worker lifecycle, revision tracking, error/retry, and
  download requests that wait for the latest local revision.
- `renderQueue.ts`: 200 ms debounce, 800 ms maximum wait during continuous typing,
  one active job, one latest pending job, and a two-revision cache. Superseded results
  never overwrite a newer preview.
- `pdf.worker.tsx`: DOM-free Markdown parsing, content resolution, cached font loading,
  document rendering, and section hit regions. PDF buffers are transferred to the UI.
- `ResumeDocument.tsx`: document/font envelope that selects a registered PDF template.
  Vertical is the current single-column template; shared content parts and the catalog
  live in `src/templates`. Forme handles wrapping and pagination. Design values remain
  CSS pixels and convert to points (72/96); paper dimensions are actual A4/Letter.
- `PdfPages.tsx`: visible-page rendering, detached-canvas painting before replacement,
  a minimal scoped text layer, and PDF link annotations. Zoom never regenerates a PDF.
- `designHistory.ts`: bounded formatting history, with consecutive changes to the same
  field grouped. Content edits and server acknowledgements do not enter this history.

Font files are self-hosted in `public/fonts/pdf`, with their licenses. Georgia maps to
Gelasio, Times New Roman to Tinos, and Arial/Helvetica to Arimo. Other supported families
use their own files. Fetches occur only for selected families and stay cached in the
generation worker. To refresh binaries and licenses, run from the repository root:

```
node frontend/scripts/update-pdf-fonts.mjs
```

The legacy Vertical HTML template, print page, and Go Chrome exporter remain available for
existing API consumers. They are independent of the client editor's PDF artifact and
are not the source of its preview or download. Any future server-generated documents
should use the same PDF renderer before promising identical output across both paths.

Verification: `npm run test --workspace frontend` covers generated page sizes, multi-page
vertical flow, rich text, scheduling, and formatting history. Browser verification should
also compare downloaded bytes with the displayed revision, exercise failed font loads,
test unsynced local edits, and inspect desktop/mobile pages visually. Performance needs
checking on representative mobile devices; initial WASM/font loading differs from warm
renders. No dev server is required for the unit tests or production build.
