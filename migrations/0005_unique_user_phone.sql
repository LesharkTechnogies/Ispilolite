BEGIN;

CREATE UNIQUE INDEX IF NOT EXISTS users_phone_uidx ON users (phone);

COMMIT;
