package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")

type User struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	AvatarURL     string `json:"avatar_url"`
}

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) FindByIdentity(ctx context.Context, provider, providerUserID string) (*User, error) {
	const q = `
		SELECT u.id, u.email, u.email_verified, u.name, u.avatar_url
		FROM users u
		JOIN auth_identities ai ON ai.user_id = u.id
		WHERE ai.provider = $1 AND ai.provider_user_id = $2`

	var u User
	err := r.pool.QueryRow(ctx, q, provider, providerUserID).
		Scan(&u.ID, &u.Email, &u.EmailVerified, &u.Name, &u.AvatarURL)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find by identity: %w", err)
	}
	return &u, nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
	const q = `SELECT id, email, email_verified, name, avatar_url FROM users WHERE email = $1`

	var u User
	err := r.pool.QueryRow(ctx, q, email).Scan(&u.ID, &u.Email, &u.EmailVerified, &u.Name, &u.AvatarURL)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find by email: %w", err)
	}
	return &u, nil
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (*User, error) {
	const q = `SELECT id, email, email_verified, name, avatar_url FROM users WHERE id = $1`

	var u User
	err := r.pool.QueryRow(ctx, q, id).Scan(&u.ID, &u.Email, &u.EmailVerified, &u.Name, &u.AvatarURL)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find by id: %w", err)
	}
	return &u, nil
}

func (r *UserRepository) CreateWithIdentity(ctx context.Context, u User, provider, providerUserID, providerEmail string) (*User, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	const insertUser = `
		INSERT INTO users (email, email_verified, name, avatar_url)
		VALUES ($1, $2, $3, $4)
		RETURNING id`

	var userID string
	if err := tx.QueryRow(ctx, insertUser, u.Email, u.EmailVerified, u.Name, u.AvatarURL).Scan(&userID); err != nil {
		return nil, fmt.Errorf("insert user: %w", err)
	}

	if err := insertIdentity(ctx, tx, userID, provider, providerUserID, providerEmail); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	u.ID = userID
	return &u, nil
}

func (r *UserRepository) LinkIdentity(ctx context.Context, userID, provider, providerUserID, providerEmail string) error {
	return insertIdentity(ctx, r.pool, userID, provider, providerUserID, providerEmail)
}

type querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func insertIdentity(ctx context.Context, q querier, userID, provider, providerUserID, providerEmail string) error {
	const insertIdentitySQL = `
		INSERT INTO auth_identities (user_id, provider, provider_user_id, provider_email)
		VALUES ($1, $2, $3, $4)`

	if _, err := q.Exec(ctx, insertIdentitySQL, userID, provider, providerUserID, providerEmail); err != nil {
		return fmt.Errorf("insert identity: %w", err)
	}
	return nil
}
