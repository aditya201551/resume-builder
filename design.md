# Design: Resume Builder

The shared system for the application's screens. Resume templates and PDF documents keep their independent user-controlled typography, colours, and geometry.

## System

- Genre: modern-minimal
- Scope: app screens, including sign-in
- Theme: Coral, adapted to a working document editor
- Axes: light / grotesk sans / warm, with restrained colour
- Signature: real resume documents on a writing desk
- Navigation: compact top bar, with Home, My resumes and Job map; the editor keeps its own mode bar
- Composition: dashboard writing prompt beside its introduction; document collection below; stable controls on the left and PDF preview on the right in every editor mode

## Tokens

`frontend/src/styles/tokens.css` is the source of truth. Application semantic aliases live in `frontend/src/index.css`.

- Paper: `--color-paper`, `--color-paper-2`, `--color-paper-3`
- Document surface: `--color-cotton`; preview desk: `--color-desk`
- Text: `--color-ink`, `--color-ink-2`, `--color-ink-muted`
- Dividers: `--color-rule`; input boundaries: `--color-rule-2`
- Coral: `--color-signal`, `--color-focus`, reserved for selection and focus
- Type: locally bundled Geist Variable, for both body and headings
- Radii: 8px surfaces, 6px controls
- Spacing: the declared 4px scale, with 2px optical adjustments
- Dark mode: `:root.dark` redefines surface, text, boundary, and focus tokens
- PDF page white stays independent of the application's dark mode

The Hallmark accent and muted roles are named `--color-signal` and `--color-ink-muted` to avoid collisions with Tailwind/shadcn semantic colours. Geist is retained as the app-screen adaptation so the interface needs no network-loaded font.

## Controls

- Primary: dark ink, 6px radius, typically 40px tall
- Secondary: outlined or text, matching the same geometry
- Touch: 44px minimum button height
- Focus: instant 2px coral outline
- Disabled: distinct fill and text plus a not-allowed cursor
- Forms: visible labels and visible input boundaries

## Motion

Still at rest. Short colour and state transitions only; no decorative entrance animation. Reduced-motion preferences disable movement immediately.

## Exports

- `frontend/src/styles/tokens.css`: colour, type, spacing, radius, and motion tokens
- `frontend/src/styles/base.css`: scoped accessibility and browser defaults
- `frontend/src/styles/workspace.css`: application layout and interaction surfaces
- `frontend/src/styles/jobs.css`: job discovery, map/list workspace and job details
