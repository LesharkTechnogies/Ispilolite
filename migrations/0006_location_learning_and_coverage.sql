BEGIN;

ALTER TABLE locations ADD COLUMN IF NOT EXISTS parent_id UUID;
ALTER TABLE locations ADD COLUMN IF NOT EXISTS county TEXT NOT NULL DEFAULT '';
ALTER TABLE locations ADD COLUMN IF NOT EXISTS sub_county TEXT NOT NULL DEFAULT '';
ALTER TABLE locations ADD COLUMN IF NOT EXISTS ward TEXT NOT NULL DEFAULT '';
ALTER TABLE locations ADD COLUMN IF NOT EXISTS latitude DOUBLE PRECISION;
ALTER TABLE locations ADD COLUMN IF NOT EXISTS longitude DOUBLE PRECISION;
ALTER TABLE locations ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'pending';
ALTER TABLE locations ADD COLUMN IF NOT EXISTS is_verified BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE locations ADD COLUMN IF NOT EXISTS submission_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE locations ADD COLUMN IF NOT EXISTS popularity_score DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE locations ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

CREATE UNIQUE INDEX IF NOT EXISTS locations_name_type_parent_uidx
  ON locations (lower(trim(name)), type, COALESCE(parent_id::text, ''));

CREATE TABLE IF NOT EXISTS location_submissions (
  location_id UUID NOT NULL REFERENCES locations(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  latitude DOUBLE PRECISION,
  longitude DOUBLE PRECISION,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (location_id, user_id)
);

CREATE TABLE IF NOT EXISTS isp_coverage_locations (
  isp_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  location_id UUID NOT NULL REFERENCES locations(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (isp_id, location_id)
);

CREATE TABLE IF NOT EXISTS notifications (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  type TEXT NOT NULL,
  title TEXT NOT NULL,
  message TEXT NOT NULL,
  data JSONB NOT NULL DEFAULT '{}'::jsonb,
  read_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS notifications_coverage_recommendation_uidx
  ON notifications (user_id, type, (data->>'location_id'))
  WHERE type = 'coverage_recommendation';

CREATE INDEX IF NOT EXISTS locations_county_popularity_idx
  ON locations (lower(county), popularity_score DESC);

COMMIT;
