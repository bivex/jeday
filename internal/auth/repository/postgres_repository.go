package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jeday/auth/internal/db"
)

// Repository defines all functions to execute db queries and transactions
type Repository interface {
	db.Querier
	ExecTx(ctx context.Context, fn func(*db.Queries) error) error
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
