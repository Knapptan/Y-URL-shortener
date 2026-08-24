CREATE TABLE IF NOT EXISTS short_urls (
    id TEXT PRIMARY KEY,
    original_url TEXT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS unique_original_url ON short_urls (original_url);