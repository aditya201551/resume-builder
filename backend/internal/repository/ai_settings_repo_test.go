package repository

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"testing"
)

func TestAIConnectionSelectionRequiresAccountOwnership(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	mock.ExpectQuery(`SELECT .* FROM ai_connections WHERE user_id=\$1 AND provider=\$2`).WithArgs("other-user", "anthropic").WillReturnError(pgx.ErrNoRows)
	repo := NewAISettingsRepository(mock)
	if err = repo.Select(context.Background(), "other-user", "byok", "anthropic"); !errors.Is(err, ErrNotFound) {
		t.Fatal("unowned connection selected", err)
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
func TestDeletingActiveAIKeyRetainsExplicitSource(t *testing.T) {
	for _, fail := range []bool{false, true} {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT id FROM ai_connections WHERE user_id=\$1 AND provider=\$2 FOR UPDATE`).WithArgs("owner", "anthropic").WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow("connection"))
		mock.ExpectExec(`UPDATE ai_preferences SET active_connection_id=NULL,updated_at=now\(\) WHERE user_id=\$1 AND active_connection_id=\$2`).WithArgs("owner", "connection").WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		deleteRow := mock.ExpectExec(`DELETE FROM ai_connections WHERE user_id=\$1 AND id=\$2`).WithArgs("owner", "connection")
		if fail {
			deleteRow.WillReturnError(errors.New("write failed"))
			mock.ExpectRollback()
		} else {
			deleteRow.WillReturnResult(pgxmock.NewResult("DELETE", 1))
			mock.ExpectCommit()
		}
		err = NewAISettingsRepository(mock).Delete(context.Background(), "owner", "anthropic")
		if (err != nil) != fail {
			t.Fatal("deletion did not commit/rollback atomically", err)
		}
		if err = mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
		mock.Close()
	}
}
