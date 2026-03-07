CREATE TABLE password_upgrade_queue (
    user_id UUID PRIMARY KEY,
    weak_password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO password_upgrade_queue (user_id, weak_password_hash, created_at)
SELECT u.id, COALESCE(u.weak_password_hash, up.weak_password_hash), u.created_at
FROM users u
LEFT JOIN user_passwords up ON up.user_id = u.id
WHERE COALESCE(u.weak_password_hash, up.weak_password_hash) IS NOT NULL
ON CONFLICT (user_id) DO NOTHING;

CREATE INDEX password_upgrade_queue_created_at_idx
ON password_upgrade_queue (created_at, user_id);

DROP INDEX IF EXISTS users_weak_password_queue_idx;

ALTER TABLE users DROP COLUMN weak_password_hash;