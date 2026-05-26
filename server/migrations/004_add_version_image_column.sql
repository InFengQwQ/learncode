-- Add image column to language_versions for storing the resolved Docker image reference.
-- This is populated during initialization.
ALTER TABLE language_versions ADD COLUMN IF NOT EXISTS image TEXT NOT NULL DEFAULT '';

-- Add initialized_at timestamp to track when the environment was verified.
ALTER TABLE language_versions ADD COLUMN IF NOT EXISTS initialized_at TIMESTAMPTZ DEFAULT NULL;