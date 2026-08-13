BEGIN;

ALTER TABLE users ADD COLUMN IF NOT EXISTS username TEXT NOT NULL DEFAULT '';
CREATE UNIQUE INDEX IF NOT EXISTS users_username_uidx ON users (lower(username)) WHERE username <> '';

ALTER TABLE installations DROP CONSTRAINT IF EXISTS installations_status_chk;
ALTER TABLE installations ADD CONSTRAINT installations_status_chk CHECK (status IN ('pending', 'accepted', 'assigned', 'in_progress', 'completed', 'cancelled', 'rejected'));

COMMIT;
