// Package ai owns provider configuration and clients shared by AI-backed features.
// Chat uses Eino; PDFClient adds one-shot inline document requests because the
// current Eino adapter cannot send PDF blocks. Both use AI_PROVIDER, AI_API_KEY
// and AI_MODEL. The selected model must support native PDF input for imports.
package ai
