# Import existing resume

The Home and My resumes pages offer a standalone PDF import dialog. Importing
creates a new resume and opens its AI workspace. No file is attached to chat:
subsequent messages use the current structured resume through the existing draft
context. Existing resumes are never overwritten.

`POST /api/resumes/import` requires authentication and exactly one multipart field
named `file`. Limits are 10 MB, 1-10 pages, and an unlocked, structurally valid PDF.
The backend validates the PDF with pdfcpu before calling the AI provider. The PDF
is processed in memory, with no application file storage or Files API upload.
Provider-side handling follows the configured provider's policies.

The document is sent inline as base64. `internal/ai/PDFClient` supports the existing
Anthropic and OpenRouter configurations. It uses `AI_PROVIDER`, `AI_API_KEY` and
`AI_MODEL`, independently of chat's `AGENT_MODEL` override. The chosen model must
support native PDFs; OpenRouter explicitly selects the native parser engine.
No new environment variables or database migrations are required.

The one-shot operation has no tools, conversation history, or database access.
Its prompt classifies the whole document and extracts content into a typed JSON
contract. Rejected, uncertain, unreadable, malformed, incomplete, or insufficient
results create no resume. Classification is probabilistic; prompts and evidence
are not a guarantee against misclassification. Quality should be evaluated on
representative resumes, languages, scans and unrelated documents with the actual
configured model before public rollout.

`ResumeImportService` validates extracted fields and delegates persistence to
`ResumeImportRepository`. Existing entity repositories accept either a pool or
transaction, allowing the import to reuse CRUD inside one transaction. Ownership,
IDs, list positions and the Vertical design are assigned by the server. All rows
commit together, or roll back. Additional sections use custom sections.

There are at most two active imports per API process and one per user per process.
Processing has a 100-second deadline; model output is bounded and truncated
responses are rejected. These are process-local concurrency limits, not a durable
queue or distributed rate limiter. Uploads do not retry automatically. A connection
failure after commit may leave a completed resume in the library, so errors advise
checking the library before retrying.

After success, the editor displays a dismissible review notice and any extraction
warnings. Missing dates stay empty; year-only date labels are preserved in text
because the current resume model stores full dates. Normal automatic sync applies
to subsequent edits.

Protocol references:
- https://platform.claude.com/docs/en/build-with-claude/pdf-support
- https://openrouter.ai/docs/guides/overview/multimodal/pdfs

Tests cover inline provider payloads, document limits, content gates, authenticated
multipart handling, transaction commit/rollback, and client upload errors. Live
provider extraction quality and a live Postgres import require an integration check.
