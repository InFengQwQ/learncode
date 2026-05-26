-- Store deep research results and timestamp on the language record.
-- research_data holds the full ResearchResult JSON (docs, runtimes, specs).
-- researched_at is NULL until research completes, then set to the timestamp.

ALTER TABLE languages ADD COLUMN IF NOT EXISTS research_data JSONB DEFAULT NULL;
ALTER TABLE languages ADD COLUMN IF NOT EXISTS researched_at TIMESTAMPTZ DEFAULT NULL;