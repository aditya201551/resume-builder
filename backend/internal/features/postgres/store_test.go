package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

func TestFeatureUpdatesRequireExistingFlagAndWriteAuditAtomically(t *testing.T) {
	for _, affected := range []int64{0, 1} {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		mock.ExpectExec("WITH changed AS .*UPDATE feature_flags.*INSERT INTO feature_flag_audit").
			WithArgs("jobs.enabled", true, "owner").WillReturnResult(pgxmock.NewResult("INSERT", affected))
		err = NewStore(mock).Set(context.Background(), "jobs.enabled", true, "owner")
		if affected == 0 && !errors.Is(err, ErrNotFound) {
			t.Fatal("missing flag was silently created")
		}
		if affected == 1 && err != nil {
			t.Fatal(err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
		mock.Close()
	}
}
