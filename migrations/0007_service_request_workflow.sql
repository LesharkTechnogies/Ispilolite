BEGIN;

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS auth_sessions (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash TEXT NOT NULL UNIQUE,
  expires_at TIMESTAMPTZ NOT NULL,
  revoked_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  last_used_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS auth_sessions_user_active_idx ON auth_sessions (user_id, expires_at) WHERE revoked_at IS NULL;

CREATE TABLE IF NOT EXISTS service_requests (
  id UUID PRIMARY KEY,
  customer_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  request_type TEXT NOT NULL CHECK (request_type IN ('internet','technician_service')),
  mode TEXT NOT NULL CHECK (mode IN ('direct','broadcast')),
  target_isp_id UUID REFERENCES users(id) ON DELETE SET NULL,
  target_technician_id UUID REFERENCES users(id) ON DELETE SET NULL,
  assigned_isp_id UUID REFERENCES users(id) ON DELETE SET NULL,
  assigned_technician_id UUID REFERENCES users(id) ON DELETE SET NULL,
  location_id UUID REFERENCES locations(id) ON DELETE SET NULL,
  county TEXT NOT NULL DEFAULT '',
  town TEXT NOT NULL DEFAULT '',
  village TEXT NOT NULL DEFAULT '',
  service_type TEXT NOT NULL,
  speed_mbps INTEGER,
  description TEXT NOT NULL DEFAULT '',
  budget NUMERIC(12,2) NOT NULL DEFAULT 0,
  preferred_date TIMESTAMPTZ,
  status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open','assigned','accepted','in_progress','completed','unavailable','cancelled','deleted')),
  is_available BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at TIMESTAMPTZ,
  CHECK (speed_mbps IS NULL OR speed_mbps > 0),
  CHECK (budget >= 0)
);

CREATE TABLE IF NOT EXISTS service_request_applications (
  id UUID PRIMARY KEY,
  request_id UUID NOT NULL REFERENCES service_requests(id) ON DELETE CASCADE,
  applicant_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  applicant_role TEXT NOT NULL CHECK (applicant_role IN ('technician','isp')),
  message TEXT NOT NULL DEFAULT '',
  proposed_price NUMERIC(12,2) NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','accepted','rejected','withdrawn')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (request_id, applicant_id),
  CHECK (proposed_price >= 0)
);

CREATE INDEX IF NOT EXISTS service_requests_available_idx
  ON service_requests (lower(county), lower(town), service_type, created_at DESC)
  WHERE is_available = true AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS service_request_applications_request_idx
  ON service_request_applications (request_id, status, created_at);

CREATE OR REPLACE FUNCTION notify_service_request_recipients() RETURNS trigger AS $$
BEGIN
  IF NEW.mode = 'direct' THEN
    IF NEW.target_isp_id IS NOT NULL THEN
      INSERT INTO notifications (id,user_id,type,title,message,data,created_at)
      VALUES (gen_random_uuid(), NEW.target_isp_id, 'service_request', 'Direct internet request',
        'A customer sent you an internet request.', jsonb_build_object('request_id',NEW.id,'request_type',NEW.request_type), now());
    ELSIF NEW.target_technician_id IS NOT NULL THEN
      INSERT INTO notifications (id,user_id,type,title,message,data,created_at)
      VALUES (gen_random_uuid(), NEW.target_technician_id, 'service_request', 'Direct service request',
        'A customer sent you a service request.', jsonb_build_object('request_id',NEW.id,'request_type',NEW.request_type), now());
    END IF;
  ELSE
    INSERT INTO notifications (id,user_id,type,title,message,data,created_at)
    SELECT gen_random_uuid(), u.id, 'available_job', 'New available job',
      'A new job is available in ' || COALESCE(NULLIF(NEW.town,''),NULLIF(NEW.county,''),'your area') || '.',
      jsonb_build_object('request_id',NEW.id,'request_type',NEW.request_type), now()
    FROM users u
    WHERE u.id <> NEW.customer_id AND u.role IN ('technician','isp') AND u.status = 'active'
      AND (NEW.county = '' OR lower(COALESCE(u.county,'')) = lower(NEW.county));
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS service_request_notification_trigger ON service_requests;
CREATE TRIGGER service_request_notification_trigger
AFTER INSERT ON service_requests
FOR EACH ROW EXECUTE FUNCTION notify_service_request_recipients();

COMMIT;
