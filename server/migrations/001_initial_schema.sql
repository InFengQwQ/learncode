CREATE TABLE languages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    compatibility_model TEXT NOT NULL CHECK (compatibility_model IN ('strict', 'versioned')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE language_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    language_id UUID NOT NULL REFERENCES languages(id) ON DELETE CASCADE,
    version TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'archived')),
    runtime_config JSONB NOT NULL DEFAULT '{}',
    source_urls JSONB NOT NULL DEFAULT '{}',
    last_version_check_at TIMESTAMPTZ,
    initialized BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(language_id, version)
);
