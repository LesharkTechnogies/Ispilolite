-- 0001_add_avatar_url.sql
--
-- Adds the avatar_url image column to the searchable profile tables for
-- deployments whose tables predate the search subsystem. Fresh installs get
-- the column from internal/search/schema.sql; this migration brings existing
-- databases up to the same shape. Idempotent (IF NOT EXISTS), safe to re-run.
--
-- Apply with:  psql "$DATABASE_URL" -f migrations/0001_add_avatar_url.sql

BEGIN;

-- Technicians: the user-facing avatar image URL (per the search API contract).
ALTER TABLE technicians ADD COLUMN IF NOT EXISTS avatar_url TEXT;

-- ISPs carry an avatar/logo URL as well; the fallback SELECT COALESCE(avatar_url,'').
ALTER TABLE isps ADD COLUMN IF NOT EXISTS avatar_url TEXT;

-- Base user accounts expose avatar_url too (models.User.AvatarURL).
ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_url TEXT;

COMMIT;
