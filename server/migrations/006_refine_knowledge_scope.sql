-- 006: Refine knowledge scope from shared/private to core/version/idiom
-- core: language fundamentals shared across all versions (version_id = NULL)
-- version: version-specific features (version_id set)
-- idiom: cross-version best practices and conventions (version_id = NULL)

-- Drop old constraint FIRST so UPDATE can change values freely
ALTER TABLE knowledge_entries DROP CONSTRAINT IF EXISTS knowledge_entries_scope_check;

-- Now migrate data without any constraint blocking
UPDATE knowledge_entries SET scope = 'core' WHERE scope = 'shared';
UPDATE knowledge_entries SET scope = 'version' WHERE scope = 'private';

-- Finally add the new constraint
ALTER TABLE knowledge_entries ADD CONSTRAINT knowledge_entries_scope_check
  CHECK (scope IN ('core', 'version', 'idiom'));
