BEGIN;

CREATE TABLE IF NOT EXISTS reviews (
  id UUID PRIMARY KEY,
  target_id UUID NOT NULL,
  target_type TEXT NOT NULL CHECK (target_type IN ('isp','technician')),
  user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  rating INTEGER NOT NULL CHECK (rating BETWEEN 1 AND 5),
  comment TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (target_id, target_type, user_id)
);

CREATE INDEX IF NOT EXISTS reviews_target_idx ON reviews (target_id, target_type, created_at DESC);

COMMIT;
