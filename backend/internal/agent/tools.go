package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"resume-builder/backend/internal/auth"
	"resume-builder/backend/internal/service"
)

// getFullResumeTool lets the agent read a resume's full content before
// drafting a rewrite or summary. It wraps ResumeService.GetFullResume
// directly — no MCP, no network hop, since the agent runs in the same
// process as the rest of the API (see doc.go). If this package is ever
// split into its own service, this is the tool that would move behind an
// MCP server or HTTP call instead of a direct method call.
type getFullResumeTool struct {
	resumes *service.ResumeService
}

func newGetFullResumeTool(resumes *service.ResumeService) tool.InvokableTool {
	return &getFullResumeTool{resumes: resumes}
}

func (t *getFullResumeTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "get_full_resume",
		Desc: "Fetch every section of a resume (work experience, education, skills, projects, certifications, languages, misc entries, custom sections) by resume ID.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"resume_id": {
				Type:     schema.String,
				Desc:     "UUID of the resume to read",
				Required: true,
			},
		}),
	}, nil
}

type getFullResumeArgs struct {
	ResumeID string `json:"resume_id"`
}

func (t *getFullResumeTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args getFullResumeArgs
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("parse get_full_resume arguments: %w", err)
	}

	userID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return "", fmt.Errorf("no authenticated user in context")
	}

	full, err := t.resumes.GetFullResume(ctx, userID, args.ResumeID)
	if err != nil {
		return "", err
	}

	out, err := json.Marshal(full)
	if err != nil {
		return "", fmt.Errorf("marshal full resume: %w", err)
	}
	return string(out), nil
}
