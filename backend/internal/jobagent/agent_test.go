package jobagent

import (
	"context"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"resume-builder/backend/internal/jobsearch"
	"testing"
)

func TestRefinementChangesPreserveManualCriteriaAndOwnVersion(t *testing.T) {
	current := jobsearch.Intent{Version: 1, Query: "Go", Branches: []jobsearch.Branch{{Workplace: "remote", Country: "IN"}}, Seniority: "senior", ExcludeCompanies: []string{"Acme"}, IncludeUnknown: true}
	out, err := decodeRefinement(`{"apply":true,"message":"Search React roles.","changes":{"query":"React"}}`, current)
	if err != nil || out.Intent.Version != 1 || out.Intent.Query != "React" || out.Intent.Branches[0].Country != "IN" || out.Intent.Seniority != "senior" || out.Intent.ExcludeCompanies[0] != "Acme" {
		t.Fatal("partial change overwrote manual criteria", out, err)
	}
	out, err = decodeRefinement(`{"apply":true,"message":"Clear exclusions.","changes":{"excludeCompanies":[],"includeUnknown":false,"seniority":""}}`, current)
	if err != nil || len(out.Intent.ExcludeCompanies) != 0 || out.Intent.IncludeUnknown || out.Intent.Seniority != "" {
		t.Fatal("explicit clears were lost", out, err)
	}
	if len(current.ExcludeCompanies) != 1 || current.Seniority != "senior" {
		t.Fatal("input criteria were mutated")
	}
	out, err = decodeRefinement(`{"apply":false,"message":"Which country?"}`, current)
	if err != nil || out.Intent.Query != "Go" || out.Intent.Branches[0].Country != "IN" {
		t.Fatal("clarification replaced existing criteria", out, err)
	}
	for _, raw := range []string{
		`{"apply":true,"message":"OK","changes":{"branches":[]}}`,
		`{"apply":true,"message":"OK","changes":{"salary":100000}}`,
		`{"apply":true,"message":"OK","changes":{"version":0}}`,
		`{"apply":true,"message":"OK","changes":{"branches":[{"workplace":"remote","country":"invented"}]}}`,
		`{"apply":true,"message":"OK","changes":{"includeUnknown":"true"}}`,
		`{"apply":true,"message":"OK","changes":{}} {}`,
	} {
		if _, err := decodeRefinement(raw, current); err == nil {
			t.Fatal("invalid criteria accepted", raw)
		}
	}
}

type planningModel struct {
	replies []string
	calls   int
}

func (m *planningModel) Generate(context.Context, []*schema.Message, ...model.Option) (*schema.Message, error) {
	reply := m.replies[min(m.calls, len(m.replies)-1)]
	m.calls++
	return &schema.Message{Role: schema.Assistant, ToolCalls: []schema.ToolCall{{Function: schema.FunctionCall{Name: "plan_job_search", Arguments: reply}}}}, nil
}
func (m *planningModel) Stream(context.Context, []*schema.Message, ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	panic("unused streaming path")
}
func (m *planningModel) WithTools([]*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	return m, nil
}

func TestInvalidPlanIsRepairedOnceWithoutGeneratingSearchVersion(t *testing.T) {
	m := &planningModel{replies: []string{`{"apply":true,"message":"Search","changes":{"version":0}}`, `{"apply":true,"message":"Search React","changes":{"query":"React"}}`}}
	a := &Agent{model: m, gate: make(chan struct{}, 1)}
	current := jobsearch.Intent{Version: 1, Query: "Go", Branches: []jobsearch.Branch{{Workplace: "remote", Country: "IN"}}}
	out, err := a.Refine(context.Background(), current, "Find React roles", nil)
	if err != nil || m.calls != 2 || !out.Apply || out.Intent.Version != 1 || out.Intent.Query != "React" || out.Intent.Branches[0].Country != "IN" {
		t.Fatal(out, err, m.calls)
	}
}

func TestRepeatedMalformedPlansKeepSearchUnchanged(t *testing.T) {
	m := &planningModel{replies: []string{`{"apply":true,"message":"Search","changes":{"salary":100000}}`}}
	a := &Agent{model: m, gate: make(chan struct{}, 1)}
	current := jobsearch.Intent{Version: 1, Query: "Go", Branches: []jobsearch.Branch{{Workplace: "remote", Country: "IN"}}}
	out, err := a.Refine(context.Background(), current, "Search", nil)
	if err != nil || m.calls != 2 || out.Apply || out.Intent.Query != "Go" || out.Intent.Branches[0].Country != "IN" || out.Message == "" {
		t.Fatal(out, err, m.calls)
	}
}
