# Interactive resume assistant

OpenUI is a presentation layer inside the existing chat. The first catalog offers
a guided form for **new work experience** and editable **rewrite comparisons** for
the summary, headline, work descriptions and project descriptions. The chat starter
buttons include Add work experience and Write my summary. Ordinary prose and all
existing manual editing, automatic sync, design and PDF behavior remain supported.

The assistant defaults to normal mutation tools when the request is clear and
required facts are available. It asks only for blocking missing facts or ambiguous
targets, preserving details already supplied in the conversation. For example,
"Google, Germany, Jan 2026 to present" needs only a job-title question before the
entry can be added; missing responsibilities or metrics do not require a form.
Cards are reserved for explicitly requested guided input or alternatives, or
unresolved questions that benefit from structured input or comparison. Ordinary
summary rewrites and the default starter buttons use the normal proposal flow.
Complete new entries from `create_resume_entry` are validated by the backend and
marked `applyDirectly`. The client adds them immediately and shows a collapsed
“Added” result, including when automatic acceptance of other edits is off. Missing
required facts block creation; missing optional descriptions or metrics stay empty.
Updates and removals retain the existing review and automatic acceptance settings.
Retrying pending dependencies never bypasses those settings.

When a form is needed, prefill all supplied or unambiguously implied facts, including
normalized month dates and current-employment status. Leave only unknown fields
empty. Never infer a new role from an earlier job or invent responsibilities or metrics.

## Request and action flow

1. The Go/Eino agent calls `show_resume_ui` with a bounded OpenUI Lang program.
   Its tool description includes the generated component catalog and syntax rules.
2. The backend validates the card kind and resolves any rewrite target from the
   current draft. Original text is supplied by the backend, not by the UI program.
3. A separate SSE `ui` event sends a complete, version-1 card to the client. This
   first integration delivers cards after tool arguments are complete; it does not
   render partial tool JSON. Ordinary assistant text continues to stream normally.
4. The lazy-loaded renderer displays only our registered components, styled with
   existing shadcn controls and application tokens. Unsupported output has a fallback.
5. Review experience / Review this version builds a normal **pending** proposal.
   Opening a card, typing, and selecting Review do not mutate the resume. UI-created
   proposals require review even if automatic acceptance is enabled for agent tools.
6. Accept change goes through the existing proposal validator, reducer and sync.
   A comparison's source is checked both when selected and when accepted. Modified
   or deleted sources block stale rewrites. Temporary IDs resolve through the same
   identity map as other proposals. Reject lets the user choose/review again.

No OpenUI toolProvider, MCP client, arbitrary callbacks, generated HTML, or network
actions are exposed. Component callbacks use a small typed action bridge that can
only create a new work entry or propose the bound text change. Backend ownership
checks still apply to every eventual API write. At most two cards are added per turn.

## Catalog and prompt maintenance

- `frontend/src/components/agent/openui/library.tsx`: schemas and components.
- `frontend/src/lib/assistantUI.ts`: envelope validation and action-to-proposal mapping.
- `backend/internal/agent/ui.go`: model tool, target binding and versioned event.
- `backend/internal/agent/openui_prompt.txt` and `.spec.json`: generated artifacts.

After catalog changes run, from the repo root:

```sh
npm run generate:openui --workspace frontend
```

Commit both generated files. The Go binary embeds the text, so production needs no
Node server or OpenUI cloud service. A frontend test checks that the embedded prompt
matches the catalog. Regenerate before restarting/rebuilding the API.

The current pins are `@openuidev/react-lang` 1.0.2 and Zod 4.6.7. Zod is also pinned
at workspace root so npm resolves OpenUI and application schemas to the same v4
implementation. The older v4 compatibility layer in Zod 3 lacks runtime functions
used by OpenUI 1.0.2. Keep the dependency alignment when upgrading.

Installation telemetry is disabled in the Docker build and was disabled during
installation here. The generator uses `--no-telemetry`; renderer observability is
disabled. Local installs can use `OPENUI_TELEMETRY_DISABLED=1`. No Thesys API key,
Gateway, Autofix service or separate OpenUI inference call is required; cards use
the existing agent tool-call loop.

## Scope and validation

Cards and edited form state live in current chat memory, like existing messages.
They survive ordinary rerenders and mode switches, but not a page reload. Persistent
chat storage and additional card types are separate work.

Tests cover event validation, target binding, generated prompt alignment, dates,
proposal construction, stale-source protection, and separation from mutations.
Production browser smoke checks cover form/rewrite review, one-time sync, error
fallback and mobile sizing with mocked AI/API responses. Live generation quality
still needs evaluation with the configured model and representative user requests.

References: [OpenUI](https://github.com/thesysdev/openui),
[component libraries](https://www.openui.com/docs/openui-lang/defining-components),
[prompt generation](https://www.openui.com/docs/openui-lang/system-prompts).
