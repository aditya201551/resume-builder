package repository

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
)

func TestResumeImportTransaction(t *testing.T) {
	for _, fail := range []bool{false, true} {
		name := "commit"
		if fail {
			name = "rollback child failure"
		}
		t.Run(name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Fatal(err)
			}
			defer mock.Close()
			mock.ExpectBegin()
			now := time.Now()
			mock.ExpectQuery("INSERT INTO resumes").WithArgs("user", "Imported", "Anika", (*string)(nil), (*string)(nil), (*string)(nil), (*string)(nil), (*string)(nil), (*string)(nil), json.RawMessage(`[]`)).WillReturnRows(pgxmock.NewRows(strings.Split(resumeColumns, ", ")).AddRow("resume-id", "user", "Imported", "Anika", nil, nil, nil, nil, nil, nil, []byte(`[]`), now, now, nil))
			if fail {
				mock.ExpectQuery("INSERT INTO educations").WithArgs("resume-id", "University", (*string)(nil), (*string)(nil), (*string)(nil), (*time.Time)(nil), (*time.Time)(nil), (*string)(nil), (*string)(nil), 0, (*string)(nil)).WillReturnError(errors.New("database write failed"))
				mock.ExpectRollback()
			} else {
				mock.ExpectQuery("INSERT INTO educations").WithArgs("resume-id", "University", (*string)(nil), (*string)(nil), (*string)(nil), (*time.Time)(nil), (*time.Time)(nil), (*string)(nil), (*string)(nil), 0, (*string)(nil)).WillReturnRows(pgxmock.NewRows(strings.Split(educationColumns, ", ")).AddRow("education-id", "resume-id", "University", nil, nil, nil, nil, nil, nil, nil, 0, now, now))
				mock.ExpectQuery("INSERT INTO resume_designs").WithArgs("resume-id", pgxmock.AnyArg()).WillReturnRows(pgxmock.NewRows([]string{"resume_id", "design"}).AddRow("resume-id", []byte(`{"templateId":"vertical"}`)))
				mock.ExpectCommit()
			}
			repo := &ResumeImportRepository{pool: mock}
			result, err := repo.Create(context.Background(), "user", ImportedResume{Resume: ResumeMetaInput{Label: "Imported", FullName: "Anika"}, Educations: []EducationInput{{Institution: "University"}}})
			if fail && err == nil {
				t.Fatal("child failure was ignored")
			}
			if !fail && (err != nil || result.ID != "resume-id") {
				t.Fatalf("commit failed: %v", err)
			}
			if err = mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
