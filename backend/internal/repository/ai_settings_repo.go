package repository

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"time"
)

type AIConnection struct {
	ID                string     `json:"id"`
	UserID            string     `json:"-"`
	Provider          string     `json:"provider"`
	Model             string     `json:"model"`
	EncryptedKey      []byte     `json:"-"`
	EncryptionVersion string     `json:"-"`
	KeySuffix         string     `json:"key_suffix"`
	CredentialVersion int64      `json:"-"`
	PDFSupported      bool       `json:"pdf_supported"`
	ValidatedAt       *time.Time `json:"validated_at"`
}
type AIPreference struct {
	Source   string `json:"source"`
	Provider string `json:"provider"`
}
type aiSettingsDatabase interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Begin(context.Context) (pgx.Tx, error)
}
type AISettingsRepository struct{ pool aiSettingsDatabase }

func NewAISettingsRepository(pool aiSettingsDatabase) *AISettingsRepository {
	return &AISettingsRepository{pool}
}

const aiColumns = `id,user_id,provider,model,encrypted_key,encryption_version,key_suffix,credential_version,pdf_supported,validated_at`

func scanAI(row pgx.Row) (*AIConnection, error) {
	c := &AIConnection{}
	err := row.Scan(&c.ID, &c.UserID, &c.Provider, &c.Model, &c.EncryptedKey, &c.EncryptionVersion, &c.KeySuffix, &c.CredentialVersion, &c.PDFSupported, &c.ValidatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return c, err
}
func (r *AISettingsRepository) List(ctx context.Context, user string) ([]AIConnection, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+aiColumns+` FROM ai_connections WHERE user_id=$1 ORDER BY provider`, user)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []AIConnection{}
	for rows.Next() {
		c, e := scanAI(rows)
		if e != nil {
			return nil, e
		}
		items = append(items, *c)
	}
	return items, rows.Err()
}
func (r *AISettingsRepository) Get(ctx context.Context, user, provider string) (*AIConnection, error) {
	return scanAI(r.pool.QueryRow(ctx, `SELECT `+aiColumns+` FROM ai_connections WHERE user_id=$1 AND provider=$2`, user, provider))
}
func (r *AISettingsRepository) Save(ctx context.Context, c AIConnection) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO ai_connections(user_id,provider,model,encrypted_key,encryption_version,key_suffix,pdf_supported)
 VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(user_id,provider) DO UPDATE SET
 model=EXCLUDED.model,encrypted_key=EXCLUDED.encrypted_key,encryption_version=EXCLUDED.encryption_version,
 key_suffix=EXCLUDED.key_suffix,pdf_supported=EXCLUDED.pdf_supported,validated_at=NULL,
 credential_version=ai_connections.credential_version+1,updated_at=now()`, c.UserID, c.Provider, c.Model, c.EncryptedKey, c.EncryptionVersion, c.KeySuffix, c.PDFSupported)
	return err
}
func (r *AISettingsRepository) Validate(ctx context.Context, user, provider string, version int64) error {
	tag, err := r.pool.Exec(ctx, `UPDATE ai_connections SET validated_at=now() WHERE user_id=$1 AND provider=$2 AND credential_version=$3`, user, provider, version)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}
func (r *AISettingsRepository) Delete(ctx context.Context, user, provider string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var id string
	err = tx.QueryRow(ctx, `SELECT id FROM ai_connections WHERE user_id=$1 AND provider=$2 FOR UPDATE`, user, provider).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE ai_preferences SET active_connection_id=NULL,updated_at=now() WHERE user_id=$1 AND active_connection_id=$2`, user, id); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM ai_connections WHERE user_id=$1 AND id=$2`, user, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (r *AISettingsRepository) Preference(ctx context.Context, user string) (*AIPreference, error) {
	p := &AIPreference{}
	err := r.pool.QueryRow(ctx, `SELECT p.source,COALESCE(c.provider,'') FROM ai_preferences p LEFT JOIN ai_connections c ON c.user_id=p.user_id AND c.id=p.active_connection_id WHERE p.user_id=$1`, user).Scan(&p.Source, &p.Provider)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return p, err
}
func (r *AISettingsRepository) Select(ctx context.Context, user, source, provider string) error {
	var id *string
	if source == "byok" {
		c, err := r.Get(ctx, user, provider)
		if err != nil {
			return err
		}
		id = &c.ID
	}
	_, err := r.pool.Exec(ctx, `INSERT INTO ai_preferences(user_id,source,active_connection_id) VALUES($1,$2,$3)
 ON CONFLICT(user_id) DO UPDATE SET source=EXCLUDED.source,active_connection_id=EXCLUDED.active_connection_id,updated_at=now()`, user, source, id)
	return err
}
func (r *AISettingsRepository) Allowed(ctx context.Context, user string) (bool, error) {
	var allowed bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM platform_ai_allowlist WHERE user_id=$1)`, user).Scan(&allowed)
	return allowed, err
}
