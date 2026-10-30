package design

import "encoding/json"

func Default(templateID string) ResumeDesign {
	template, ok := FindTemplate(templateID)
	if !ok {
		template, _ = FindTemplate(DefaultTemplateID())
	}
	return ResumeDesign{TemplateID: template.ID, DesignDefaults: template.Defaults}
}

// Decode fills missing saved fields from the selected template, then overlays
// the saved values. Opening a resume never reapplies a formatting preset.
func Decode(raw []byte) (ResumeDesign, error) {
	var selected struct {
		TemplateID string `json:"templateId"`
	}
	if err := json.Unmarshal(raw, &selected); err != nil {
		return ResumeDesign{}, err
	}
	d := Default(selected.TemplateID)
	if err := json.Unmarshal(raw, &d); err != nil {
		return ResumeDesign{}, err
	}
	return NormalizeTemplate(d), nil
}
