-- 007: Add unique constraint on knowledge_entries for upsert support
-- Allows ON CONFLICT (language_id, scope, topic) for idempotent KB builds

ALTER TABLE knowledge_entries ADD CONSTRAINT knowledge_entries_unique_topic
  UNIQUE (language_id, scope, topic);
