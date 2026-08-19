BEGIN;

ALTER TABLE tax_rates ADD COLUMN IF NOT EXISTS owner_id UUID REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE tax_rates ADD COLUMN IF NOT EXISTS created_by UUID REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE tax_rates ADD COLUMN IF NOT EXISTS scope TEXT NOT NULL DEFAULT 'SYSTEM' CHECK(scope IN ('SYSTEM','CUSTOM'));
CREATE UNIQUE INDEX IF NOT EXISTS tax_rates_custom_owner_name_uidx ON tax_rates(owner_id,lower(name)) WHERE scope='CUSTOM';

CREATE TABLE IF NOT EXISTS business_profiles (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  legal_name TEXT NOT NULL,
  registration_number TEXT NOT NULL DEFAULT '',
  tax_number TEXT NOT NULL DEFAULT '',
  address TEXT NOT NULL DEFAULT '',
  phone TEXT NOT NULL DEFAULT '',
  email TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','approved','rejected','suspended')),
  moderated_by UUID REFERENCES users(id) ON DELETE SET NULL,
  moderated_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE isp_packages ADD COLUMN IF NOT EXISTS moderation_status TEXT NOT NULL DEFAULT 'pending' CHECK(moderation_status IN ('pending','approved','rejected','suspended'));
ALTER TABLE isp_packages ADD COLUMN IF NOT EXISTS moderation_note TEXT NOT NULL DEFAULT '';
ALTER TABLE isp_packages ADD COLUMN IF NOT EXISTS moderated_by UUID REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE isp_packages ADD COLUMN IF NOT EXISTS moderated_at TIMESTAMPTZ;

CREATE TABLE IF NOT EXISTS audit_events (
  id UUID PRIMARY KEY,
  actor_id UUID REFERENCES users(id) ON DELETE SET NULL,
  actor_role TEXT NOT NULL DEFAULT '',
  event TEXT NOT NULL,
  action TEXT NOT NULL CHECK(action IN ('CREATE','UPDATE','DELETE','GET','LOGIN','REGISTER','REQUEST','EXPORT','SYSTEM')),
  resource_type TEXT NOT NULL DEFAULT '',
  resource_id TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL,
  success BOOLEAN NOT NULL,
  request_id TEXT NOT NULL DEFAULT '',
  ip_address TEXT NOT NULL DEFAULT '',
  user_agent TEXT NOT NULL DEFAULT '',
  metadata JSONB NOT NULL DEFAULT '{}',
  occurred_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS audit_events_time_idx ON audit_events(occurred_at DESC);
CREATE INDEX IF NOT EXISTS audit_events_event_idx ON audit_events(event,occurred_at DESC);
CREATE INDEX IF NOT EXISTS audit_events_actor_idx ON audit_events(actor_id,occurred_at DESC);

CREATE TABLE IF NOT EXISTS audit_archives (
  id UUID PRIMARY KEY,
  period_type TEXT NOT NULL CHECK(period_type IN ('DAILY','WEEKLY','MONTHLY','YEARLY','CUSTOM')),
  period_start TIMESTAMPTZ NOT NULL,
  period_end TIMESTAMPTZ NOT NULL,
  file_name TEXT NOT NULL UNIQUE,
  storage_path TEXT NOT NULL,
  checksum TEXT NOT NULL,
  event_count INTEGER NOT NULL,
  created_by UUID REFERENCES users(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(period_type,period_start,period_end)
);

COMMIT;
