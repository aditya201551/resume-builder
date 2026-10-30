package service

import (
	"encoding/json"
	"net/url"
	"reflect"
	"strings"

	"resume-builder/backend/internal/repository"
)

func validateImportedContent(in *repository.ImportedResume) error {
	invalid := func() error {
		return importError("invalid_extraction", "Some resume sections couldn't be extracted reliably. Please try another PDF.")
	}
	if !boundedImportValue(reflect.ValueOf(*in)) {
		return invalid()
	}
	count := len(in.WorkExperiences) + len(in.Educations) + len(in.Projects) + len(in.Certifications) + len(in.MiscEntries) + len(in.Languages)
	for _, e := range in.WorkExperiences {
		if strings.TrimSpace(e.Company) == "" && strings.TrimSpace(e.Title) == "" {
			return invalid()
		}
	}
	for _, e := range in.Educations {
		if strings.TrimSpace(e.Institution) == "" {
			return invalid()
		}
	}
	for _, e := range in.Projects {
		if strings.TrimSpace(e.Name) == "" {
			return invalid()
		}
	}
	for _, e := range in.Certifications {
		if strings.TrimSpace(e.Name) == "" {
			return invalid()
		}
	}
	for _, e := range in.Languages {
		if strings.TrimSpace(e.Name) == "" {
			return invalid()
		}
	}
	for _, e := range in.MiscEntries {
		if strings.TrimSpace(e.Title) == "" || (e.Kind != "award" && e.Kind != "publication" && e.Kind != "volunteer") {
			return invalid()
		}
	}
	for _, g := range in.SkillGroups {
		if strings.TrimSpace(g.Name) == "" || len(g.Items) == 0 {
			return invalid()
		}
		for _, e := range g.Items {
			if strings.TrimSpace(e.Name) == "" {
				return invalid()
			}
		}
		count += len(g.Items)
	}
	for _, s := range in.CustomSections {
		if strings.TrimSpace(s.Title) == "" || len(s.Entries) == 0 {
			return invalid()
		}
		for _, e := range s.Entries {
			if (e.Title == nil || strings.TrimSpace(*e.Title) == "") && (e.Description == nil || strings.TrimSpace(*e.Description) == "") {
				return invalid()
			}
		}
		count += len(s.Entries)
	}
	career := len(in.WorkExperiences) + len(in.Educations) + len(in.Projects) + len(in.Certifications) + len(in.MiscEntries) + len(in.CustomSections)
	// Sparse/student resumes are valid, but a name or generic skills list alone
	// must not create an otherwise empty resume, even if the classifier says yes.
	if count == 0 || count > 300 || (career == 0 && (len(in.SkillGroups) == 0 || in.Resume.Summary == nil || len(strings.TrimSpace(*in.Resume.Summary)) < 20)) {
		return importError("uncertain_document", "We couldn't find enough resume content to import. Upload a resume or CV PDF.")
	}
	var links []struct {
		Label string `json:"label"`
		URL   string `json:"url"`
	}
	if len(in.Resume.Links) > 0 && json.Unmarshal(in.Resume.Links, &links) != nil {
		return invalid()
	}
	safeLinks := links[:0]
	for _, link := range links {
		if safeImportedURL(link.URL) {
			safeLinks = append(safeLinks, link)
		}
	}
	if safeLinks == nil {
		in.Resume.Links = json.RawMessage(`[]`)
	} else {
		in.Resume.Links, _ = json.Marshal(safeLinks)
	}
	cleanURL := func(p **string) {
		if *p != nil && !safeImportedURL(**p) {
			*p = nil
		}
	}
	for i := range in.WorkExperiences {
		cleanURL(&in.WorkExperiences[i].CompanyURL)
		if !validImportedTechnologies(&in.WorkExperiences[i].Technologies) {
			return invalid()
		}
		if in.WorkExperiences[i].IsCurrent {
			in.WorkExperiences[i].EndDate = nil
		}
	}
	for i := range in.Projects {
		cleanURL(&in.Projects[i].URL)
		if !validImportedTechnologies(&in.Projects[i].Technologies) {
			return invalid()
		}
	}
	for i := range in.Certifications {
		cleanURL(&in.Certifications[i].CredentialURL)
	}
	for i := range in.MiscEntries {
		cleanURL(&in.MiscEntries[i].URL)
	}
	return nil
}

func safeImportedURL(value string) bool {
	u, err := url.Parse(value)
	return err == nil && (u.Scheme == "https" || u.Scheme == "http") && u.Host != "" && u.User == nil
}

func validImportedTechnologies(raw *json.RawMessage) bool {
	if len(*raw) == 0 || string(*raw) == "null" {
		*raw = json.RawMessage(`[]`)
		return true
	}
	var names []string
	return json.Unmarshal(*raw, &names) == nil && len(names) <= 100
}

func boundedImportValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		return len(v.String()) <= 16000 && !strings.ContainsRune(v.String(), 0)
	case reflect.Pointer:
		return v.IsNil() || boundedImportValue(v.Elem())
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if !boundedImportValue(v.Field(i)) {
				return false
			}
		}
	case reflect.Slice:
		// Raw JSON is validated by its field-specific parser above.
		if v.Type() == reflect.TypeFor[json.RawMessage]() {
			return v.Len() <= 32000
		}
		if v.Len() > 100 {
			return false
		}
		for i := 0; i < v.Len(); i++ {
			if !boundedImportValue(v.Index(i)) {
				return false
			}
		}
	}
	return true
}
