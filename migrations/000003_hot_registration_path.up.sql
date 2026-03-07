ALTER TABLE users ADD COLUMN weak_password_hash TEXT;

UPDATE users u
SET weak_password_hash = up.weak_password_hash
FROM user_passwords up
WHERE up.user_id = u.id
  AND up.weak_password_hash IS NOT NULL
  AND u.weak_password_hash IS NULL;

UPDATE user_passwords
SET weak_password_hash = NULL
WHERE weak_password_hash IS NOT NULL;