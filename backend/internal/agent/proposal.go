package agent

import (
	"fmt"
	"sync"
	"time"
)

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
	DesignPatch     map[string]any `json:"designPatch,omitempty"`
	DesignUpdates   map[string]any `json:"designUpdates,omitempty"`
	ToolCallID      string         `json:"toolCallId,omitempty"`
}

var flatEntities = map[string]bool{
	"work_experiences": true,
	"educations":       true,
	"projects":         true,
	"certifications":   true,
	"languages":        true,
	"misc_entries":     true,
}

type ProposalSink struct {
	mu    sync.Mutex
	items []Proposal
}

func (s *ProposalSink) add(p Proposal) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = append(s.items, p)
}

func (s *ProposalSink) Drain() []Proposal {
	s.mu.Lock()
	defer s.mu.Unlock()
	items := s.items
	s.items = nil
	return items
}

var tempIDCounter int64
var tempIDMu sync.Mutex

func tempID() string {
	tempIDMu.Lock()
	n := tempIDCounter
	tempIDCounter++
	tempIDMu.Unlock()
	return fmt.Sprintf("temp-%d-%d", time.Now().UnixNano(), n)
}
