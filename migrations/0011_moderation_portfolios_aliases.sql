BEGIN;

ALTER TABLE reviews ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'pending';
ALTER TABLE reviews ADD COLUMN IF NOT EXISTS moderation_note TEXT NOT NULL DEFAULT '';
ALTER TABLE reviews ADD COLUMN IF NOT EXISTS moderated_by UUID REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE reviews ADD COLUMN IF NOT EXISTS moderated_at TIMESTAMPTZ;
ALTER TABLE reviews ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();
ALTER TABLE reviews DROP CONSTRAINT IF EXISTS reviews_status_chk;
ALTER TABLE reviews ADD CONSTRAINT reviews_status_chk CHECK (status IN ('pending','approved','rejected'));

CREATE TABLE IF NOT EXISTS review_reports (
  id UUID PRIMARY KEY, review_id UUID NOT NULL REFERENCES reviews(id) ON DELETE CASCADE,
  reporter_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  reason TEXT NOT NULL, status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open','resolved','dismissed')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(), UNIQUE(review_id, reporter_id)
);

ALTER TABLE technicians ADD COLUMN IF NOT EXISTS review_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE technicians ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

CREATE TABLE IF NOT EXISTS technician_profiles (
  technician_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  bio TEXT NOT NULL DEFAULT '', experience_years INTEGER NOT NULL DEFAULT 0 CHECK (experience_years >= 0),
  price_per_hour NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (price_per_hour >= 0), is_available BOOLEAN NOT NULL DEFAULT true,
  county TEXT NOT NULL DEFAULT '', town TEXT NOT NULL DEFAULT '', village TEXT NOT NULL DEFAULT '', updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS technician_profile_skills (
  technician_id UUID NOT NULL REFERENCES technician_profiles(technician_id) ON DELETE CASCADE,
  skill TEXT NOT NULL, PRIMARY KEY(technician_id, skill)
);
CREATE TABLE IF NOT EXISTS technician_posts (
  id UUID PRIMARY KEY, technician_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  title TEXT NOT NULL, description TEXT NOT NULL DEFAULT '', service_type TEXT NOT NULL,
  media_urls JSONB NOT NULL DEFAULT '[]', status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','published','archived')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS technician_posts_public_idx ON technician_posts(service_type,created_at DESC) WHERE status='published';

CREATE TABLE IF NOT EXISTS location_aliases (
  id UUID PRIMARY KEY, location_id UUID NOT NULL REFERENCES locations(id) ON DELETE CASCADE,
  alias TEXT NOT NULL, normalized_alias TEXT NOT NULL, created_by UUID REFERENCES users(id) ON DELETE SET NULL,
  status TEXT NOT NULL DEFAULT 'approved' CHECK(status IN ('pending','approved','rejected')), created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(location_id, normalized_alias)
);
CREATE INDEX IF NOT EXISTS location_aliases_search_idx ON location_aliases USING gin(normalized_alias gin_trgm_ops) WHERE status='approved';

CREATE TABLE IF NOT EXISTS location_boundaries (
  location_id UUID PRIMARY KEY REFERENCES locations(id) ON DELETE CASCADE,
  min_lat DOUBLE PRECISION NOT NULL CHECK(min_lat BETWEEN -90 AND 90),
  max_lat DOUBLE PRECISION NOT NULL CHECK(max_lat BETWEEN -90 AND 90),
  min_lon DOUBLE PRECISION NOT NULL CHECK(min_lon BETWEEN -180 AND 180),
  max_lon DOUBLE PRECISION NOT NULL CHECK(max_lon BETWEEN -180 AND 180),
  CHECK(min_lat <= max_lat AND min_lon <= max_lon)
);

COMMIT;
