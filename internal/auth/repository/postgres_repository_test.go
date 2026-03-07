package repository

import (
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jeday/auth/internal/db"
)

func TestSplitBatchUsersPreservesOrder(t *testing.T) {
	users := []db.CreateUserParams{
		{Email: "first@example.com", Username: "first"},
		{Email: "second@example.com", Username: "second"},
	}

	emails, usernames := splitBatchUsers(users)

	if emails[0] != "first@example.com" || emails[1] != "second@example.com" {
		t.Fatalf("emails = %v, want original order", emails)
	}
	if usernames[0] != "first" || usernames[1] != "second" {
		t.Fatalf("usernames = %v, want original order", usernames)
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
