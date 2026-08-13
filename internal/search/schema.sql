-- schema.sql — Postgres fallback schema for the Ispilolite search subsystem.
--
-- This is the ground truth for the columns and extensions the PostgresRepository
-- (internal/search/pg_repository.go) expects. Elasticsearch is the fast path;
-- these tables back the fallback that keeps search available when ES is down.
--
-- Apply with:  psql "$DATABASE_URL" -f internal/search/schema.sql
--
-- Everything is IF NOT EXISTS / idempotent so it is safe to re-run.

-- ---------------------------------------------------------------------------
-- Extensions
-- ---------------------------------------------------------------------------
-- pg_trgm      : similarity() for fuzzy ranking + "did you mean" corrections.
-- cube         : prerequisite for earthdistance.
-- earthdistance: ll_to_earth / earth_box / earth_distance for near-me search.
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS cube;
CREATE EXTENSION IF NOT EXISTS earthdistance;

-- ---------------------------------------------------------------------------
-- isps — searchable ISP profiles (SearchISPs)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS isps (
    id           TEXT PRIMARY KEY,
    name         TEXT        NOT NULL,
    description  TEXT        NOT NULL DEFAULT '',
    avatar_url   TEXT,                              -- image URL; COALESCE'd to '' on read
    county       TEXT        NOT NULL DEFAULT '',
    sub_county   TEXT        NOT NULL DEFAULT '',
    village      TEXT        NOT NULL DEFAULT '',
    rating       DOUBLE PRECISION NOT NULL DEFAULT 0,
    review_count INTEGER     NOT NULL DEFAULT 0,
    is_active    BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Trigram indexes accelerate ILIKE '%...%' and similarity() ranking.
CREATE INDEX IF NOT EXISTS isps_name_trgm  ON isps USING gin (name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS isps_desc_trgm  ON isps USING gin (description gin_trgm_ops);
CREATE INDEX IF NOT EXISTS isps_county_idx ON isps (county);
CREATE INDEX IF NOT EXISTS isps_active_idx ON isps (is_active);

-- ---------------------------------------------------------------------------
-- technicians — searchable field-technician profiles
-- (SearchTechnicians / SearchTechniciansNear)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS technicians (
    id           TEXT PRIMARY KEY,
    user_id      TEXT        NOT NULL DEFAULT '',
    name         TEXT        NOT NULL,
    avatar_url   TEXT,                              -- image URL; COALESCE'd to '' on read
    isp_id       TEXT        NOT NULL DEFAULT '',
    isp_name     TEXT        NOT NULL DEFAULT '',
    county       TEXT        NOT NULL DEFAULT '',
    sub_county   TEXT        NOT NULL DEFAULT '',
    village      TEXT        NOT NULL DEFAULT '',
    rating       DOUBLE PRECISION NOT NULL DEFAULT 0,
    jobs_done    INTEGER     NOT NULL DEFAULT 0,
    is_available BOOLEAN     NOT NULL DEFAULT TRUE,
    lat          DOUBLE PRECISION,                  -- NULL when unlocated; guarded in near-me
    lon          DOUBLE PRECISION,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS technicians_name_trgm  ON technicians USING gin (name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS technicians_isp_trgm   ON technicians USING gin (isp_name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS technicians_county_idx ON technicians (county);
CREATE INDEX IF NOT EXISTS technicians_avail_idx  ON technicians (is_available);
-- GiST index over the earth point makes earth_box() containment fast.
CREATE INDEX IF NOT EXISTS technicians_earth_idx
    ON technicians USING gist (ll_to_earth(lat, lon))
    WHERE lat IS NOT NULL AND lon IS NOT NULL;

-- Roles and skills are many-per-technician, matched via EXISTS (...) subqueries
-- and aggregated with array_agg in the list query.
CREATE TABLE IF NOT EXISTS technician_roles (
    technician_id TEXT NOT NULL REFERENCES technicians(id) ON DELETE CASCADE,
    role          TEXT NOT NULL,
    PRIMARY KEY (technician_id, role)
);
CREATE INDEX IF NOT EXISTS technician_roles_role_idx  ON technician_roles (role);
CREATE INDEX IF NOT EXISTS technician_roles_role_trgm ON technician_roles USING gin (role gin_trgm_ops);

CREATE TABLE IF NOT EXISTS technician_skills (
    technician_id TEXT NOT NULL REFERENCES technicians(id) ON DELETE CASCADE,
    skill         TEXT NOT NULL,
    PRIMARY KEY (technician_id, skill)
);
CREATE INDEX IF NOT EXISTS technician_skills_skill_idx  ON technician_skills (skill);
CREATE INDEX IF NOT EXISTS technician_skills_skill_trgm ON technician_skills USING gin (skill gin_trgm_ops);

-- ---------------------------------------------------------------------------
-- locations — administrative places for place search, suggest, "did you mean"
-- (SearchLocations / Suggest / didYouMean)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS locations (
    id         TEXT PRIMARY KEY,
    name       TEXT        NOT NULL,
    type       TEXT        NOT NULL DEFAULT 'village',  -- county | sub_county | ward | village
    county     TEXT        NOT NULL DEFAULT '',
    sub_county TEXT        NOT NULL DEFAULT '',
    ward       TEXT        NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Trigram index powers both ILIKE prefix suggest and similarity() did-you-mean.
CREATE INDEX IF NOT EXISTS locations_name_trgm ON locations USING gin (name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS locations_type_idx  ON locations (type);
