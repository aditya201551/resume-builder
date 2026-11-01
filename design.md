# Design: Resume Builder

The application uses **Cobalt and ember**, the gradient direction selected by the owner. Resume templates and generated PDFs retain independent user-controlled typography, colours and geometry.

## System

- Scope: sign-in, home, resume library, editor, assistant and job search.
- Signature: cobalt and coral/peach light surrounding a focused, solid AI composer.
- Composition: fixed, non-scrolling home with a viewport-filling gradient. The compact AI composer, starter tasks and recent resumes form a single group, with a bounded 24–40px gap before the recent-resume strip rather than anchoring it to the viewport bottom. The strip shows up to four resumes, View all and Import existing resume. The home resume picker is a compact pill with a separate menu that accommodates long titles. The full document gallery lives on My resumes; manual creation remains available. Editor controls and PDF preview remain stable; jobs separates search from results.
- Navigation: slim, translucent glass top bars with backdrop blur, a neutral selected surface, and Home, My resumes and Jobs. The editor retains AI workspace, Content and Design modes in the same glass treatment.
- The resume assistant has no upper header or scope bar. Its conversation gets the full upper pane, with job context beside auto-accept in a compact bottom composer. Job context expands inside the composer; its fields can scroll on short screens while send/stop remain visible. The composer has a soft neutral surface, no border, a growing message field and a circular up-arrow send button. The job-search assistant retains its own heading and filter controls.
- Sign-in has no navigation bar or outer frame. Its gradient fills the viewport, with introductory text on the left and Google/GitHub sign-in on the right; the two sections stack on phones. Branding sits within the introduction.
- The application is light-only, including when the operating system or a previously saved preference requests dark mode. There is no theme switch or dark palette.
- Home uses black for action buttons, starter icons and keyboard focus, while retaining the cobalt/ember background gradient. Recent resumes use larger document-preview cards with a distinct paper stage, resume title, owner, edit date and Continue editing action. The gallery uses larger columns for one to three items and four columns for four; narrow screens use a horizontal strip with previews retained.
- Geist Variable is bundled locally and supplies app headings, controls and body text.

## Tokens and surfaces

`frontend/src/styles/tokens.css` owns colour, type, spacing, radius, shadow and motion tokens. `frontend/src/index.css` maps these onto Tailwind/shadcn roles. Glass uses the `--color-glass*` and `--shadow-glass` tokens, with an opaque fallback when backdrop blur is unavailable.

- Grounds: `--color-paper`, `--color-paper-2`, `--color-paper-3`.
- Opaque reading surfaces: `--color-cotton`; document surround: `--color-desk`.
- Text and boundaries: `--color-ink*`, `--color-rule*`.
- Interaction: `--color-signal`, `--color-focus`, `--color-signal-soft`.
- Atmosphere: `--color-halo-cobalt`, `--color-halo-ember`, `--color-halo-peach`; restrained workspace echoes use `--color-halo-soft`.
- Geometry: 9px controls, 16px work surfaces, 22px composers, 28px atmospheric stages. Spacing follows the declared 4px scale.
- Gradient concentration is strongest on home and sign-in. Forms, assistant responses, job results and document pages remain readable opaque surfaces.

## Icons and controls

Lucide is the single interface icon family, normally 16–20px with a consistent 1.8 stroke. The existing file mark gains the brand gradient; selected states use cobalt. Icon-only buttons have accessible names. The compact home composer uses a circular up-arrow send button and examples that rotate every 1.5 seconds while empty. Each example fades upward before the next slides in with a short word stagger, using only opacity and transforms. The hint cannot intercept clicks and disappears immediately on typing or choosing a starter. Reduced-motion preferences show a static example, including when the setting changes while the page is open.

App controls use background and text changes for keyboard focus, without focus outlines or rings. Decorative borders around buttons, form fields, cards, navigation and workspace panels are removed at their shared sources. Fields use neutral fills; floating menus and dialogs use elevation. Row dividers, validation feedback and editable PDF section targets retain their functional distinctions. Text inputs, textareas, rich-text editors and composer containers never add a focus border, outline or ring. Touch controls have a 44px hit target. Home actions use black; other AI accents use the app signal colour. Selection indicators do not imply submission.

## Motion

Use CSS for hover, focus, contextual disclosures and surface transitions. Use `motion/react-m` under `LazyMotion` for the editor's selected-mode indicator and success feedback on Copy and Bookmark. Retain dnd-kit for sorting; do not add another drag engine.

- Micro feedback: 120ms; standard state transitions: 180ms; pane transitions: 240ms.
- Home and sign-in gradients play a single 3000ms wave entrance: cobalt and ember/peach layers drift in from opposing directions, pass gently through their resting positions, and settle. The final colours and layering stay the same. Only background opacity and transform animate; content is immediately available and the gradient stays still afterward. Layers are clipped to prevent temporary scrolling and cannot intercept pointer events. Reduced-motion preferences show the final static gradient immediately.
- Loading motion follows actual work. New words in resume replies use the same 280ms, 6px upward fade and 35ms stagger as the home examples. Markdown updates retain the reveal progress of incoming words and leave earlier text still; batch stagger is capped at 120ms so the UI never queues behind the network. Job replies use the same effect on arrival (their endpoint returns a complete reply). Markdown, links, code, copying and cancellation retain their behavior. Reduced-motion preferences reveal all text immediately.
- `MotionConfig reducedMotion="user"` and CSS reduced-motion rules disable spatial motion. Keyboard focus and interactions never wait for an animation.
- Editor panes stay mounted when their mode is hidden, preserving form, chat and draft state.

## Responsive and export boundaries

Below 768px, Home omits recent resumes entirely, including their thumbnail fetches. Its composer is the focus, with examples behind Try an example and manual creation/import below. The main navigation keeps Home, My resumes, Jobs and Account in one roomy row. Home stays fixed; expanded examples/context and short viewports can scroll inside the introductory region. The library uses one column with compact document previews and larger actions. Sign-in removes the secondary feature list and keeps the introduction and OAuth actions spaced apart.

Below 900px, the editor uses a compact title/status row with an accessible export icon, followed by a single AI/Content/Design/Preview selector. The separate controls/preview switch is removed; panes remain mounted to preserve edits and chat drafts. Below 768px, auto-accept moves into Assistant options, inputs use 16px text with larger targets, dialogs respect the viewport, and bottom controls account for safe areas. Jobs combines AI assistant/Filters/Results into one selector below Discover/Saved, with single-column filters and no duplicate mobile assistant header. Clear conversation remains in the job composer. Other pages retain normal scrolling.

Keep PDF canvas, text, link and section-target geometry intact in `workspace.css`. `--color-pdf-paper` remains exact white. App selectors are scoped so gradient, type and motion rules cannot change template content or document dimensions. The client preview and download continue to use the same generated PDF revision; the independent legacy print route is retained.

## Files and verification

- `styles/tokens.css`: shared light and glass design tokens.
- `styles/base.css`: accessibility and browser defaults.
- `styles/workspace.css`: app shell, dashboard, sign-in, editor and assistant surfaces.
- `styles/jobs.css`: search, results and job details.
- `main.tsx`: light appearance and reduced-motion-aware animation providers.

Verify frontend build, lint and existing tests. Review mobile/desktop, light appearance under dark system preferences, home viewport fit with expanded context, glass navigation, neutral text-field focus, loading/empty/error/full states, draft preservation, AI review/apply, search/filter/save, keyboard focus and PDF parity. Do not start development servers without the owner's request.
