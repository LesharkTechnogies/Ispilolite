BEGIN;

CREATE TABLE IF NOT EXISTS documents (
  id UUID PRIMARY KEY,
  owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  quotation_id UUID UNIQUE REFERENCES quotations(id) ON DELETE CASCADE,
  cloudinary_public_id TEXT NOT NULL DEFAULT '',
  cloudinary_url TEXT NOT NULL DEFAULT '',
  storage_path TEXT NOT NULL DEFAULT '',
  file_name TEXT NOT NULL,
  content_type TEXT NOT NULL DEFAULT 'application/pdf',
  visibility TEXT NOT NULL DEFAULT 'PRIVATE' CHECK (visibility IN ('PRIVATE','PUBLIC')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS document_shares (
  id UUID PRIMARY KEY,
  document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  permission TEXT NOT NULL CHECK (permission IN ('VIEW','DOWNLOAD')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(document_id, user_id)
);

CREATE INDEX IF NOT EXISTS documents_owner_idx ON documents(owner_id, created_at DESC);
CREATE INDEX IF NOT EXISTS document_shares_user_idx ON document_shares(user_id, document_id);

COMMIT;
