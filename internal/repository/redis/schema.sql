-- Enable extensions if they don't exist
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "postgis";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- ISPs table to store internet service provider profiles
CREATE TABLE IF NOT EXISTS locations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL CHECK (type IN ('county', 'sub_county', 'village', 'building')),
    parent_id UUID REFERENCES locations(id) ON DELETE SET NULL,
    point GEOGRAPHY(POINT, 4326),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(name, parent_id)
);

-- Index for text search on locations
CREATE INDEX IF NOT EXISTS locations_name_trgm_idx ON locations USING GIN (name gin_trgm_ops);
-- Geospatial index for locations
CREATE INDEX IF NOT EXISTS locations_point_gist_idx ON locations USING GIST (point);
-- Index for parent_id for faster hierarchy traversal
CREATE INDEX IF NOT EXISTS locations_parent_id_idx ON locations (parent_id);

CREATE TABLE IF NOT EXISTS isps (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    avatar_url VARCHAR(255),
    rating NUMERIC(3, 2) DEFAULT 0.0,
    review_count INT DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,
    office_location_id UUID REFERENCES locations(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Client profiles table, linked to a user account
CREATE TABLE IF NOT EXISTS clients (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Technician profiles table, linked to a user account
CREATE TABLE IF NOT EXISTS technicians (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE UNIQUE,
    isp_id UUID REFERENCES isps(id) ON DELETE SET NULL,
    isp_name VARCHAR(255), -- Denormalized for convenience in search
    location_id UUID REFERENCES locations(id) ON DELETE SET NULL,
    rating NUMERIC(3, 2) DEFAULT 0.0,
    jobs_done INT DEFAULT 0,
    is_available BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Join table for technician skills (many-to-many)
CREATE TABLE IF NOT EXISTS technician_skills (
    technician_id UUID NOT NULL REFERENCES technicians(id) ON DELETE CASCADE,
    skill VARCHAR(100) NOT NULL,
    PRIMARY KEY (technician_id, skill)
);

-- Join table for technician roles (many-to-many)
CREATE TABLE IF NOT EXISTS technician_roles (
    technician_id UUID NOT NULL REFERENCES technicians(id) ON DELETE CASCADE,
    role VARCHAR(100) NOT NULL,
    PRIMARY KEY (technician_id, role)
);

-- Junction table to link ISPs to the locations they serve
CREATE TABLE IF NOT EXISTS isp_served_locations (
    isp_id UUID NOT NULL REFERENCES isps(id) ON DELETE CASCADE,
    location_id UUID NOT NULL REFERENCES locations(id) ON DELETE CASCADE,
    PRIMARY KEY (isp_id, location_id)
);

-- Index for text search
CREATE INDEX IF NOT EXISTS isps_name_trgm_idx ON isps USING GIN (name gin_trgm_ops);
-- Geospatial index

-- ISP coverage areas (polygons)
CREATE TABLE IF NOT EXISTS isp_coverage_areas (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    isp_id UUID NOT NULL REFERENCES isps(id) ON DELETE CASCADE,
    area_name VARCHAR(255),
    coverage_polygon GEOGRAPHY(POLYGON, 4326) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Geospatial index for coverage areas
CREATE INDEX IF NOT EXISTS isp_coverage_areas_polygon_gist_idx ON isp_coverage_areas USING GIST (coverage_polygon);

-- Trigger to update updated_at on isps table
CREATE OR REPLACE FUNCTION trigger_set_timestamp() RETURNS TRIGGER AS $$
BEGIN NEW.updated_at = NOW(); RETURN NEW; END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER set_isps_timestamp BEFORE UPDATE ON isps FOR EACH ROW EXECUTE PROCEDURE trigger_set_timestamp();
CREATE TRIGGER set_clients_timestamp BEFORE UPDATE ON clients FOR EACH ROW EXECUTE PROCEDURE trigger_set_timestamp();
CREATE TRIGGER set_technicians_timestamp BEFORE UPDATE ON technicians FOR EACH ROW EXECUTE PROCEDURE trigger_set_timestamp();