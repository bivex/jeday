-- Миграция для асинхронного хеширования
ALTER TABLE user_passwords ADD COLUMN weak_password_hash TEXT;
ALTER TABLE user_passwords ALTER COLUMN password_hash DROP NOT NULL;
