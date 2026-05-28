-- Add wiki_title column to languages table for storing the exact Wikipedia page title
-- This avoids disambiguation page issues when re-querying Wikipedia later.
ALTER TABLE languages ADD COLUMN IF NOT EXISTS wiki_title TEXT NOT NULL DEFAULT '';
