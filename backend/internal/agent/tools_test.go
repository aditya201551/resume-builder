package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/cloudwego/eino/components/tool"
)

func TestProposeToolsReportBadArgumentsWithoutFailingTheRun(t *testing.T) {
	cases := []struct {
		name     string
		tool     tool.InvokableTool
		args     string
		contains string
	}{
		{
			name:     "truncated json",
			tool:     newProposeCreateTool(),
			args:     `{"entity":"work_experiences","fields":{"company":"Acme","content":"- did a thing`,
			contains: "could not parse",
		},
		{
			name:     "unknown entity",
			tool:     newProposeCreateTool(),
			args:     `{"entity":"hobbies","fields":{"name":"chess"}}`,
			contains: "unknown entity",
		},
		{
			name:     "unknown field",
			tool:     newProposeCreateTool(),
			args:     `{"entity":"languages","fields":{"name":"English","nonsense":1}}`,
			contains: "nonsense",
		},
		{
			name:     "missing required field",
			tool:     newProposeCreateTool(),
			args:     `{"entity":"work_experiences","fields":{"company":"Acme"}}`,
			contains: "title",
		},
		{
			name:     "update without id",
			tool:     newProposeUpdateTool(),
			args:     `{"entity":"projects","patch":{"name":"X"}}`,
			contains: "id is required",
		},
		{
			name:     "delete without id",
			tool:     newProposeDeleteTool(),
			args:     `{"entity":"projects"}`,
			contains: "id is required",
		},
		{
			name:     "unknown skill action",
			tool:     newProposeSkillChangeTool(),
			args:     `{"action":"frobnicate"}`,
			contains: "unknown action",
		},
		{
			name:     "skill item without group",
			tool:     newProposeSkillChangeTool(),
			args:     `{"action":"create_item","data":{"name":"Go"}}`,
			contains: "group_id is required",
		},
		{
			name:     "custom entry without section",
			tool:     newProposeCustomSectionChangeTool(),
			args:     `{"action":"create_entry","data":{"title":"X"}}`,
			contains: "section_id is required",
		},
		{
			name:     "meta update with unknown field",
			tool:     newProposeMetaUpdateTool(),
			args:     `{"patch":{"favourite_colour":"blue"}}`,
			contains: "favourite_colour",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := tc.tool.InvokableRun(context.Background(), tc.args)
			if err != nil {
				t.Fatalf("InvokableRun returned a Go error, which aborts the whole run: %v", err)
			}
			if !strings.HasPrefix(out, "ERROR:") {
				t.Errorf("result should be flagged as an error to the model, got: %s", out)
			}
			if !strings.Contains(out, tc.contains) {
				t.Errorf("result should mention %q, got: %s", tc.contains, out)
			}
			if !strings.Contains(out, "call the tool again") {
				t.Errorf("result should tell the model to retry, got: %s", out)
			}
		})
	}
}

func TestProposeToolsStillErrorWhenSinkIsMissing(t *testing.T) {
	out, err := newProposeCreateTool().InvokableRun(
		context.Background(),
		`{"entity":"languages","fields":{"name":"English"}}`,
	)
	if err == nil {
		t.Fatalf("expected a real error when no proposal sink is in context, got result: %s", out)
	}
	if !strings.Contains(err.Error(), "sink") {
		t.Errorf("unexpected error: %v", err)
	}
}
