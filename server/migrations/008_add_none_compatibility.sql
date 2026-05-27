-- Add 'none' as a valid compatibility_model for languages without version distinctions
-- (e.g., SQL dialects, esoteric languages, domain-specific languages)
ALTER TABLE languages DROP CONSTRAINT IF EXISTS languages_compatibility_model_check;
ALTER TABLE languages ADD CONSTRAINT languages_compatibility_model_check
    CHECK (compatibility_model IN ('strict', 'versioned', 'none'));
