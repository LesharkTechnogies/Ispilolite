BEGIN;

CREATE TABLE IF NOT EXISTS package_units (
  id UUID PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  symbol TEXT NOT NULL UNIQUE,
  dimension TEXT NOT NULL CHECK (dimension IN ('bandwidth','data')),
  multiplier NUMERIC(18,6) NOT NULL CHECK (multiplier > 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO package_units (id,name,symbol,dimension,multiplier) VALUES
  (gen_random_uuid(),'Megabits per second','Mbps','bandwidth',1),
  (gen_random_uuid(),'Gigabits per second','Gbps','bandwidth',1000),
  (gen_random_uuid(),'Gigabytes','GB','data',1),
  (gen_random_uuid(),'Terabytes','TB','data',1024)
ON CONFLICT (symbol) DO NOTHING;

ALTER TABLE isp_packages ADD COLUMN IF NOT EXISTS category TEXT NOT NULL DEFAULT 'pppoe';
ALTER TABLE isp_packages ADD COLUMN IF NOT EXISTS speed_value NUMERIC(12,3) NOT NULL DEFAULT 0;
ALTER TABLE isp_packages ADD COLUMN IF NOT EXISTS speed_unit_id UUID REFERENCES package_units(id);
ALTER TABLE isp_packages ADD COLUMN IF NOT EXISTS base_price NUMERIC(14,2) NOT NULL DEFAULT 0;
ALTER TABLE isp_packages ADD COLUMN IF NOT EXISTS billing_cycle TEXT NOT NULL DEFAULT 'monthly';
ALTER TABLE isp_packages ADD COLUMN IF NOT EXISTS capacity_type TEXT NOT NULL DEFAULT 'unlimited';
ALTER TABLE isp_packages ADD COLUMN IF NOT EXISTS capacity_value NUMERIC(14,3);
ALTER TABLE isp_packages ADD COLUMN IF NOT EXISTS capacity_unit_id UUID REFERENCES package_units(id);
ALTER TABLE isp_packages ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE isp_packages ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

-- Preserve the original catalog while moving calculations to normalized units.
-- Legacy columns are retained for compatibility but are no longer used by the
-- application after this migration.
ALTER TABLE isp_packages ADD COLUMN IF NOT EXISTS speed TEXT;
ALTER TABLE isp_packages ADD COLUMN IF NOT EXISTS price NUMERIC(14,2);
ALTER TABLE isp_packages ALTER COLUMN speed DROP NOT NULL;
ALTER TABLE isp_packages ALTER COLUMN price DROP NOT NULL;

UPDATE isp_packages
SET speed_value = CASE
      WHEN COALESCE(speed, '') ~ '^[[:space:]]*[0-9]+([.][0-9]+)?' THEN
        substring(speed from '^[[:space:]]*([0-9]+([.][0-9]+)?)')::numeric
      ELSE speed_value
    END,
    base_price = CASE
      WHEN price IS NOT NULL AND price >= 0 THEN price
      ELSE base_price
    END
WHERE speed_unit_id IS NULL OR base_price = 0;

UPDATE isp_packages
SET speed_unit_id = (SELECT id FROM package_units WHERE symbol = 'Mbps' LIMIT 1)
WHERE speed_unit_id IS NULL;

UPDATE isp_packages
SET category = CASE WHEN lower(COALESCE(category, '')) IN ('pppoe','hotspot') THEN lower(category) ELSE 'pppoe' END,
    billing_cycle = CASE WHEN lower(COALESCE(billing_cycle, '')) IN ('monthly','weekly','daily','hourly') THEN lower(billing_cycle) ELSE 'monthly' END,
    capacity_type = CASE WHEN lower(COALESCE(capacity_type, '')) IN ('unlimited','capped') THEN lower(capacity_type) ELSE 'unlimited' END;
ALTER TABLE isp_packages DROP CONSTRAINT IF EXISTS isp_packages_category_chk;
ALTER TABLE isp_packages DROP CONSTRAINT IF EXISTS isp_packages_cycle_chk;
ALTER TABLE isp_packages DROP CONSTRAINT IF EXISTS isp_packages_capacity_chk;
ALTER TABLE isp_packages ADD CONSTRAINT isp_packages_category_chk CHECK (category IN ('pppoe','hotspot'));
ALTER TABLE isp_packages ADD CONSTRAINT isp_packages_cycle_chk CHECK (billing_cycle IN ('monthly','weekly','daily','hourly'));
ALTER TABLE isp_packages ADD CONSTRAINT isp_packages_capacity_chk CHECK (capacity_type IN ('unlimited','capped'));

CREATE TABLE IF NOT EXISTS isp_package_county_prices (
  package_id UUID NOT NULL REFERENCES isp_packages(id) ON DELETE CASCADE,
  county TEXT NOT NULL,
  price NUMERIC(14,2) NOT NULL CHECK (price >= 0),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (package_id, county)
);
CREATE INDEX IF NOT EXISTS isp_packages_active_price_idx ON isp_packages (isp_id, is_active, base_price);
CREATE INDEX IF NOT EXISTS isp_package_county_prices_county_price_idx ON isp_package_county_prices (lower(county), price);

COMMIT;
