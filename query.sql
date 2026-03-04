-- name: CreateUser :one
INSERT INTO users (email, username)
VALUES ($1, $2)
RETURNING id, email, username, status, created_at, updated_at;

-- name: CreateUserPassword :one
INSERT INTO user_passwords (user_id, password_hash, weak_password_hash)
VALUES ($1, $2, $3)
RETURNING user_id, last_changed_at;

-- name: CreateUserWeakPassword :one
INSERT INTO user_passwords (user_id, weak_password_hash)
VALUES ($1, $2)
RETURNING user_id, last_changed_at;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1 LIMIT 1;

-- name: GetUserPassword :one
SELECT * FROM user_passwords
WHERE user_id = $1 LIMIT 1;

-- name: CreateSession :one
INSERT INTO sessions (user_id, refresh_token_hash, user_agent, ip_address, expires_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, user_id, expires_at, created_at;

-- name: GetSession :one
SELECT * FROM sessions
WHERE id = $1 LIMIT 1;

-- name: GetSessionByRefreshToken :one
SELECT * FROM sessions
WHERE refresh_token_hash = $1 LIMIT 1;

-- name: DeleteSession :exec
DELETE FROM sessions
WHERE id = $1;

-- name: GetUserById :one
SELECT * FROM users
WHERE id = $1 LIMIT 1;

-- name: ListWeakPasswords :many
SELECT * FROM user_passwords
WHERE weak_password_hash IS NOT NULL
LIMIT $1;

-- name: UpgradePassword :exec
UPDATE user_passwords
SET password_hash = $2, weak_password_hash = NULL, last_changed_at = NOW()
WHERE user_id = $1;

