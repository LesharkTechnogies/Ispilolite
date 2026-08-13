BEGIN;

CREATE SEQUENCE IF NOT EXISTS quotation_number_seq;

CREATE TABLE IF NOT EXISTS quotation_units (
  id UUID PRIMARY KEY,
  name TEXT NOT NULL,
  singular_name TEXT NOT NULL,
  plural_name TEXT NOT NULL,
  symbol TEXT NOT NULL DEFAULT '',
  is_system BOOLEAN NOT NULL DEFAULT false,
  issuer_id UUID REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS quotation_system_units_name_uidx
  ON quotation_units (lower(name)) WHERE is_system = true;
CREATE UNIQUE INDEX IF NOT EXISTS quotation_custom_units_name_uidx
  ON quotation_units (issuer_id, lower(name)) WHERE is_system = false;

INSERT INTO quotation_units (id,name,singular_name,plural_name,symbol,is_system)
VALUES
  (gen_random_uuid(),'Meter','meter','meters','m',true),
  (gen_random_uuid(),'Piece','piece','pieces','pcs',true),
  (gen_random_uuid(),'Box','box','boxes','box',true),
  (gen_random_uuid(),'Pack','pack','packs','pkt',true),
  (gen_random_uuid(),'Roll','roll','rolls','roll',true),
  (gen_random_uuid(),'Set','set','sets','set',true),
  (gen_random_uuid(),'Hour','hour','hours','hr',true),
  (gen_random_uuid(),'Day','day','days','day',true),
  (gen_random_uuid(),'Month','month','months','month',true),
  (gen_random_uuid(),'Service','service','services','svc',true)
ON CONFLICT DO NOTHING;

CREATE TABLE IF NOT EXISTS tax_rates (
  id UUID PRIMARY KEY,
  name TEXT NOT NULL,
  rate NUMERIC(7,4) NOT NULL CHECK (rate >= 0 AND rate <= 100),
  is_active BOOLEAN NOT NULL DEFAULT true,
  is_default BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO tax_rates (id,name,rate,is_active,is_default)
SELECT gen_random_uuid(),'VAT',16,true,true
WHERE NOT EXISTS (SELECT 1 FROM tax_rates WHERE is_default=true);

CREATE TABLE IF NOT EXISTS quotations (
  id UUID PRIMARY KEY,
  public_code TEXT NOT NULL UNIQUE,
  quotation_number TEXT NOT NULL UNIQUE,
  issuer_id UUID NOT NULL REFERENCES users(id),
  issuer_role TEXT NOT NULL CHECK (issuer_role IN ('isp','technician')),
  customer_id UUID NOT NULL REFERENCES users(id),
  request_id UUID REFERENCES service_requests(id) ON DELETE SET NULL,
  company_name TEXT NOT NULL, company_phone TEXT NOT NULL DEFAULT '', company_email TEXT NOT NULL DEFAULT '', company_address TEXT NOT NULL DEFAULT '', company_logo_url TEXT NOT NULL DEFAULT '', business_type TEXT NOT NULL,
  customer_name TEXT NOT NULL, customer_phone TEXT NOT NULL DEFAULT '', customer_email TEXT NOT NULL DEFAULT '', customer_location TEXT NOT NULL DEFAULT '',
  currency TEXT NOT NULL DEFAULT 'KES', subtotal NUMERIC(14,2) NOT NULL, discount_amount NUMERIC(14,2) NOT NULL DEFAULT 0,
  transport_enabled BOOLEAN NOT NULL DEFAULT false, transport_amount NUMERIC(14,2) NOT NULL DEFAULT 0,
  tax_enabled BOOLEAN NOT NULL DEFAULT false, taxable_amount NUMERIC(14,2) NOT NULL, tax_amount NUMERIC(14,2) NOT NULL DEFAULT 0, tax_rate NUMERIC(7,4) NOT NULL DEFAULT 0, tax_mode TEXT NOT NULL DEFAULT 'NONE',
  total_amount NUMERIC(14,2) NOT NULL, payment_method TEXT NOT NULL DEFAULT 'NONE', payment_details JSONB NOT NULL DEFAULT '{}', terms JSONB NOT NULL DEFAULT '[]', notes TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'FINALIZED' CHECK (status IN ('FINALIZED','SENT','VIEWED','ACCEPTED','REJECTED','EXPIRED')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(), finalized_at TIMESTAMPTZ NOT NULL DEFAULT now(), expires_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS quotation_items (
  id UUID PRIMARY KEY, quotation_id UUID NOT NULL REFERENCES quotations(id) ON DELETE CASCADE,
  item TEXT NOT NULL, description TEXT NOT NULL DEFAULT '', unit_id UUID NOT NULL REFERENCES quotation_units(id), unit_name TEXT NOT NULL, unit_symbol TEXT NOT NULL DEFAULT '',
  quantity NUMERIC(14,4) NOT NULL CHECK (quantity > 0), unit_price NUMERIC(14,2) NOT NULL CHECK (unit_price >= 0), gross_amount NUMERIC(14,2) NOT NULL,
  discount_type TEXT NOT NULL DEFAULT 'NONE' CHECK (discount_type IN ('NONE','FIXED','PERCENTAGE')), discount_value NUMERIC(14,4) NOT NULL DEFAULT 0, discount_amount NUMERIC(14,2) NOT NULL DEFAULT 0, amount NUMERIC(14,2) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS quotation_tax (
  id UUID PRIMARY KEY, quotation_id UUID NOT NULL UNIQUE REFERENCES quotations(id) ON DELETE CASCADE, tax_rate_id UUID REFERENCES tax_rates(id), rate NUMERIC(7,4) NOT NULL,
  calculation_type TEXT NOT NULL CHECK (calculation_type IN ('EXCLUSIVE','INCLUSIVE')), taxable_amount NUMERIC(14,2) NOT NULL, tax_amount NUMERIC(14,2) NOT NULL
);

CREATE INDEX IF NOT EXISTS quotations_issuer_idx ON quotations (issuer_id, created_at DESC);
CREATE INDEX IF NOT EXISTS quotations_customer_idx ON quotations (customer_id, created_at DESC);
CREATE INDEX IF NOT EXISTS quotation_items_quotation_idx ON quotation_items (quotation_id);

COMMIT;
