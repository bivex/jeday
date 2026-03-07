CREATE INDEX users_weak_password_queue_idx
ON users (created_at, id)
WHERE weak_password_hash IS NOT NULL;