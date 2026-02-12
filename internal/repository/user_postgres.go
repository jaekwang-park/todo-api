package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jaekwang-park/todo-api/internal/model"
)

type PostgresUserRepository struct {
	db *sql.DB
}

func NewPostgresUser(db *sql.DB) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

func (r *PostgresUserRepository) GetOrCreate(ctx context.Context, cognitoSub, email string) (model.User, error) {
	query := `
		INSERT INTO users (cognito_sub, email)
		VALUES ($1, $2)
		ON CONFLICT (cognito_sub) DO UPDATE SET email = EXCLUDED.email
		RETURNING id, cognito_sub, email, nickname, profile_image_url, created_at, updated_at`

	row := r.db.QueryRowContext(ctx, query, cognitoSub, email)
	return scanUser(row)
}

func (r *PostgresUserRepository) GetByCognitoSub(ctx context.Context, cognitoSub string) (model.User, error) {
	query := `
		SELECT id, cognito_sub, email, nickname, profile_image_url, created_at, updated_at
		FROM users
		WHERE cognito_sub = $1`

	row := r.db.QueryRowContext(ctx, query, cognitoSub)
	return scanUser(row)
}

func (r *PostgresUserRepository) Update(ctx context.Context, user model.User) (model.User, error) {
	query := `
		UPDATE users
		SET nickname = $1, profile_image_url = $2, updated_at = now()
		WHERE id = $3
		RETURNING id, cognito_sub, email, nickname, profile_image_url, created_at, updated_at`

	row := r.db.QueryRowContext(ctx, query, user.Nickname, user.ProfileImageURL, user.ID)
	return scanUser(row)
}

func scanUser(row scannable) (model.User, error) {
	var u model.User
	err := row.Scan(
		&u.ID, &u.CognitoSub, &u.Email, &u.Nickname,
		&u.ProfileImageURL, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return model.User{}, fmt.Errorf("failed to scan user: %w", err)
	}
	return u, nil
}

var _ UserRepository = (*PostgresUserRepository)(nil)
