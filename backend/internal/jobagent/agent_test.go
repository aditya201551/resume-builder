package jobagent

import (
	"encoding/json"
	"resume-builder/backend/internal/jobsearch"
	"testing"
)

func TestRefinementValidationProtectsSearchContract(t *testing.T) {
	current := jobsearch.Intent{Version: 1, Query: "Go", Branches: []jobsearch.Branch{{Workplace: "remote", Country: "IN"}}}
	raw, _ := json.Marshal(jobsearch.Refinement{Intent: jobsearch.Intent{Version: 1, Query: "React", Branches: []jobsearch.Branch{{Workplace: "remote", Country: "DE"}}}, Message: "Search React roles open to Germany.", Apply: true})
	out, err := decodeRefinement(string(raw), current)
	if err != nil || out.Intent.Branches[0].Country != "DE" {
		t.Fatal(out, err)
	}
	out, err = decodeRefinement(`{"apply":false,"message":"Which country?","intent":{}}`, current)
	if err != nil || out.Intent.Query != "Go" || out.Intent.Branches[0].Country != "IN" {
		t.Fatal("clarification replaced existing criteria", out, err)
	}
	for _, raw := range []string{`{"apply":true,"message":"OK","intent":{"version":1,"branches":[]}}`, `{"apply":true,"message":"OK","intent":{"version":1,"branches":[{"workplace":"remote"}],"salary":100000}}`, `{"apply":true,"message":"OK","intent":{"version":1,"branches":[{"workplace":"remote","country":"invented"}]}}`} {
		if _, err := decodeRefinement(raw, current); err == nil {
			t.Fatal("invalid or unsupported criteria accepted", raw)
		}
	}
}
