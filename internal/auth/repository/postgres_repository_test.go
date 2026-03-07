package repository

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jeday/auth/internal/db"
)

func TestBuildUsersForCopyPreservesOrderAndDefaults(t *testing.T) {
	users := []db.CreateUserParams{
		{Email: "first@example.com", Username: "first"},
		{Email: "second@example.com", Username: "second"},
	}
	now := time.Unix(1700000000, 0).UTC()

	created, err := buildUsersForCopy(users, now)
	if err != nil {
		t.Fatalf("buildUsersForCopy() error = %v", err)
	}

	if len(created) != len(users) {
		t.Fatalf("len(created) = %d, want %d", len(created), len(users))
	}
	if created[0].Email != "first@example.com" || created[1].Email != "second@example.com" {
		t.Fatalf("emails = %#v, want original order", created)
	}
	if created[0].Username != "first" || created[1].Username != "second" {
		t.Fatalf("usernames = %#v, want original order", created)
	}
	if created[0].Status != "pending" || created[1].Status != "pending" {
		t.Fatalf("statuses = %#v, want pending", created)
	}
	if !created[0].ID.Valid || !created[1].ID.Valid || created[0].ID == created[1].ID {
		t.Fatalf("generated IDs = %#v, want distinct valid UUIDs", created)
	}
	if !created[0].CreatedAt.Valid || !created[0].UpdatedAt.Valid || created[0].CreatedAt.Time != now || created[0].UpdatedAt.Time != now {
		t.Fatalf("timestamps = %#v, want %v", created[0], now)
	}
}

func TestBuildUserRowsPreservesFieldOrder(t *testing.T) {
	user := db.User{
		ID:        pgtype.UUID{Bytes: [16]byte{1}, Valid: true},
		Email:     "first@example.com",
		Username:  "first",
		Status:    "pending",
		CreatedAt: pgtype.Timestamptz{Valid: true},
		UpdatedAt: pgtype.Timestamptz{Valid: true},
	}

	rows := buildUserRows([]db.User{user})

	if len(rows) != 1 || len(rows[0]) != 6 {
		t.Fatalf("rows = %#v, want single 6-column row", rows)
	}
	if got := rows[0][0].(pgtype.UUID); got != user.ID {
		t.Fatalf("rows[0][0] = %v, want %v", got, user.ID)
	}
	if got := rows[0][1].(string); got != user.Email {
		t.Fatalf("rows[0][1] = %q, want %q", got, user.Email)
	}
}

func TestBuildPasswordUpgradeQueueRowsPreservesUserPasswordPairs(t *testing.T) {
	users := []db.User{
		{ID: pgtype.UUID{Bytes: [16]byte{1}, Valid: true}},
		{ID: pgtype.UUID{Bytes: [16]byte{2}, Valid: true}},
	}
	passwords := []string{"hash-1", "hash-2"}

	rows := buildPasswordUpgradeQueueRows(users, passwords)

	if len(rows) != 2 {
		t.Fatalf("len(rows) = %d, want 2", len(rows))
	}
	if got := rows[0][0].(pgtype.UUID); got != users[0].ID {
		t.Fatalf("rows[0][0] = %v, want %v", got, users[0].ID)
	}
	if got := rows[1][1].(string); got != "hash-2" {
		t.Fatalf("rows[1][1] = %q, want hash-2", got)
	}
}
