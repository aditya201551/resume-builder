# Design: Resume Builder

The application uses **Cobalt and ember**, the gradient direction selected by the owner. Resume templates and generated PDFs retain independent user-controlled typography, colours and geometry.

## System

- Scope: sign-in, home, resume library, editor, assistant and job search.
- Signature: cobalt and coral/peach light surrounding a focused, solid AI composer.
- Composition: fixed, non-scrolling home with a viewport-filling gradient. The compact AI composer, starter tasks and recent resumes form a single group, with a bounded 24–40px gap before the recent-resume strip rather than anchoring it to the viewport bottom. The strip shows up to four resumes, View all and Import existing resume. The home resume picker is a compact pill with a separate menu that accommodates long titles. The full document gallery lives on My resumes; manual creation remains available. Editor controls and PDF preview remain stable; jobs separates search from results.
- Navigation: slim, translucent glass top bars with backdrop blur, a neutral selected surface, and Home, My resumes and Jobs. The editor retains AI workspace, Content and Design modes in the same glass treatment.
- Sign-in has no navigation bar or outer frame. Its gradient fills the viewport, with introductory text on the left and Google/GitHub sign-in on the right; the two sections stack on phones. Branding sits within the introduction.
- The application is light-only, including when the operating system or a previously saved preference requests dark mode. There is no theme switch or dark palette.
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

Lucide is the single interface icon family, normally 16–20px with a consistent 1.8 stroke. The existing file mark gains the brand gradient; selected states use cobalt. Icon-only buttons have accessible names. The compact home composer uses a circular up-arrow send button and examples that rotate every five seconds while empty, with a static example for reduced-motion preferences.

Buttons retain an immediate 2px focus outline. As requested by the owner, text inputs, textareas, rich-text editors and their composer containers do not add a border, outline or ring when focused, including keyboard focus. Their resting surfaces and validation states stay unchanged. Controls use distinct hover, pressed, disabled and error states; touch controls have a 44px hit target. Primary actions use the ink treatment, with the AI entry action using cobalt. Selection indicators do not imply submission.

## Motion

Use CSS for hover, focus, contextual disclosures and surface transitions. Use `motion/react-m` under `LazyMotion` for the editor's selected-mode indicator and success feedback on Copy and Bookmark. Retain dnd-kit for sorting; do not add another drag engine.

- Micro feedback: 120ms; standard state transitions: 180ms; pane transitions: 240ms.
- Home and sign-in gradients play a single 3000ms wave entrance: cobalt and ember/peach layers drift in from opposing directions, pass gently through their resting positions, and settle. The final colours and layering stay the same. Only background opacity and transform animate; content is immediately available and the gradient stays still afterward. Layers are clipped to prevent temporary scrolling and cannot intercept pointer events. Reduced-motion preferences show the final static gradient immediately.
- Loading motion follows actual work. Streaming messages are not reanimated for every text chunk.
- `MotionConfig reducedMotion="user"` and CSS reduced-motion rules disable spatial motion. Keyboard focus and interactions never wait for an animation.
- Editor panes stay mounted when their mode is hidden, preserving form, chat and draft state.

## Responsive and export boundaries

On phones, keep explicit controls/preview and controls/results switches. Do not squeeze two desktop panes into narrow columns. The home navigation stays in one compact row; starter tasks fit a compact grid and recent resumes form a compact horizontal strip. Home stays fixed; long text and expanded job context can scroll within the composer, and the introductory controls have a bounded scrolling region on short screens so the recent-resume actions remain accessible. Other pages retain normal scrolling and document galleries reduce columns.

Keep PDF canvas, text, link and section-target geometry intact in `workspace.css`. `--color-pdf-paper` remains exact white. App selectors are scoped so gradient, type and motion rules cannot change template content or document dimensions. The client preview and download continue to use the same generated PDF revision; the independent legacy print route is retained.

## Files and verification

- `styles/tokens.css`: shared light and glass design tokens.
- `styles/base.css`: accessibility and browser defaults.
- `styles/workspace.css`: app shell, dashboard, sign-in, editor and assistant surfaces.
- `styles/jobs.css`: search, results and job details.
- `main.tsx`: light appearance and reduced-motion-aware animation providers.

Verify frontend build, lint and existing tests. Review mobile/desktop, light appearance under dark system preferences, home viewport fit with expanded context, glass navigation, neutral text-field focus, loading/empty/error/full states, draft preservation, AI review/apply, search/filter/save, keyboard focus and PDF parity. Do not start development servers without the owner's request.
