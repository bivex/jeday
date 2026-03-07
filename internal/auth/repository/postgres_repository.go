package repository

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
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

	createdUsers, err := buildUsersForCopy(users, time.Now().UTC())
	if err != nil {
		return nil, err
	}

	insertedUsers, err := tx.CopyFrom(
		ctx,
		pgx.Identifier{"users"},
		[]string{"id", "email", "username", "status", "created_at", "updated_at"},
		pgx.CopyFromRows(buildUserRows(createdUsers)),
	)
	if err != nil {
		return nil, err
	}
	if insertedUsers != int64(len(createdUsers)) {
		return nil, fmt.Errorf("bulk insert created %d users, expected %d", insertedUsers, len(createdUsers))
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

func buildUsersForCopy(users []db.CreateUserParams, now time.Time) ([]db.User, error) {
	createdUsers := make([]db.User, len(users))
	createdAt := pgtype.Timestamptz{Time: now, Valid: true}

	for i, user := range users {
		id, err := newUUIDv4()
		if err != nil {
			return nil, err
		}

		createdUsers[i] = db.User{
			ID:        id,
			Email:     user.Email,
			Username:  user.Username,
			Status:    "pending",
			CreatedAt: createdAt,
			UpdatedAt: createdAt,
		}
	}

	return createdUsers, nil
}

func buildUserRows(users []db.User) [][]any {
	rows := make([][]any, len(users))
	for i := range users {
		rows[i] = []any{
			users[i].ID,
			users[i].Email,
			users[i].Username,
			users[i].Status,
			users[i].CreatedAt,
			users[i].UpdatedAt,
		}
	}

	return rows
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

func newUUIDv4() (pgtype.UUID, error) {
	var uuid pgtype.UUID
	if _, err := rand.Read(uuid.Bytes[:]); err != nil {
		return pgtype.UUID{}, err
	}
	uuid.Bytes[6] = (uuid.Bytes[6] & 0x0f) | 0x40
	uuid.Bytes[8] = (uuid.Bytes[8] & 0x3f) | 0x80
	uuid.Valid = true
	return uuid, nil
}
