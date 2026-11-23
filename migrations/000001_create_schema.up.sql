-- PRIVEX Initial PostgreSQL Schema Migration
-- Matches Section 6 Data Model (PostgreSQL)

CREATE TABLE IF NOT EXISTS domains (
    id BIGSERIAL PRIMARY KEY,
    host TEXT NOT NULL UNIQUE,
    robots_txt TEXT,
    robots_fetched_at TIMESTAMPTZ,
    crawl_delay_ms INT DEFAULT 1000,
    status TEXT NOT NULL DEFAULT 'active',
    last_fetch_at TIMESTAMPTZ,
    page_count INT DEFAULT 0
);

CREATE TABLE IF NOT EXISTS urls (
    id BIGSERIAL PRIMARY KEY,
    domain_id BIGINT NOT NULL REFERENCES domains(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    url_hash BYTEA NOT NULL UNIQUE,
    canonical_url_id BIGINT REFERENCES urls(id),
    status TEXT NOT NULL DEFAULT 'pending', -- pending, leased, done, failed, blocked
    priority INT NOT NULL DEFAULT 0,
    depth INT NOT NULL DEFAULT 0,
    retry_count INT NOT NULL DEFAULT 0,
    lease_expires_at TIMESTAMPTZ,
    next_fetch_at TIMESTAMPTZ DEFAULT NOW(),
    etag TEXT,
    last_modified TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_urls_lease_claim 
    ON urls (priority DESC, next_fetch_at ASC) 
    WHERE status = 'pending';

CREATE TABLE IF NOT EXISTS documents (
    id BIGSERIAL PRIMARY KEY,
    url_id BIGINT NOT NULL UNIQUE REFERENCES urls(id) ON DELETE CASCADE,
    title TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    headings TEXT[] NOT NULL DEFAULT '{}',
    body TEXT NOT NULL,
    lang VARCHAR(10) NOT NULL DEFAULT 'en',
    word_count INT NOT NULL DEFAULT 0,
    content_hash BYTEA NOT NULL,
    simhash BIGINT,
    fetched_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    indexed_at TIMESTAMPTZ,
    deleted BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS links (
    id BIGSERIAL PRIMARY KEY,
    src_doc_id BIGINT NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    dst_url_id BIGINT NOT NULL REFERENCES urls(id) ON DELETE CASCADE,
    anchor_text TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_links_src ON links(src_doc_id);
CREATE INDEX IF NOT EXISTS idx_links_dst ON links(dst_url_id);

CREATE TABLE IF NOT EXISTS fetch_log (
    id BIGSERIAL PRIMARY KEY,
    url_id BIGINT NOT NULL REFERENCES urls(id) ON DELETE CASCADE,
    status_code INT NOT NULL,
    response_ms INT NOT NULL,
    bytes INT NOT NULL,
    error TEXT,
    fetched_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS index_segments (
    id BIGSERIAL PRIMARY KEY,
    path TEXT NOT NULL,
    doc_count INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    state TEXT NOT NULL DEFAULT 'building' -- building, live, merged
);

CREATE TABLE IF NOT EXISTS index_state (
    version BIGSERIAL PRIMARY KEY,
    total_docs BIGINT NOT NULL DEFAULT 0,
    avg_field_lengths JSONB NOT NULL DEFAULT '{}',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
