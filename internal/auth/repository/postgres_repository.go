package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jeday/auth/internal/db"
)

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
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	q := db.New(tx)
	createdUsers := make([]db.User, 0, len(users))

	for i, userParam := range users {
		user, err := q.CreateUser(ctx, userParam)
		if err != nil {
			return nil, err
		}

		_, err = q.CreateUserWeakPassword(ctx, db.CreateUserWeakPasswordParams{
			UserID:           user.ID,
			WeakPasswordHash: pgtype.Text{String: passwords[i], Valid: true},
		})
		if err != nil {
			return nil, err
		}
		createdUsers = append(createdUsers, user)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, err
	}

	return createdUsers, nil
}
