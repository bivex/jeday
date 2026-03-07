package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jeday/auth/internal/db"
)

const createUsersBulk = `
WITH input AS (
	SELECT email, username, ord::int
	FROM unnest($1::text[], $2::text[]) WITH ORDINALITY AS t(email, username, ord)
), inserted AS (
	INSERT INTO users (email, username)
	SELECT email, username
	FROM input
	ORDER BY ord
	RETURNING id, email, username, status, created_at, updated_at
)
SELECT inserted.id, inserted.email, inserted.username, inserted.status, inserted.created_at, inserted.updated_at, input.ord
FROM inserted
JOIN input
	ON inserted.email = input.email AND inserted.username = input.username
ORDER BY input.ord
`

// Repository defines all functions to execute db queries and transactions
type Repository interface {
	db.Querier
	ExecTx(ctx context.Context, fn func(*db.Queries) error) error
	CreateUsersBatch(ctx context.Context, users []db.CreateUserParams, passwords []string) ([]db.User, error)
}

// PostgresRepository provides all functions to execute db queries and transactions
// using PostgreSQL.
type PostgresRepository struct {
	*db.Queries
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new instance of PostgresRepository.
func NewPostgresRepository(queries *db.Queries, pool *pgxpool.Pool) Repository {
	return &PostgresRepository{
		Queries: queries,
		pool:    pool,
	}
}

// ExecTx executes a function within a database transaction.
func (r *PostgresRepository) ExecTx(ctx context.Context, fn func(*db.Queries) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}

	q := db.New(tx)
	err = fn(q)
	if err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("tx err: %v, rb err: %v", err, rbErr)
		}
		return err
	}

	return tx.Commit(ctx)
}

func (r *PostgresRepository) CreateUsersBatch(ctx context.Context, users []db.CreateUserParams, passwords []string) ([]db.User, error) {
	if len(users) == 0 {
		return nil, nil
	}
	if len(users) != len(passwords) {
		return nil, fmt.Errorf("users/passwords batch length mismatch: %d != %d", len(users), len(passwords))
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	emails, usernames := splitBatchUsers(users)
	rows, err := tx.Query(ctx, createUsersBulk, emails, usernames)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	createdUsers := make([]db.User, len(users))
	createdCount := 0
	for rows.Next() {
		var user db.User
		var ord int
		if err := rows.Scan(&user.ID, &user.Email, &user.Username, &user.Status, &user.CreatedAt, &user.UpdatedAt, &ord); err != nil {
			return nil, err
		}

		if ord < 1 || ord > len(users) {
			return nil, fmt.Errorf("bulk insert returned invalid ordinality %d", ord)
		}

		createdUsers[ord-1] = user
		createdCount++
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if createdCount != len(users) {
		return nil, fmt.Errorf("bulk insert created %d users, expected %d", createdCount, len(users))
	}

	enqueuedPasswords, err := tx.CopyFrom(
		ctx,
		pgx.Identifier{"password_upgrade_queue"},
		[]string{"user_id", "weak_password_hash"},
		pgx.CopyFromRows(buildPasswordUpgradeQueueRows(createdUsers, passwords)),
	)
	if err != nil {
		return nil, err
	}
	if enqueuedPasswords != int64(len(passwords)) {
		return nil, fmt.Errorf("bulk insert created %d password_upgrade_queue rows, expected %d", enqueuedPasswords, len(passwords))
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, err
	}

	return createdUsers, nil
}

func splitBatchUsers(users []db.CreateUserParams) ([]string, []string) {
	emails := make([]string, len(users))
	usernames := make([]string, len(users))
	for i, user := range users {
		emails[i] = user.Email
		usernames[i] = user.Username
	}

	return emails, usernames
}

func buildPasswordUpgradeQueueRows(users []db.User, passwords []string) [][]any {
	rows := make([][]any, len(users))
	for i := range users {
		rows[i] = []any{
			users[i].ID,
			passwords[i],
		}
	}

	return rows
}
