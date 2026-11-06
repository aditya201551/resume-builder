// Package ai owns provider configuration and clients shared by AI-backed features.
// Chat uses Eino; PDFClient adds one-shot inline document requests because the
// current Eino adapter cannot send PDF blocks. Callers supply request-scoped
// configuration resolved from platform environment settings or an encrypted
// account connection. The selected model must support native PDF input for imports.
package ai
