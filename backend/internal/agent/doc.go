// Package agent implements the Phase 2 AI assistant (PRD "Requirements —
// P1 (Phase 2: AI-Assisted Content)"): an Eino ChatModelAgent, backed by
// Claude, that reads the client's current resume draft and proposes
// structured edits to it — see chat.go's package comment for the propose/
// accept flow. It runs in-process inside cmd/api for now — Agent.Chat is
// called directly from internal/api/handlers/agent_handler.go, no network
// hop, no MCP server.
//
// Split-out plan: this package deliberately has no dependency on anything
// under cmd/api or internal/api, or even internal/service/internal/auth —
// only Eino/Claude and the client-supplied draft JSON it's handed per
// request. If it later needs to run as its own service, the move is:
//  1. copy this package into a new cmd/agent, with a thin main.go that
//     wires Config + New the same way cmd/api/main.go does today and
//     exposes Chat over HTTP;
//  2. replace the direct Agent.Chat call in agent_handler.go with an HTTP
//     (or MCP, if a second consumer ever needs these tools) call to that
//     service.
//
// Nothing in this package itself has to change for that split — it's
// already written as if it were a standalone service that happens to be
// called in-process.
package agent
