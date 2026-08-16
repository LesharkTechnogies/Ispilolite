BEGIN;

ALTER TABLE isp_packages ADD COLUMN IF NOT EXISTS version INTEGER NOT NULL DEFAULT 1;
ALTER TABLE isp_packages ADD COLUMN IF NOT EXISTS archived_at TIMESTAMPTZ;
ALTER TABLE isp_packages ADD COLUMN IF NOT EXISTS max_subscriptions INTEGER CHECK(max_subscriptions IS NULL OR max_subscriptions >= 0);

CREATE TABLE IF NOT EXISTS isp_package_versions (
  id UUID PRIMARY KEY, package_id UUID NOT NULL REFERENCES isp_packages(id) ON DELETE RESTRICT,
  version INTEGER NOT NULL, name TEXT NOT NULL, category TEXT NOT NULL,
  speed_value NUMERIC(12,3) NOT NULL, speed_unit_id UUID NOT NULL REFERENCES package_units(id),
  base_price NUMERIC(14,2) NOT NULL, billing_cycle TEXT NOT NULL, capacity_type TEXT NOT NULL,
  capacity_value NUMERIC(14,3), capacity_unit_id UUID REFERENCES package_units(id),
  description TEXT NOT NULL DEFAULT '', max_subscriptions INTEGER,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(), UNIQUE(package_id, version)
);

INSERT INTO isp_package_versions(id,package_id,version,name,category,speed_value,speed_unit_id,base_price,billing_cycle,capacity_type,capacity_value,capacity_unit_id,description,max_subscriptions)
SELECT gen_random_uuid(),p.id,p.version,p.name,p.category,p.speed_value,p.speed_unit_id,p.base_price,p.billing_cycle,p.capacity_type,p.capacity_value,p.capacity_unit_id,p.description,p.max_subscriptions
FROM isp_packages p WHERE p.speed_unit_id IS NOT NULL
ON CONFLICT(package_id,version) DO NOTHING;

CREATE TABLE IF NOT EXISTS package_capacity_reservations (
  id UUID PRIMARY KEY, package_id UUID NOT NULL REFERENCES isp_packages(id) ON DELETE RESTRICT,
  package_version_id UUID NOT NULL REFERENCES isp_package_versions(id) ON DELETE RESTRICT,
  customer_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  county TEXT NOT NULL DEFAULT '', status TEXT NOT NULL DEFAULT 'reserved' CHECK(status IN ('reserved','converted','released','expired')),
  expires_at TIMESTAMPTZ NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS package_reservation_customer_active_uidx ON package_capacity_reservations(package_id,customer_id) WHERE status='reserved';

CREATE TABLE IF NOT EXISTS package_subscriptions (
  id UUID PRIMARY KEY, package_id UUID NOT NULL REFERENCES isp_packages(id) ON DELETE RESTRICT,
  package_version_id UUID NOT NULL REFERENCES isp_package_versions(id) ON DELETE RESTRICT,
  customer_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT, isp_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  reservation_id UUID REFERENCES package_capacity_reservations(id) ON DELETE SET NULL,
  status TEXT NOT NULL CHECK(status IN ('pending','active','suspended','cancelled','expired')),
  county TEXT NOT NULL DEFAULT '', price NUMERIC(14,2) NOT NULL,
  package_name TEXT NOT NULL, category TEXT NOT NULL, speed_value NUMERIC(12,3) NOT NULL, speed_unit TEXT NOT NULL,
  started_at TIMESTAMPTZ, ends_at TIMESTAMPTZ, created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS package_subscriptions_customer_idx ON package_subscriptions(customer_id,status,created_at DESC);
CREATE INDEX IF NOT EXISTS package_subscriptions_isp_idx ON package_subscriptions(isp_id,status,created_at DESC);
CREATE INDEX IF NOT EXISTS package_subscriptions_capacity_idx ON package_subscriptions(package_id,status) WHERE status IN ('pending','active','suspended');

COMMIT;
