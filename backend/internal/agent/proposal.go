package agent

import (
	"fmt"
	"sync"
	"time"
)

// Proposal mirrors one variant of the frontend's DraftAction union (see
// frontend/src/hooks/resumeDraftReducer.ts). It is deliberately one flexible
// struct rather than one Go type per variant — Type selects which fields are
// meaningful, and the JSON tags match the TS shape exactly so the frontend
// reducer can dispatch it without translation. Fields/Patch stay untyped
// (map[string]any) because the agent produces plain strings for dates etc.,
// same as a human typing into the form — normalization into typed values
// happens where it already happens for manual edits, not here.
type Proposal struct {
	Type            string         `json:"type"`
	Entity          string         `json:"entity,omitempty"`
	ID              string         `json:"id,omitempty"`
	TempID          string         `json:"tempId,omitempty"`
	GroupID         string         `json:"groupId,omitempty"`
	SectionID       string         `json:"sectionId,omitempty"`
	SectionType     string         `json:"sectionType,omitempty"`
	CustomSectionID *string        `json:"customSectionId,omitempty"`
	Fields          map[string]any `json:"fields,omitempty"`
	Patch           map[string]any `json:"patch,omitempty"`
	// DesignPatch is only populated for Type == "design_update" — shaped
	// exactly like the frontend's design_update DraftAction patch (one or
	// more top-level ResumeDesign groups, each the group's full current
	// value with the requested field(s) overridden). See
	// propose_design_update in tools.go for how it's built. This is what
	// gets applied; DesignUpdates below is what gets displayed.
	DesignPatch map[string]any `json:"designPatch,omitempty"`
	// DesignUpdates is only populated for Type == "design_update" — the
	// flat dot-path key/value pairs the model actually requested (e.g.
	// {"colors.accent": "#3457d5"}), separate from DesignPatch because the
	// patch carries every untouched sibling in the same group forward too
	// (required for the reducer's shallow merge) and would otherwise make
	// the review card show fields that didn't change.
	DesignUpdates map[string]any `json:"designUpdates,omitempty"`
	// ToolCallID is the id of the propose_* call that produced this
	// proposal (compose.GetToolCallID, stamped by each propose_* tool
	// before calling sink.add). The "tool" and "proposal" SSE events are
	// otherwise uncorrelated — when the model fires several propose_* calls
	// concurrently, their "done" events and proposal events can each arrive
	// in a different order, so the frontend needs this id to know which
	// tool call a given proposal belongs to rather than guessing from
	// stream position.
	ToolCallID string `json:"toolCallId,omitempty"`
}

// flatEntities are the entity names the propose_create/update/delete tools
// accept — must match frontend FlatKind exactly (resumeDraftReducer.ts).
var flatEntities = map[string]bool{
	"work_experiences": true,
	"educations":       true,
	"projects":         true,
	"certifications":   true,
	"languages":        true,
	"misc_entries":     true,
}

// ProposalSink collects proposals emitted by tool calls during one chat run.
// Tool execution happens synchronously as part of draining the Eino event
// iterator (no separate goroutine touches this), so a mutex here guards
// against nothing but is cheap insurance if that assumption ever changes.
type ProposalSink struct {
	mu    sync.Mutex
	items []Proposal
}

func (s *ProposalSink) add(p Proposal) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = append(s.items, p)
}

// Drain returns every proposal added since the last Drain call and clears
// the buffer. The chat handler calls this after each iterator step so
// proposals stream to the client as soon as the model decides on them.
func (s *ProposalSink) Drain() []Proposal {
	s.mu.Lock()
	defer s.mu.Unlock()
	items := s.items
	s.items = nil
	return items
}

var tempIDCounter int64
var tempIDMu sync.Mutex

// tempID mirrors the shape frontend/src/lib/tempId.ts produces (a "temp-"
// prefix is all resumeDraftSync.ts's isTempId actually checks for).
func tempID() string {
	tempIDMu.Lock()
	n := tempIDCounter
	tempIDCounter++
	tempIDMu.Unlock()
	return fmt.Sprintf("temp-%d-%d", time.Now().UnixNano(), n)
}
