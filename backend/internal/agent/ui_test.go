package agent

import (
	"strings"
	"testing"
)

const testUIDraft = `{"resume":{"id":"r1","summary":"Original summary","headline":"Engineer"},"work_experiences":[{"id":"w1","content":"Built reliable APIs"}],"projects":[{"id":"p1","content":"Campus project"}]}`

func TestPrepareResumeUIBindsSourceFromCurrentDraft(t *testing.T) {
	for _, tc := range []struct{ entity, id, field, original string }{
		{"resume", "r1", "summary", "Original summary"},
		{"resume", "r1", "headline", "Engineer"},
		{"work_experiences", "w1", "content", "Built reliable APIs"},
		{"projects", "p1", "content", "Campus project"},
	} {
		ui, err := prepareResumeUI(showUIArgs{Kind: "rewrite_comparison", Code: `root = AssistantPanel("Compare", [choices])`, Fallback: "Compare rewrites", Entity: tc.entity, EntryID: tc.id, Field: tc.field}, testUIDraft)
		if err != nil {
			t.Fatal(err)
		}
		if ui.Version != 1 || ui.Target == nil || ui.Target.ID != tc.id || ui.Target.Original != tc.original {
			t.Fatalf("wrong source: %+v", ui)
		}
	}
}

func TestPrepareResumeUIRejectsUnsupportedTargets(t *testing.T) {
	for _, tc := range []showUIArgs{
		{Kind: "rewrite_comparison", Entity: "resume", EntryID: "r2", Field: "summary"},
		{Kind: "rewrite_comparison", Entity: "work_experiences", EntryID: "missing", Field: "content"},
		{Kind: "rewrite_comparison", Entity: "resume", EntryID: "r1", Field: "user_id"},
		{Kind: "rewrite_comparison", Entity: "projects", EntryID: "p1", Field: "summary"},
		{Kind: "delete_resume"},
	} {
		tc.Code = `root = AssistantPanel("Review", [])`
		tc.Fallback = "Review resume"
		if _, err := prepareResumeUI(tc, testUIDraft); err == nil {
			t.Fatalf("accepted invalid target: %+v", tc)
		}
	}
}

func TestPrepareResumeUIEnforcesBoundsAndDoesNotCreateProposals(t *testing.T) {
	valid := showUIArgs{Kind: "experience_form", Code: `root = AssistantPanel("New experience", [entry])`, Fallback: "Add experience"}
	ui, err := prepareResumeUI(valid, testUIDraft)
	if err != nil || ui.Target != nil {
		t.Fatal("form should not bind an existing entry")
	}
	for _, code := range []string{"", strings.Repeat("x", 16001), "```root = Card()```"} {
		args := valid
		args.Code = code
		if _, err = prepareResumeUI(args, testUIDraft); err == nil {
			t.Fatal("invalid code accepted")
		}
	}
	sink := &ProposalSink{}
	if !sink.addUI(ui) || !sink.addUI(ui) {
		t.Fatal("valid cards not added")
	}
	if len(sink.DrainUI()) != 2 || sink.addUI(ui) {
		t.Fatal("per-turn card limit bypassed after drain")
	}
	if len(sink.Drain()) != 0 {
		t.Fatal("showing a form must not propose a resume mutation")
	}
}

func TestEmbeddedOpenUICatalog(t *testing.T) {
	for _, signature := range []string{"AssistantPanel(", "ExperienceForm(", "RewriteComparison(", "TextBlock("} {
		if !strings.Contains(openUIPrompt, signature) {
			t.Fatalf("missing catalog signature: %s", signature)
		}
	}
}
