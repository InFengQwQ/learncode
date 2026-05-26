-- 005: Add knowledge_entries table, language status, and version kb_status

-- 1. Add status column to languages table
-- inactive: just created, knowledge base not yet built (default)
-- active: knowledge base ready, users can start learning
ALTER TABLE languages ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'inactive'
  CHECK (status IN ('inactive', 'active'));

-- 2. Add kb_status column to language_versions table
-- pending: not yet built (default)
-- building: currently being built
-- complete: knowledge base ready
-- failed: build failed
ALTER TABLE language_versions ADD COLUMN IF NOT EXISTS kb_status TEXT NOT NULL DEFAULT 'pending'
  CHECK (kb_status IN ('pending', 'building', 'complete', 'failed'));

-- 3. Create knowledge_entries table
CREATE TABLE IF NOT EXISTS knowledge_entries (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  language_id UUID NOT NULL REFERENCES languages(id) ON DELETE CASCADE,
  version_id UUID REFERENCES language_versions(id) ON DELETE CASCADE,
  scope TEXT NOT NULL CHECK (scope IN ('shared', 'private')),
  category TEXT NOT NULL CHECK (category IN ('factual', 'normative')),
  topic TEXT NOT NULL,
  content JSONB NOT NULL DEFAULT '{}',
  source TEXT NOT NULL DEFAULT 'llm',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_knowledge_entries_language ON knowledge_entries(language_id);
CREATE INDEX IF NOT EXISTS idx_knowledge_entries_version ON knowledge_entries(version_id) WHERE version_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_knowledge_entries_scope ON knowledge_entries(language_id, scope);